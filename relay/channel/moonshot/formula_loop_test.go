package moonshot

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	channelconstant "github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type observedKimiRequest struct {
	method string
	path   string
	header http.Header
	body   []byte
}

type trackedReadCloser struct {
	io.Reader
	closed atomic.Bool
}

type recordingBilling struct {
	mu      sync.Mutex
	targets []int
	current int
	failAt  int
}

func (b *recordingBilling) Settle(int) error    { return nil }
func (b *recordingBilling) Refund(*gin.Context) {}
func (b *recordingBilling) NeedsRefund() bool   { return false }
func (b *recordingBilling) GetPreConsumedQuota() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.current
}
func (b *recordingBilling) Reserve(target int) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.targets = append(b.targets, target)
	if b.failAt > 0 && len(b.targets) >= b.failAt {
		return errors.New("insufficient test quota")
	}
	if target > b.current {
		b.current = target
	}
	return nil
}
func (b *recordingBilling) reserveCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.targets)
}

func (r *trackedReadCloser) Close() error {
	r.closed.Store(true)
	return nil
}

func newKimiTestContext(t *testing.T, requestContext context.Context) (*gin.Context, *trackedReadCloser) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	outerBody := &trackedReadCloser{Reader: bytes.NewReader([]byte("client body"))}
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", outerBody)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Client-Trace", "trace-123")
	if requestContext != nil {
		request = request.WithContext(requestContext)
	}
	c.Request = request
	return c, outerBody
}

func newKimiRelayInfo(baseURL string, stream bool) *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		IsStream:        stream,
		RelayMode:       relayconstant.RelayModeChatCompletions,
		RelayFormat:     types.RelayFormatOpenAI,
		OriginModelName: "kimi-k3",
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       channelconstant.ChannelTypeMoonshot,
			ChannelBaseUrl:    baseURL,
			ApiKey:            "test-channel-key",
			UpstreamModelName: "kimi-k3",
			HeadersOverride: map[string]any{
				"X-Managed-Loop": "enabled",
			},
		},
	}
}

func readObservedRequest(t *testing.T, r *http.Request) observedKimiRequest {
	t.Helper()
	body, err := io.ReadAll(r.Body)
	require.NoError(t, err)
	return observedKimiRequest{
		method: r.Method,
		path:   r.URL.Path,
		header: r.Header.Clone(),
		body:   body,
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, value string) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	_, err := io.WriteString(w, value)
	require.NoError(t, err)
}

func doPreparedKimiRequest(adaptor *Adaptor, c *gin.Context, info *relaycommon.RelayInfo, body io.Reader) (any, error) {
	requestBody, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	preparedBody, err := adaptor.PreparePostOverrideRequest(requestBody)
	if err != nil {
		return nil, err
	}
	return adaptor.DoRequest(c, info, bytes.NewReader(preparedBody))
}

func disableKimiTestPersistence(t *testing.T) {
	t.Helper()
	originalBatchUpdate := common.BatchUpdateEnabled
	originalLogConsume := common.LogConsumeEnabled
	common.BatchUpdateEnabled = true
	common.LogConsumeEnabled = false
	t.Cleanup(func() {
		common.BatchUpdateEnabled = originalBatchUpdate
		common.LogConsumeEnabled = originalLogConsume
	})
}

func setupKimiConsumeLogTest(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:kimi-upstream-request-id?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Log{}))

	originalDB := model.DB
	originalLogDB := model.LOG_DB
	originalBatchUpdate := common.BatchUpdateEnabled
	originalLogConsume := common.LogConsumeEnabled
	originalDataExport := common.DataExportEnabled
	originalRedisEnabled := common.RedisEnabled
	model.DB = db
	model.LOG_DB = db
	common.BatchUpdateEnabled = true
	common.LogConsumeEnabled = true
	common.DataExportEnabled = false
	common.RedisEnabled = false
	t.Cleanup(func() {
		model.DB = originalDB
		model.LOG_DB = originalLogDB
		common.BatchUpdateEnabled = originalBatchUpdate
		common.LogConsumeEnabled = originalLogConsume
		common.DataExportEnabled = originalDataExport
		common.RedisEnabled = originalRedisEnabled
	})
	return db
}

func TestKimiFormulaLoopExecutesAllManagedToolsAndAggregatesUsage(t *testing.T) {
	service.InitHttpClient()

	var mu sync.Mutex
	var observed []observedKimiRequest
	chatRound := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		request := readObservedRequest(t, r)
		mu.Lock()
		observed = append(observed, request)
		mu.Unlock()

		assert.Equal(t, "Bearer test-channel-key", r.Header.Get("Authorization"))
		assert.Equal(t, "enabled", r.Header.Get("X-Managed-Loop"))
		switch r.URL.Path {
		case "/v1/formulas/moonshot/web-search:latest/tools":
			assert.Equal(t, http.MethodGet, r.Method)
			writeJSON(t, w, `{"tools":[{"type":"function","function":{"name":"web_search","description":"Search","parameters":{"type":"object"}}}]}`)
		case "/v1/formulas/moonshot/fetch:latest/tools":
			assert.Equal(t, http.MethodGet, r.Method)
			writeJSON(t, w, `{"tools":[{"type":"function","function":{"name":"read_web","description":"Fetch","parameters":{"type":"object"}}}]}`)
		case "/v1/formulas/moonshot/code-runner:latest/tools":
			assert.Equal(t, http.MethodGet, r.Method)
			writeJSON(t, w, `{"tools":[{"type":"function","function":{"name":"run_python","description":"Run","parameters":{"type":"object"}}}]}`)
		case "/v1/formulas/moonshot/web-search:latest/fibers":
			writeJSON(t, w, `{"status":"succeeded","context":{"encrypted_output":"cipher-token"}}`)
		case "/v1/formulas/moonshot/fetch:latest/fibers":
			writeJSON(t, w, `{"status":"succeeded","context":{"output":{"title":"example"}}}`)
		case "/v1/formulas/moonshot/code-runner:latest/fibers":
			writeJSON(t, w, `{"status":"succeeded","context":{"output":false}}`)
		case "/v1/chat/completions":
			assert.Equal(t, http.MethodPost, r.Method)
			chatRound++
			if chatRound == 1 {
				writeJSON(t, w, `{"id":"chatcmpl-round-1","object":"chat.completion","created":1,"model":"kimi-k3","choices":[{"index":0,"message":{"role":"assistant","content":null,"reasoning_content":"full private reasoning","vendor_extension":{"keep":true},"tool_calls":[{"id":"call-search","type":"function","function":{"name":"web_search","arguments":"{\"query\": \"k3\"}"}},{"id":"call-fetch","type":"function","function":{"name":"read_web","arguments":"{\"url\":\"https://example.com\"}"}},{"id":"call-code","type":"function","function":{"name":"run_python","arguments":"{\"code\":\"print(3)\"}"}}]},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":80,"completion_tokens":10,"total_tokens":90,"cached_tokens":20,"completion_tokens_details":{"reasoning_tokens":4}}}`)
				return
			}
			writeJSON(t, w, `{"id":"chatcmpl-final","object":"chat.completion","created":2,"model":"kimi-k3","choices":[{"index":0,"message":{"role":"assistant","content":"done","reasoning_content":"final reasoning"},"finish_reason":"stop"}],"usage":{"prompt_tokens":150,"completion_tokens":20,"total_tokens":170,"cached_tokens":999,"prompt_tokens_details":{"cached_tokens":40},"completion_tokens_details":{"reasoning_tokens":3}}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	c, outerBody := newKimiTestContext(t, context.Background())
	info := newKimiRelayInfo(server.URL, false)
	body := bytes.NewBufferString(`{"model":"kimi-k3","messages":[{"role":"user","content":"research"}],"tools":[{"type":"function","function":{"name":"client_fn","parameters":{"type":"object"}}}],"kimi_tools":["web-search","fetch","code-runner"]}`)

	result, err := doPreparedKimiRequest(&Adaptor{}, c, info, body)
	require.NoError(t, err)
	resp, ok := result.(*http.Response)
	require.True(t, ok)
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var response map[string]any
	require.NoError(t, common.Unmarshal(responseBody, &response))
	usage := response["usage"].(map[string]any)
	assert.EqualValues(t, 230, usage["prompt_tokens"])
	assert.EqualValues(t, 30, usage["completion_tokens"])
	assert.EqualValues(t, 260, usage["total_tokens"])
	assert.EqualValues(t, 60, usage["cached_tokens"])
	promptDetails, ok := usage["prompt_tokens_details"].(map[string]any)
	require.True(t, ok)
	assert.EqualValues(t, 60, promptDetails["cached_tokens"])
	completionDetails, ok := usage["completion_tokens_details"].(map[string]any)
	require.True(t, ok)
	assert.EqualValues(t, 7, completionDetails["reasoning_tokens"])
	assert.False(t, outerBody.closed.Load(), "internal calls must not close the client request body")
	require.NotNil(t, info.KimiToolLoop)
	assert.True(t, info.KimiToolLoop.Completed)
	assert.Empty(t, info.KimiToolLoop.ErrorCode)
	assert.Len(t, info.KimiToolLoop.Usages, 2)
	assert.Equal(t, map[string]int{"web-search": 1, "fetch": 1, "code-runner": 1}, info.KimiToolLoop.ToolCalls)
	assert.Equal(t, map[string]int{"web-search": 1, "fetch": 1, "code-runner": 1}, info.KimiToolLoop.AttemptedToolCalls)

	mu.Lock()
	requests := append([]observedKimiRequest(nil), observed...)
	mu.Unlock()
	var chats []observedKimiRequest
	var fibers []observedKimiRequest
	for _, request := range requests {
		switch request.path {
		case "/v1/chat/completions":
			chats = append(chats, request)
		case "/v1/formulas/moonshot/web-search:latest/fibers", "/v1/formulas/moonshot/fetch:latest/fibers", "/v1/formulas/moonshot/code-runner:latest/fibers":
			fibers = append(fibers, request)
		}
	}
	require.Len(t, chats, 2)
	require.Len(t, fibers, 3)

	var firstChat map[string]any
	require.NoError(t, common.Unmarshal(chats[0].body, &firstChat))
	_, markerForwarded := firstChat["kimi_tools"]
	assert.False(t, markerForwarded)
	assert.Equal(t, false, firstChat["stream"])
	assert.Len(t, firstChat["tools"], 4)

	var secondChat map[string]any
	require.NoError(t, common.Unmarshal(chats[1].body, &secondChat))
	messages := secondChat["messages"].([]any)
	require.Len(t, messages, 5)
	assistant := messages[1].(map[string]any)
	assert.Equal(t, "full private reasoning", assistant["reasoning_content"])
	assert.Equal(t, true, assistant["vendor_extension"].(map[string]any)["keep"])
	assert.Equal(t, "cipher-token", messages[2].(map[string]any)["content"])
	assert.JSONEq(t, `{"title":"example"}`, messages[3].(map[string]any)["content"].(string))
	assert.Equal(t, "false", messages[4].(map[string]any)["content"])

	wantFiberBodies := map[string]string{
		"/v1/formulas/moonshot/web-search:latest/fibers":  `{"name":"web_search","arguments":"{\"query\": \"k3\"}"}`,
		"/v1/formulas/moonshot/fetch:latest/fibers":       `{"name":"read_web","arguments":"{\"url\":\"https://example.com\"}"}`,
		"/v1/formulas/moonshot/code-runner:latest/fibers": `{"name":"run_python","arguments":"{\"code\":\"print(3)\"}"}`,
	}
	for _, fiber := range fibers {
		assert.JSONEq(t, wantFiberBodies[fiber.path], string(fiber.body))
	}
}

func TestKimiFormulaLoopEmptyMarkerUsesOrdinaryChatAndRemovesMarker(t *testing.T) {
	service.InitHttpClient()
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		assert.Equal(t, "/v1/chat/completions", r.URL.Path)
		observed := readObservedRequest(t, r)
		var body map[string]any
		require.NoError(t, common.Unmarshal(observed.body, &body))
		_, markerForwarded := body["kimi_tools"]
		assert.False(t, markerForwarded)
		writeJSON(t, w, `{"choices":[{"index":0,"message":{"role":"assistant","content":"ordinary"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`)
	}))
	defer server.Close()

	c, outerBody := newKimiTestContext(t, context.Background())
	info := newKimiRelayInfo(server.URL, false)
	result, err := doPreparedKimiRequest(&Adaptor{}, c, info, bytes.NewBufferString(`{"model":"kimi-k3","messages":[],"kimi_\u0074ools":[]}`))
	require.NoError(t, err)
	resp := result.(*http.Response)
	defer resp.Body.Close()
	assert.Equal(t, int32(1), requests.Load())
	assert.True(t, outerBody.closed.Load())
	assert.Nil(t, info.KimiToolLoop)
}

func TestKimiFormulaLoopRejectsClientToolCollisionBeforeChat(t *testing.T) {
	service.InitHttpClient()
	var chats atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/formulas/moonshot/web-search:latest/tools":
			writeJSON(t, w, `{"tools":[{"type":"function","function":{"name":"same_name","parameters":{"type":"object"}}}]}`)
		case "/v1/chat/completions":
			chats.Add(1)
			writeJSON(t, w, `{}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	c, _ := newKimiTestContext(t, context.Background())
	body := bytes.NewBufferString(`{"model":"kimi-k3","messages":[],"tools":[{"type":"function","function":{"name":"same_name","parameters":{"type":"object"}}}],"kimi_tools":["web-search"]}`)
	info := newKimiRelayInfo(server.URL, false)
	_, err := doPreparedKimiRequest(&Adaptor{}, c, info, body)
	require.Error(t, err)
	var apiErr *types.NewAPIError
	require.True(t, errors.As(err, &apiErr))
	assert.True(t, types.IsSkipRetryError(apiErr))
	assert.Zero(t, chats.Load())
	require.NotNil(t, info.KimiToolLoop)
	assert.False(t, info.KimiToolLoop.Completed)
	assert.Equal(t, "kimi_tool_loop_tool_collision", info.KimiToolLoop.ErrorCode)
}

func TestKimiFormulaLoopReturnsMixedToolBatchWithoutPartialExecution(t *testing.T) {
	service.InitHttpClient()
	var fibers atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/formulas/moonshot/web-search:latest/tools":
			writeJSON(t, w, `{"tools":[{"type":"function","function":{"name":"web_search","parameters":{"type":"object"}}}]}`)
		case "/v1/chat/completions":
			writeJSON(t, w, `{"id":"mixed","object":"chat.completion","choices":[{"index":0,"message":{"role":"assistant","content":null,"tool_calls":[{"id":"managed","type":"function","function":{"name":"web_search","arguments":"{}"}},{"id":"client","type":"function","function":{"name":"client_fn","arguments":"{}"}}]},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":5,"completion_tokens":2,"total_tokens":7}}`)
		case "/v1/formulas/moonshot/web-search:latest/fibers":
			fibers.Add(1)
			writeJSON(t, w, `{"status":"succeeded","context":{"output":"should not run"}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	c, _ := newKimiTestContext(t, context.Background())
	body := bytes.NewBufferString(`{"model":"kimi-k3","messages":[],"tools":[{"type":"function","function":{"name":"client_fn","parameters":{"type":"object"}}}],"kimi_tools":["web-search"]}`)
	info := newKimiRelayInfo(server.URL, false)
	result, err := doPreparedKimiRequest(&Adaptor{}, c, info, body)
	require.NoError(t, err)
	resp := result.(*http.Response)
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(responseBody), `"name":"web_search"`)
	assert.Contains(t, string(responseBody), `"name":"client_fn"`)
	assert.Zero(t, fibers.Load())
	require.NotNil(t, info.KimiToolLoop)
	assert.True(t, info.KimiToolLoop.Completed)
	assert.Len(t, info.KimiToolLoop.Usages, 1)
	assert.Empty(t, info.KimiToolLoop.ToolCalls)
}

func TestKimiFormulaLoopSanitizesFiberFailureAndStopsRetries(t *testing.T) {
	service.InitHttpClient()
	db := setupKimiConsumeLogTest(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/formulas/moonshot/web-search:latest/tools":
			writeJSON(t, w, `{"tools":[{"type":"function","function":{"name":"web_search","parameters":{"type":"object"}}}]}`)
		case "/v1/chat/completions":
			writeJSON(t, w, `{"choices":[{"index":0,"message":{"role":"assistant","content":null,"tool_calls":[{"id":"call-1","type":"function","function":{"name":"web_search","arguments":"{}"}}]},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":5,"completion_tokens":2,"total_tokens":7}}`)
		case "/v1/formulas/moonshot/web-search:latest/fibers":
			w.Header().Set(common.RequestIdKey, "fiber-partial-request-id")
			writeJSON(t, w, `{"status":"failed","error":"secret-provider-detail","context":{"error":"also-secret"}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	c, _ := newKimiTestContext(t, context.Background())
	info := newKimiRelayInfo(server.URL, false)
	info.Billing = &recordingBilling{}
	info.UserId = 1
	info.TokenId = 1
	info.ChannelId = 1
	info.UserQuota = int(^uint(0) >> 2)
	_, err := doPreparedKimiRequest(&Adaptor{}, c, info, bytes.NewBufferString(`{"model":"kimi-k3","messages":[],"kimi_tools":["web-search"]}`))
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "secret-provider-detail")
	assert.NotContains(t, err.Error(), "also-secret")
	var apiErr *types.NewAPIError
	require.True(t, errors.As(err, &apiErr))
	assert.True(t, types.IsSkipRetryError(apiErr))
	require.NotNil(t, info.KimiToolLoop)
	assert.False(t, info.KimiToolLoop.Completed)
	assert.Equal(t, map[string]int{"web-search": 1}, info.KimiToolLoop.AttemptedToolCalls)
	assert.Empty(t, info.KimiToolLoop.ToolCalls)
	assert.Equal(t, "fiber-partial-request-id", c.GetString(common.UpstreamRequestIdKey))
	var consumeLog model.Log
	require.NoError(t, db.Where("type = ?", model.LogTypeConsume).Order("id desc").First(&consumeLog).Error)
	assert.Equal(t, "fiber-partial-request-id", consumeLog.UpstreamRequestId)
}

func TestKimiFormulaLoopFormulaFetchUsesClientCancellation(t *testing.T) {
	service.InitHttpClient()
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		select {
		case <-r.Context().Done():
		case <-time.After(3 * time.Second):
			w.WriteHeader(http.StatusGatewayTimeout)
		}
	}))
	defer server.Close()

	requestContext, cancel := context.WithCancel(context.Background())
	c, _ := newKimiTestContext(t, requestContext)
	result := make(chan error, 1)
	go func() {
		_, err := doPreparedKimiRequest(&Adaptor{}, c, newKimiRelayInfo(server.URL, false), bytes.NewBufferString(`{"model":"kimi-k3","messages":[],"kimi_tools":["web-search"]}`))
		result <- err
	}()

	select {
	case <-started:
		cancel()
	case <-time.After(2 * time.Second):
		t.Fatal("formula declaration request did not start")
	}

	select {
	case err := <-result:
		require.Error(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("formula declaration request ignored client cancellation")
	}
}

func TestKimiFormulaLoopReservesExpandedInitialChatBeforeSending(t *testing.T) {
	service.InitHttpClient()
	billing := &recordingBilling{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/formulas/moonshot/web-search:latest/tools":
			writeJSON(t, w, `{"tools":[{"type":"function","function":{"name":"web_search","description":"large declaration text","parameters":{"type":"object"}}}]}`)
		case "/v1/chat/completions":
			assert.Greater(t, billing.reserveCount(), 0, "expanded initial request must be reserved before Chat")
			writeJSON(t, w, `{"choices":[{"index":0,"message":{"role":"assistant","content":"done"},"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"completion_tokens":2,"total_tokens":7}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	c, _ := newKimiTestContext(t, context.Background())
	info := newKimiRelayInfo(server.URL, false)
	info.Billing = billing
	info.PriceData.ModelRatio = 1
	info.PriceData.CompletionRatio = 1
	info.PriceData.GroupRatioInfo.GroupRatio = 1
	result, err := doPreparedKimiRequest(&Adaptor{}, c, info, bytes.NewBufferString(`{"model":"kimi-k3","messages":[{"role":"user","content":"hello"}],"max_tokens":20,"kimi_tools":["web-search"]}`))
	require.NoError(t, err)
	defer result.(*http.Response).Body.Close()
}

func TestKimiFormulaLoopQuotaFailurePreventsFiberRequest(t *testing.T) {
	service.InitHttpClient()
	disableKimiTestPersistence(t)

	billing := &recordingBilling{failAt: 2}
	var chats atomic.Int32
	var fibers atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/formulas/moonshot/web-search:latest/tools":
			writeJSON(t, w, `{"tools":[{"type":"function","function":{"name":"web_search","parameters":{"type":"object"}}}]}`)
		case "/v1/chat/completions":
			chats.Add(1)
			writeJSON(t, w, `{"choices":[{"index":0,"message":{"role":"assistant","content":null,"tool_calls":[{"id":"call-1","type":"function","function":{"name":"web_search","arguments":"{}"}}]},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":5,"completion_tokens":2,"total_tokens":7}}`)
		case "/v1/formulas/moonshot/web-search:latest/fibers":
			fibers.Add(1)
			writeJSON(t, w, `{"status":"succeeded","context":{"output":"unexpected"}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	c, _ := newKimiTestContext(t, context.Background())
	info := newKimiRelayInfo(server.URL, false)
	info.Billing = billing
	info.UserQuota = int(^uint(0) >> 2)
	info.PriceData.ModelRatio = 1
	info.PriceData.CompletionRatio = 1
	info.PriceData.GroupRatioInfo.GroupRatio = 1
	_, err := doPreparedKimiRequest(&Adaptor{}, c, info, bytes.NewBufferString(`{"model":"kimi-k3","messages":[],"kimi_tools":["web-search"]}`))
	require.Error(t, err)
	var apiErr *types.NewAPIError
	require.True(t, errors.As(err, &apiErr))
	assert.True(t, types.IsSkipRetryError(apiErr))
	assert.EqualValues(t, 1, chats.Load())
	assert.Zero(t, fibers.Load())
	assert.Zero(t, info.KimiToolLoop.AttemptedToolCalls["web-search"], "failed reservation must roll back the planned attempt")
}

func TestKimiFormulaLoopSSEFinalPassesThroughOpenAIResponseHandler(t *testing.T) {
	service.InitHttpClient()
	originalStreamingTimeout := channelconstant.StreamingTimeout
	channelconstant.StreamingTimeout = 30
	t.Cleanup(func() { channelconstant.StreamingTimeout = originalStreamingTimeout })
	var fibers atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/formulas/moonshot/web-search:latest/tools":
			writeJSON(t, w, `{"tools":[{"type":"function","function":{"name":"web_search","parameters":{"type":"object"}}}]}`)
		case "/v1/chat/completions":
			writeJSON(t, w, `{"id":"mixed-stream","object":"chat.completion","created":5,"model":"kimi-k3","choices":[{"index":0,"message":{"role":"assistant","content":null,"tool_calls":[{"id":"managed","type":"function","function":{"name":"web_search","arguments":"{}"}},{"id":"client","type":"function","function":{"name":"client_fn","arguments":"{}"}}]},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":9,"completion_tokens":4,"total_tokens":13,"cached_tokens":2}}`)
		case "/v1/formulas/moonshot/web-search:latest/fibers":
			fibers.Add(1)
			writeJSON(t, w, `{"status":"succeeded","context":{"output":"unexpected"}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	outerBody := &trackedReadCloser{Reader: bytes.NewReader([]byte("outer"))}
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", outerBody)
	c.Request.Header.Set("Content-Type", "application/json")
	info := newKimiRelayInfo(server.URL, true)
	info.ShouldIncludeUsage = false
	adaptor := &Adaptor{}
	result, err := doPreparedKimiRequest(adaptor, c, info, bytes.NewBufferString(`{"model":"kimi-k3","stream":true,"messages":[],"tools":[{"type":"function","function":{"name":"client_fn","parameters":{"type":"object"}}}],"kimi_tools":["web-search"]}`))
	require.NoError(t, err)
	usage, apiErr := adaptor.DoResponse(c, result.(*http.Response), info)
	require.Nil(t, apiErr)
	require.NotNil(t, usage)
	assert.Equal(t, 9, usage.(*dto.Usage).PromptTokens)
	assert.Equal(t, 4, usage.(*dto.Usage).CompletionTokens)
	assert.Zero(t, fibers.Load())
	assert.False(t, outerBody.closed.Load())
	streamBody := recorder.Body.String()
	assert.Contains(t, streamBody, "data: [DONE]")
	firstFrame := strings.TrimPrefix(strings.SplitN(streamBody, "\n\n", 2)[0], "data: ")
	var streamPayload map[string]any
	require.NoError(t, common.Unmarshal([]byte(firstFrame), &streamPayload))
	choices := streamPayload["choices"].([]any)
	delta := choices[0].(map[string]any)["delta"].(map[string]any)
	toolCalls := delta["tool_calls"].([]any)
	assert.EqualValues(t, 0, toolCalls[0].(map[string]any)["index"])
	assert.EqualValues(t, 1, toolCalls[1].(map[string]any)["index"])
}

func TestKimiFormulaLoopDoesNotExecuteMalformedToolBatch(t *testing.T) {
	service.InitHttpClient()
	disableKimiTestPersistence(t)
	var fibers atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/formulas/moonshot/web-search:latest/tools":
			writeJSON(t, w, `{"tools":[{"type":"function","function":{"name":"web_search","parameters":{"type":"object"}}}]}`)
		case "/v1/chat/completions":
			writeJSON(t, w, `{"choices":[{"index":0,"message":{"role":"assistant","content":null,"tool_calls":[{"id":"call-1","type":"function","function":{"name":"web_search","arguments":"{}"}}]},"finish_reason":"length"}],"usage":{"prompt_tokens":5,"completion_tokens":2,"total_tokens":7}}`)
		case "/v1/formulas/moonshot/web-search:latest/fibers":
			fibers.Add(1)
			writeJSON(t, w, `{"status":"succeeded","context":{"output":"unexpected"}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	c, _ := newKimiTestContext(t, context.Background())
	_, err := doPreparedKimiRequest(&Adaptor{}, c, newKimiRelayInfo(server.URL, false), bytes.NewBufferString(`{"model":"kimi-k3","messages":[],"kimi_tools":["web-search"]}`))
	require.Error(t, err)
	assert.Zero(t, fibers.Load())
}

func TestKimiFormulaLoopCountsSucceededFiberWithMissingOutput(t *testing.T) {
	service.InitHttpClient()
	disableKimiTestPersistence(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/formulas/moonshot/web-search:latest/tools":
			writeJSON(t, w, `{"tools":[{"type":"function","function":{"name":"web_search","parameters":{"type":"object"}}}]}`)
		case "/v1/chat/completions":
			writeJSON(t, w, `{"choices":[{"index":0,"message":{"role":"assistant","content":null,"tool_calls":[{"id":"call-1","type":"function","function":{"name":"web_search","arguments":"{}"}}]},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":5,"completion_tokens":2,"total_tokens":7}}`)
		case "/v1/formulas/moonshot/web-search:latest/fibers":
			writeJSON(t, w, `{"status":"succeeded","context":{}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	c, _ := newKimiTestContext(t, context.Background())
	info := newKimiRelayInfo(server.URL, false)
	_, err := doPreparedKimiRequest(&Adaptor{}, c, info, bytes.NewBufferString(`{"model":"kimi-k3","messages":[],"kimi_tools":["web-search"]}`))
	require.Error(t, err)
	require.NotNil(t, info.KimiToolLoop)
	assert.Equal(t, map[string]int{"web-search": 1}, info.KimiToolLoop.AttemptedToolCalls)
	assert.Equal(t, map[string]int{"web-search": 1}, info.KimiToolLoop.ToolCalls)
}

func TestKimiFormulaLoopRejectsNegativeUsageBeforeBilling(t *testing.T) {
	service.InitHttpClient()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/formulas/moonshot/web-search:latest/tools":
			writeJSON(t, w, `{"tools":[{"type":"function","function":{"name":"web_search","parameters":{"type":"object"}}}]}`)
		case "/v1/chat/completions":
			writeJSON(t, w, `{"choices":[{"index":0,"message":{"role":"assistant","content":"bad usage"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":2,"total_tokens":3,"cached_tokens":-1}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	c, _ := newKimiTestContext(t, context.Background())
	info := newKimiRelayInfo(server.URL, false)
	_, err := doPreparedKimiRequest(&Adaptor{}, c, info, bytes.NewBufferString(`{"model":"kimi-k3","messages":[],"kimi_tools":["web-search"]}`))
	require.Error(t, err)
	require.NotNil(t, info.KimiToolLoop)
	assert.Empty(t, info.KimiToolLoop.Usages)
}

func TestKimiFormulaLoopEnforcesManagedBounds(t *testing.T) {
	service.InitHttpClient()
	disableKimiTestPersistence(t)

	t.Run("round limit stops before another Fiber", func(t *testing.T) {
		var chats atomic.Int32
		var fibers atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/v1/formulas/moonshot/web-search:latest/tools":
				writeJSON(t, w, `{"tools":[{"type":"function","function":{"name":"web_search","parameters":{"type":"object"}}}]}`)
			case "/v1/chat/completions":
				round := chats.Add(1)
				writeJSON(t, w, `{"choices":[{"index":0,"message":{"role":"assistant","content":null,"tool_calls":[{"id":"call-`+fmt.Sprint(round)+`","type":"function","function":{"name":"web_search","arguments":"{}"}}]},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":5,"completion_tokens":2,"total_tokens":7}}`)
			case "/v1/formulas/moonshot/web-search:latest/fibers":
				fibers.Add(1)
				writeJSON(t, w, `{"status":"succeeded","context":{"output":"continue"}}`)
			default:
				http.NotFound(w, r)
			}
		}))
		defer server.Close()

		c, _ := newKimiTestContext(t, context.Background())
		_, err := doPreparedKimiRequest(&Adaptor{}, c, newKimiRelayInfo(server.URL, false), bytes.NewBufferString(`{"model":"kimi-k3","messages":[],"kimi_tools":["web-search"]}`))
		require.Error(t, err)
		assert.EqualValues(t, 8, chats.Load())
		assert.EqualValues(t, 7, fibers.Load())
	})

	t.Run("execution limit rejects whole oversized batch", func(t *testing.T) {
		var fibers atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/v1/formulas/moonshot/web-search:latest/tools":
				writeJSON(t, w, `{"tools":[{"type":"function","function":{"name":"web_search","parameters":{"type":"object"}}}]}`)
			case "/v1/chat/completions":
				calls := make([]map[string]any, 17)
				for index := range calls {
					calls[index] = map[string]any{
						"id":   fmt.Sprintf("call-%d", index),
						"type": "function",
						"function": map[string]any{
							"name":      "web_search",
							"arguments": "{}",
						},
					}
				}
				payload, err := common.Marshal(map[string]any{
					"choices": []any{map[string]any{
						"index":         0,
						"message":       map[string]any{"role": "assistant", "content": nil, "tool_calls": calls},
						"finish_reason": "tool_calls",
					}},
					"usage": map[string]any{"prompt_tokens": 5, "completion_tokens": 2, "total_tokens": 7},
				})
				require.NoError(t, err)
				writeJSON(t, w, string(payload))
			case "/v1/formulas/moonshot/web-search:latest/fibers":
				fibers.Add(1)
				writeJSON(t, w, `{"status":"succeeded","context":{"output":"unexpected"}}`)
			default:
				http.NotFound(w, r)
			}
		}))
		defer server.Close()

		c, _ := newKimiTestContext(t, context.Background())
		_, err := doPreparedKimiRequest(&Adaptor{}, c, newKimiRelayInfo(server.URL, false), bytes.NewBufferString(`{"model":"kimi-k3","messages":[],"kimi_tools":["web-search"]}`))
		require.Error(t, err)
		assert.Zero(t, fibers.Load())
	})

	t.Run("body cap applies only to managed requests", func(t *testing.T) {
		var ordinaryBytes atomic.Int64
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			ordinaryBytes.Store(int64(len(body)))
			writeJSON(t, w, `{"choices":[{"index":0,"message":{"role":"assistant","content":"ordinary"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`)
		}))
		defer server.Close()

		largeContent := strings.Repeat("x", (8<<20)+1)
		ordinaryBody, err := common.Marshal(map[string]any{
			"model":    "kimi-k3",
			"messages": []any{map[string]any{"role": "user", "content": largeContent}},
		})
		require.NoError(t, err)
		ordinaryContext, _ := newKimiTestContext(t, context.Background())
		ordinaryResp, err := doPreparedKimiRequest(&Adaptor{}, ordinaryContext, newKimiRelayInfo(server.URL, false), bytes.NewReader(ordinaryBody))
		require.NoError(t, err)
		defer ordinaryResp.(*http.Response).Body.Close()
		assert.EqualValues(t, len(ordinaryBody), ordinaryBytes.Load())

		managedBody, err := common.Marshal(map[string]any{
			"model":      "kimi-k3",
			"messages":   []any{map[string]any{"role": "user", "content": largeContent}},
			"kimi_tools": []string{"web-search"},
		})
		require.NoError(t, err)
		managedContext, _ := newKimiTestContext(t, context.Background())
		_, err = doPreparedKimiRequest(&Adaptor{}, managedContext, newKimiRelayInfo(server.URL, false), bytes.NewReader(managedBody))
		require.Error(t, err)
		var apiErr *types.NewAPIError
		require.True(t, errors.As(err, &apiErr))
		assert.Equal(t, http.StatusRequestEntityTooLarge, apiErr.StatusCode)
		assert.EqualValues(t, len(ordinaryBody), ordinaryBytes.Load(), "managed body must be rejected before upstream")
	})
}

func TestKimiFormulaLoopSSEPreservesContentWhenThinkingToContentIsEnabled(t *testing.T) {
	service.InitHttpClient()
	originalStreamingTimeout := channelconstant.StreamingTimeout
	channelconstant.StreamingTimeout = 30
	t.Cleanup(func() { channelconstant.StreamingTimeout = originalStreamingTimeout })

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/formulas/moonshot/web-search:latest/tools":
			writeJSON(t, w, `{"tools":[{"type":"function","function":{"name":"web_search","parameters":{"type":"object"}}}]}`)
		case "/v1/chat/completions":
			writeJSON(t, w, `{"id":"reasoning-final","object":"chat.completion","created":8,"model":"kimi-k3","choices":[{"index":0,"message":{"role":"assistant","reasoning_content":"I calculated","content":"5"},"finish_reason":"stop"}],"usage":{"prompt_tokens":9,"completion_tokens":4,"total_tokens":13}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	c.Request.Header.Set("Content-Type", "application/json")
	info := newKimiRelayInfo(server.URL, true)
	info.ChannelSetting.ThinkingToContent = true
	info.ThinkingContentInfo.IsFirstThinkingContent = true
	adaptor := &Adaptor{}
	result, err := doPreparedKimiRequest(adaptor, c, info, bytes.NewBufferString(`{"model":"kimi-k3","stream":true,"messages":[],"kimi_tools":["web-search"]}`))
	require.NoError(t, err)
	_, apiErr := adaptor.DoResponse(c, result.(*http.Response), info)
	require.Nil(t, apiErr)

	streamBody := recorder.Body.String()
	var content strings.Builder
	for _, frame := range strings.Split(streamBody, "\n\n") {
		data := strings.TrimPrefix(frame, "data: ")
		if data == frame || data == "" || data == "[DONE]" {
			continue
		}
		var chunk dto.ChatCompletionsStreamResponse
		require.NoError(t, common.Unmarshal([]byte(data), &chunk))
		for _, choice := range chunk.Choices {
			content.WriteString(choice.Delta.GetContentString())
		}
	}
	assert.Contains(t, content.String(), "<think>\nI calculated")
	assert.Contains(t, content.String(), "</think>\n")
	assert.Contains(t, content.String(), "5")
}

func TestKimiFormulaLoopDeadlineCancelsStalledResponseBody(t *testing.T) {
	service.InitHttpClient()
	originalStreamingTimeout := channelconstant.StreamingTimeout
	channelconstant.StreamingTimeout = 1
	t.Cleanup(func() { channelconstant.StreamingTimeout = originalStreamingTimeout })

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/formulas/moonshot/web-search:latest/tools":
			writeJSON(t, w, `{"tools":[{"type":"function","function":{"name":"web_search","parameters":{"type":"object"}}}]}`)
		case "/v1/chat/completions":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
			<-r.Context().Done()
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	requestContext, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	c, _ := newKimiTestContext(t, requestContext)
	startedAt := time.Now()
	_, err := doPreparedKimiRequest(&Adaptor{}, c, newKimiRelayInfo(server.URL, false), bytes.NewBufferString(`{"model":"kimi-k3","messages":[],"kimi_tools":["web-search"]}`))
	require.Error(t, err)
	assert.Less(t, time.Since(startedAt), 2*time.Second)
}
