package gemini

import (
	"bytes"
	"container/list"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
)

const (
	geminiContextKey          = "gemini_native_tool_context"
	geminiContextMaxBytes     = 1 << 20
	geminiContextCacheBytes   = 32 << 20
	geminiContextCacheEntries = 512
	geminiContextTTL          = 30 * time.Minute
)

// Context is deliberately process-local and ephemeral. A miss leaves client
// history untouched; no history is inferred from another conversation.
var geminiContexts = newGeminiContextCache(geminiContextCacheEntries, geminiContextCacheBytes, geminiContextTTL)

type geminiContextEntry struct {
	key     [32]byte
	content []byte
	expires time.Time
}

type geminiContextCache struct {
	mu                          sync.Mutex
	entries                     map[[32]byte]*list.Element
	lru                         *list.List
	bytes, maxEntries, maxBytes int
	ttl                         time.Duration
}

func newGeminiContextCache(entries, size int, ttl time.Duration) *geminiContextCache {
	return &geminiContextCache{entries: make(map[[32]byte]*list.Element), lru: list.New(), maxEntries: entries, maxBytes: size, ttl: ttl}
}

func (s *geminiContextCache) get(key [32]byte, now time.Time) []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	e := s.entries[key]
	if e == nil {
		return nil
	}
	entry := e.Value.(geminiContextEntry)
	if !now.Before(entry.expires) {
		s.remove(e)
		return nil
	}
	entry.expires = now.Add(s.ttl)
	e.Value = entry
	s.lru.MoveToFront(e)
	return bytes.Clone(entry.content)
}

func (s *geminiContextCache) remove(e *list.Element) {
	entry := e.Value.(geminiContextEntry)
	s.bytes -= len(entry.content)
	delete(s.entries, entry.key)
	s.lru.Remove(e)
}

func (s *geminiContextCache) put(key [32]byte, content []byte, now time.Time) {
	if len(content) == 0 || len(content) > geminiContextMaxBytes || len(content) > s.maxBytes || s.maxEntries <= 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if e := s.entries[key]; e != nil {
		entry := e.Value.(geminiContextEntry)
		if now.Before(entry.expires) && !bytes.Equal(entry.content, content) {
			// Identical client-visible histories can hide different native tool
			// executions. Retain a tombstone until expiry rather than guess.
			s.bytes -= len(entry.content)
			entry.content = nil
			e.Value = entry
			return
		}
		s.remove(e)
	}
	for s.lru.Len() >= s.maxEntries || s.bytes+len(content) > s.maxBytes {
		s.remove(s.lru.Back())
	}
	s.entries[key] = s.lru.PushFront(geminiContextEntry{key: key, content: bytes.Clone(content), expires: now.Add(s.ttl)})
	s.bytes += len(content)
}

type geminiContextRestore struct {
	expected json.RawMessage
	native   json.RawMessage
}

type geminiContextCandidate struct {
	content  map[string]json.RawMessage
	parts    []json.RawMessage
	text     strings.Builder
	calls    map[int]dto.ToolCallResponse
	complete bool
}

type geminiToolContext struct {
	prefix                                  [32]byte
	userID, tokenID, channelID, channelType int
	model                                   string
	system                                  json.RawMessage
	restore                                 map[int]geminiContextRestore
	candidates                              map[int]*geminiContextCandidate
	enabled, failed                         bool
	size                                    int
	cache                                   *geminiContextCache
}

// Canonicalize JSON without converting large argument integers to float64.
// Differences in argument whitespace/key ordering must not break a round trip.
func geminiContextJSON(raw json.RawMessage, depth int) (any, error) {
	if depth > 64 {
		return nil, fmt.Errorf("context JSON nesting exceeds limit")
	}
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return nil, nil
	}
	switch raw[0] {
	case '{':
		var fields map[string]json.RawMessage
		if err := common.Unmarshal(raw, &fields); err != nil {
			return nil, err
		}
		out := make(map[string]any, len(fields))
		for k, v := range fields {
			value, err := geminiContextJSON(v, depth+1)
			if err != nil {
				return nil, err
			}
			out[k] = value
		}
		return out, nil
	case '[':
		var items []json.RawMessage
		if err := common.Unmarshal(raw, &items); err != nil {
			return nil, err
		}
		out := make([]any, len(items))
		for i, v := range items {
			value, err := geminiContextJSON(v, depth+1)
			if err != nil {
				return nil, err
			}
			out[i] = value
		}
		return out, nil
	case '"', 't', 'f', 'n':
		var value any
		err := common.Unmarshal(raw, &value)
		return value, err
	default:
		return json.Number(string(raw)), nil
	}
}

func sameGeminiContextJSON(left, right json.RawMessage) bool {
	a, err := geminiContextJSON(left, 0)
	if err != nil {
		return false
	}
	b, err := geminiContextJSON(right, 0)
	if err != nil {
		return false
	}
	encodedA, err := common.Marshal(a)
	if err != nil {
		return false
	}
	encodedB, err := common.Marshal(b)
	return err == nil && bytes.Equal(encodedA, encodedB)
}

func geminiHistoryHash(prefix [32]byte, message dto.Message) ([32]byte, bool) {
	role := message.Role
	if role == "model" {
		role = "assistant"
	}
	var content []dto.MediaContent
	for _, part := range message.ParseContent() {
		if part.Type == dto.ContentTypeText {
			if part.Text == "" {
				continue
			}
			if len(content) > 0 && content[len(content)-1].Type == dto.ContentTypeText {
				content[len(content)-1].Text += part.Text
				continue
			}
		}
		content = append(content, part)
	}
	var calls []any
	for _, call := range message.ParseToolCalls() {
		args, err := geminiContextJSON(json.RawMessage(call.Function.Arguments), 0)
		if err != nil {
			return prefix, false
		}
		calls = append(calls, struct {
			ID, Name  string
			Arguments any
		}{call.ID, call.Function.Name, args})
	}
	name := ""
	if message.Name != nil {
		name = *message.Name
	}
	data, err := common.Marshal(struct {
		Role             string
		Content          []dto.MediaContent
		ToolCalls        []any
		ToolCallID, Name string
	}{role, content, calls, message.ToolCallId, name})
	if err != nil || len(data) > geminiContextMaxBytes {
		return prefix, false
	}
	hash := sha256.New()
	hash.Write(prefix[:])
	hash.Write(data)
	var result [32]byte
	copy(result[:], hash.Sum(nil))
	return result, true
}

func newGeminiToolContext(info *relaycommon.RelayInfo, request *dto.GeneralOpenAIRequest, converted *dto.GeminiChatRequest) *geminiToolContext {
	if info == nil || info.UserId <= 0 || info.ChannelMeta == nil || info.ChannelId <= 0 || info.RelayFormat != types.RelayFormatOpenAI {
		return nil
	}
	state := &geminiToolContext{prefix: sha256.Sum256([]byte(fmt.Sprintf("%d:%d:%d:%q", info.UserId, info.TokenId, info.ChannelId, info.UpstreamModelName))), restore: make(map[int]geminiContextRestore), candidates: make(map[int]*geminiContextCandidate), cache: geminiContexts}
	state.userID, state.tokenID = info.UserId, info.TokenId
	state.channelID, state.channelType, state.model = info.ChannelId, info.ChannelType, info.UpstreamModelName
	var err error
	state.system, err = common.Marshal(converted.SystemInstructions)
	if err != nil {
		return nil
	}
	var modelIndices []int
	for i, content := range converted.Contents {
		if content.Role == "model" {
			modelIndices = append(modelIndices, i)
		}
	}
	modelIndex := 0
	for _, message := range request.Messages {
		var ok bool
		state.prefix, ok = geminiHistoryHash(state.prefix, message)
		if !ok {
			return nil
		}
		if message.Role != "assistant" && message.Role != "model" {
			continue
		}
		hasParts := len(message.ParseToolCalls()) > 0
		for _, part := range message.ParseContent() {
			hasParts = hasParts || part.Type != dto.ContentTypeText || part.Text != ""
		}
		if !hasParts {
			continue
		}
		if modelIndex >= len(modelIndices) {
			return nil
		}
		index := modelIndices[modelIndex]
		modelIndex++
		native := state.cache.get(state.prefix, time.Now())
		if native == nil {
			continue
		}
		expected, err := common.Marshal(converted.Contents[:index+1])
		if err != nil {
			return nil
		}
		state.restore[index] = geminiContextRestore{expected: expected, native: native}
	}
	if modelIndex != len(modelIndices) {
		return nil
	}
	return state
}

func geminiContextFor(c *gin.Context, info *relaycommon.RelayInfo) *geminiToolContext {
	if c == nil || info == nil || info.ChannelMeta == nil || info.RelayFormat != types.RelayFormatOpenAI {
		return nil
	}
	value, _ := c.Get(geminiContextKey)
	state, _ := value.(*geminiToolContext)
	// A retry can reuse gin.Context without invoking this adaptor again
	// (for example when the next channel uses the shared Vertex handler).
	if state == nil || state.userID != info.UserId || state.tokenID != info.TokenId ||
		state.channelID != info.ChannelId || state.channelType != info.ChannelType || state.model != info.UpstreamModelName {
		return nil
	}
	return state
}

// The hook sees tools inserted by parameter overrides. Raw JSON avoids dropping
// provider fields that the gateway's typed DTOs do not yet know about.
func (a *Adaptor) PreparePostOverrideRequest(body []byte) ([]byte, error) {
	if !strings.HasPrefix(a.model, "gemini-3.") && !strings.HasPrefix(a.model, "gemini-3-") {
		return body, nil
	}
	var request map[string]json.RawMessage
	if err := common.Unmarshal(body, &request); err != nil {
		return nil, err
	}
	var contents []json.RawMessage
	if err := common.Unmarshal(request["contents"], &contents); err != nil {
		return nil, err
	}
	restored := false
	if a.toolContext != nil {
		originalContents := append([]json.RawMessage(nil), contents...)
		for index, saved := range a.toolContext.restore {
			if index >= len(contents) {
				continue
			}
			currentPrefix, err := common.Marshal(originalContents[:index+1])
			if err != nil {
				return nil, err
			}
			// An explicit override of any preceding content takes precedence.
			if !sameGeminiContextJSON(currentPrefix, saved.expected) ||
				!sameGeminiContextJSON(request["systemInstruction"], a.toolContext.system) {
				continue
			}
			contents[index] = saved.native
			restored = true
		}
	}
	var tools []map[string]json.RawMessage
	if raw := bytes.TrimSpace(request["tools"]); len(raw) > 0 && string(raw) != "null" {
		if raw[0] == '{' {
			var tool map[string]json.RawMessage
			if err := common.Unmarshal(raw, &tool); err != nil {
				return nil, err
			}
			tools = append(tools, tool)
		} else if err := common.Unmarshal(raw, &tools); err != nil {
			return nil, err
		}
	}
	hasFunctions, hasBuiltin := false, false
	for _, tool := range tools {
		var declarations []json.RawMessage
		if raw := tool["functionDeclarations"]; len(raw) > 0 {
			if err := common.Unmarshal(raw, &declarations); err != nil {
				return nil, err
			}
		}
		hasFunctions = hasFunctions || len(declarations) > 0
		for _, name := range []string{"googleSearch", "googleSearchRetrieval", "urlContext", "codeExecution", "googleMaps", "enterpriseWebSearch", "fileSearch"} {
			if raw := bytes.TrimSpace(tool[name]); len(raw) > 0 && string(raw) != "null" {
				hasBuiltin = true
			}
		}
	}
	config := make(map[string]json.RawMessage)
	if raw := request["toolConfig"]; len(raw) > 0 && string(raw) != "null" {
		if err := common.Unmarshal(raw, &config); err != nil {
			return nil, err
		}
	}
	enabled := hasFunctions && hasBuiltin || restored || string(config["includeServerSideToolInvocations"]) == "true"
	if a.toolContext != nil {
		a.toolContext.enabled = enabled
	}
	if !enabled {
		return body, nil
	}
	config["includeServerSideToolInvocations"] = json.RawMessage("true")
	calling := make(map[string]json.RawMessage)
	if raw := config["functionCallingConfig"]; len(raw) > 0 && string(raw) != "null" {
		if err := common.Unmarshal(raw, &calling); err != nil {
			return nil, err
		}
	}
	var mode string
	if raw := calling["mode"]; len(raw) > 0 {
		if err := common.Unmarshal(raw, &mode); err != nil {
			return nil, err
		}
	}
	if mode == "" || mode == "AUTO" {
		calling["mode"] = json.RawMessage(`"VALIDATED"`)
	}
	var err error
	config["functionCallingConfig"], err = common.Marshal(calling)
	if err != nil {
		return nil, err
	}
	request["toolConfig"], err = common.Marshal(config)
	if err != nil {
		return nil, err
	}
	if restored {
		request["contents"], err = common.Marshal(contents)
		if err != nil {
			return nil, err
		}
	}
	return common.Marshal(request)
}

// Observe the actual SSE bytes after OpenAI formatting (including
// ThinkingToContent), so cached history matches what the client sends back.
type geminiContextWriter struct {
	gin.ResponseWriter
	state   *geminiToolContext
	pending []byte
	size    int
}

func (w *geminiContextWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *geminiContextWriter) Write(data []byte) (int, error) {
	n, err := w.ResponseWriter.Write(data)
	if err != nil {
		w.state.failed = true
	}
	w.observe(data[:n])
	return n, err
}

func (w *geminiContextWriter) WriteString(data string) (int, error) {
	n, err := w.ResponseWriter.WriteString(data)
	if err != nil {
		w.state.failed = true
	}
	w.observe([]byte(data[:n]))
	return n, err
}

func (w *geminiContextWriter) observe(data []byte) {
	if !w.state.enabled || w.state.failed {
		w.pending = nil
		return
	}
	if len(data) > geminiContextMaxBytes-w.size {
		w.state.failed = true
		w.state.candidates = nil
		w.pending = nil
		return
	}
	w.size += len(data)
	w.pending = append(w.pending, data...)
	for {
		index := bytes.IndexByte(w.pending, '\n')
		if index < 0 {
			return
		}
		line := bytes.TrimSuffix(w.pending[:index], []byte{'\r'})
		if payload, ok := bytes.CutPrefix(line, []byte("data: ")); ok && !bytes.Equal(payload, []byte("[DONE]")) {
			var response dto.ChatCompletionsStreamResponse
			if common.Unmarshal(payload, &response) != nil {
				w.state.failed = true
				w.pending = nil
				return
			}
			w.state.project(&response)
		}
		w.pending = w.pending[index+1:]
		if len(w.pending) == 0 {
			w.pending = nil
			return
		}
	}
}

func (s *geminiToolContext) capture(raw []byte) {
	if s == nil || !s.enabled || s.failed {
		return
	}
	if len(raw) > geminiContextMaxBytes-s.size {
		s.failed = true
		s.candidates = nil
		return
	}
	s.size += len(raw)
	var response struct {
		Candidates []struct {
			Index        int                        `json:"index"`
			Content      map[string]json.RawMessage `json:"content"`
			FinishReason string                     `json:"finishReason"`
		} `json:"candidates"`
	}
	if common.Unmarshal(raw, &response) != nil {
		s.failed = true
		s.candidates = nil
		return
	}
	for _, candidate := range response.Candidates {
		state := s.candidates[candidate.Index]
		if state == nil {
			state = &geminiContextCandidate{content: make(map[string]json.RawMessage), calls: make(map[int]dto.ToolCallResponse)}
			s.candidates[candidate.Index] = state
		}
		for key, value := range candidate.Content {
			if key != "parts" {
				state.content[key] = value
			}
		}
		var parts []json.RawMessage
		if rawParts := candidate.Content["parts"]; len(rawParts) > 0 {
			if common.Unmarshal(rawParts, &parts) != nil {
				s.failed = true
				s.candidates = nil
				return
			}
		}
		state.parts = append(state.parts, parts...)
		state.complete = state.complete || candidate.FinishReason == "STOP"
	}
}

func (s *geminiToolContext) project(response *dto.ChatCompletionsStreamResponse) {
	if s == nil || !s.enabled || s.failed {
		return
	}
	for _, choice := range response.Choices {
		state := s.candidates[choice.Index]
		if state == nil {
			continue
		}
		if choice.Delta.Content != nil {
			state.text.WriteString(*choice.Delta.Content)
		}
		for _, tool := range choice.Delta.ToolCalls {
			index := 0
			if tool.Index != nil {
				index = *tool.Index
			}
			call := state.calls[index]
			if tool.ID != "" {
				call.ID = tool.ID
			}
			if tool.Type != nil {
				call.Type = tool.Type
			}
			if tool.Function.Name != "" {
				call.Function.Name = tool.Function.Name
			}
			call.Function.Arguments += tool.Function.Arguments
			state.calls[index] = call
		}
	}
}

func (s *geminiToolContext) save() {
	if s == nil || !s.enabled || s.failed {
		return
	}
	for _, state := range s.candidates {
		if !state.complete || len(state.parts) == 0 {
			continue
		}
		message := dto.Message{Role: "assistant", Content: state.text.String()}
		calls := make([]dto.ToolCallResponse, 0, len(state.calls))
		for i := range len(state.calls) {
			call, ok := state.calls[i]
			if !ok {
				s.failed = true
				return
			}
			calls = append(calls, call)
		}
		if len(calls) > 0 {
			message.SetToolCalls(calls)
		}
		key, ok := geminiHistoryHash(s.prefix, message)
		if !ok {
			continue
		}
		parts, err := common.Marshal(state.parts)
		if err != nil {
			continue
		}
		state.content["parts"] = parts
		if len(state.content["role"]) == 0 {
			state.content["role"] = json.RawMessage(`"model"`)
		}
		content, err := common.Marshal(state.content)
		if err != nil {
			continue
		}
		s.cache.put(key, content, time.Now())
	}
}
