package service

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/relayconvert"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeminiImageConversionPreservesRequestContext(t *testing.T) {
	previousDebug := common.DebugEnabled
	common.DebugEnabled = true
	t.Cleanup(func() { common.DebugEnabled = previousDebug })
	const png = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jRZkAAAAASUVORK5CYII="

	for _, entry := range []string{"default", "by-id", "via"} {
		t.Run(entry, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Set(common.RequestIdKey, "gemini-image-regression")
			t.Cleanup(func() { CleanupFileSources(c) })
			var request dto.GeneralOpenAIRequest
			require.NoError(t, common.UnmarshalJsonStr(`{"model":"gemini-3.5-flash-lite-minimal","messages":[{"role":"user","content":[{"type":"text","text":"Describe this image."},{"type":"image_url","image_url":{"url":"data:image/png;base64,`+png+`"}}]}]}`, &request))
			info := &relaycommon.RelayInfo{}
			var result *relayconvert.RequestResult
			var err error
			require.NotPanics(t, func() {
				switch entry {
				case "by-id":
					result, err = ConvertRequestByID(c, info, relayconvert.ConverterOpenAIChatToGeminiContent, &request)
				case "via":
					result, err = ConvertRequestVia(c, info, &request, types.RelayFormatGemini)
				default:
					result, err = ConvertRequest(c, info, types.RelayFormatGemini, &request)
				}
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			converted, ok := result.Value.(*dto.GeminiChatRequest)
			require.True(t, ok)
			require.Len(t, converted.Contents, 1)
			require.Len(t, converted.Contents[0].Parts, 2)
			assert.Equal(t, "Describe this image.", converted.Contents[0].Parts[0].Text)
			require.NotNil(t, converted.Contents[0].Parts[1].InlineData)
			assert.Equal(t, "image/png", converted.Contents[0].Parts[1].InlineData.MimeType)
			assert.Equal(t, png, converted.Contents[0].Parts[1].InlineData.Data)
			sources, exists := c.Get(string(constant.ContextKeyFileSourcesToCleanup))
			require.True(t, exists, "converted media must be registered for request cleanup")
			require.Len(t, sources, 1)
		})
	}
	t.Run("optional request context", func(t *testing.T) {
		imageBytes, err := base64.StdEncoding.DecodeString(png)
		require.NoError(t, err)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(imageBytes)
		}))
		defer server.Close()
		serverURL, err := url.Parse(server.URL)
		require.NoError(t, err)
		fetchSetting := system_setting.GetFetchSetting()
		previousFetch := *fetchSetting
		previousWorker := system_setting.WorkerUrl
		previousDownloadLimit := constant.MaxFileDownloadMB
		*fetchSetting = system_setting.FetchSetting{EnableSSRFProtection: true, AllowPrivateIp: true, AllowedPorts: []string{serverURL.Port()}}
		system_setting.WorkerUrl = ""
		constant.MaxFileDownloadMB = 1
		t.Cleanup(func() {
			*fetchSetting = previousFetch
			system_setting.WorkerUrl = previousWorker
			constant.MaxFileDownloadMB = previousDownloadLimit
		})
		initDefaultHTTPClientFixture(t)
		for _, imageURL := range []string{"data:image/png;base64," + png, server.URL + "/image.png"} {
			var request dto.GeneralOpenAIRequest
			require.NoError(t, common.UnmarshalJsonStr(`{"model":"gemini-3.5-flash-lite","messages":[{"role":"user","content":[{"type":"image_url","image_url":{"url":"`+imageURL+`"}}]}]}`, &request))
			var result *relayconvert.RequestResult
			var err error
			require.NotPanics(t, func() {
				result, err = ConvertRequest(nil, &relaycommon.RelayInfo{}, types.RelayFormatGemini, &request)
			})
			require.NoError(t, err)
			require.NotNil(t, result)
			converted, ok := result.Value.(*dto.GeminiChatRequest)
			require.True(t, ok)
			require.Len(t, converted.Contents, 1)
			require.Len(t, converted.Contents[0].Parts, 1)
			require.NotNil(t, converted.Contents[0].Parts[0].InlineData)
			assert.Equal(t, "image/png", converted.Contents[0].Parts[0].InlineData.MimeType)
			assert.Equal(t, png, converted.Contents[0].Parts[0].InlineData.Data)
		}
	})
}

func TestResponseConverterFacades(t *testing.T) {
	cache5m, cache1h := NormalizeCacheCreationSplit(10, 3, 2)
	assert.Equal(t, 8, cache5m)
	assert.Equal(t, 2, cache1h)

	chatResp := &dto.OpenAITextResponse{
		Id:    "chatcmpl_1",
		Model: "gpt-test",
		Choices: []dto.OpenAITextResponseChoice{
			{
				Message: dto.Message{
					Role:    "assistant",
					Content: "hello",
				},
				FinishReason: "stop",
			},
		},
	}

	claudeResp := ResponseOpenAI2Claude(chatResp, &relaycommon.RelayInfo{})
	require.NotNil(t, claudeResp)
	assert.Equal(t, "message", claudeResp.Type)

	geminiResp := ResponseOpenAI2Gemini(chatResp, &relaycommon.RelayInfo{})
	require.NotNil(t, geminiResp)
	require.Len(t, geminiResp.Candidates, 1)
}

func TestStreamResponseConverterFacades(t *testing.T) {
	info := &relaycommon.RelayInfo{
		SendResponseCount: 1,
		ClaudeConvertInfo: &relaycommon.ClaudeConvertInfo{
			LastMessagesType: relaycommon.LastMessageTypeNone,
		},
	}
	streamResp := &dto.ChatCompletionsStreamResponse{
		Id:    "chatcmpl_1",
		Model: "gpt-test",
		Choices: []dto.ChatCompletionsStreamResponseChoice{
			{
				Delta: dto.ChatCompletionsStreamResponseChoiceDelta{
					Content: ptrValue("hello"),
				},
			},
		},
	}

	claudeResponses := StreamResponseOpenAI2Claude(streamResp, info)
	require.NotEmpty(t, claudeResponses)

	geminiResp := StreamResponseOpenAI2Gemini(streamResp, &relaycommon.RelayInfo{})
	require.NotNil(t, geminiResp)
	require.Len(t, geminiResp.Candidates, 1)
}

func TestRequestConverterFacadeAcceptsTypedNilRelayInfo(t *testing.T) {
	for _, target := range []types.RelayFormat{types.RelayFormatClaude, types.RelayFormatGemini} {
		t.Run(string(target), func(t *testing.T) {
			var info *relaycommon.RelayInfo
			request := &dto.GeneralOpenAIRequest{
				Model: "test-model",
				Messages: []dto.Message{
					{Role: "user", Content: "hello"},
				},
			}

			result, err := ConvertRequest(nil, info, target, request)

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, target, result.To)
		})
	}
}

func TestStreamResponseConverterFacadesAcceptTypedNilRelayInfo(t *testing.T) {
	var info *relaycommon.RelayInfo
	streamResp := &dto.ChatCompletionsStreamResponse{
		Id:    "chatcmpl_typed_nil",
		Model: "gpt-test",
		Choices: []dto.ChatCompletionsStreamResponseChoice{
			{
				Delta: dto.ChatCompletionsStreamResponseChoiceDelta{
					Content: ptrValue("hello"),
				},
			},
		},
	}

	claudeResponses := StreamResponseOpenAI2Claude(streamResp, info)
	require.NotEmpty(t, claudeResponses)
	assert.Equal(t, "content_block_start", claudeResponses[0].Type)

	geminiResp := StreamResponseOpenAI2Gemini(streamResp, info)
	require.NotNil(t, geminiResp)
	require.Len(t, geminiResp.Candidates, 1)
	assert.Zero(t, geminiResp.UsageMetadata.PromptTokenCount)
}

func ptrValue[T any](value T) *T {
	return &value
}
