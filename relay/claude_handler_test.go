package relay

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestClaudeHelperClearsTopPForSonnet46EffortSuffix(t *testing.T) {
	type upstreamResult struct {
		request dto.ClaudeRequest
		err     error
	}

	upstreamResults := make(chan upstreamResult, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request dto.ClaudeRequest
		err := common.DecodeJson(r.Body, &request)
		if r.URL.Path != "/v1/messages" {
			err = fmt.Errorf("unexpected upstream path: %s", r.URL.Path)
		}
		upstreamResults <- upstreamResult{request: request, err: err}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"type":"error","error":{"type":"invalid_request_error","message":"test stop"}}`))
	}))
	t.Cleanup(upstream.Close)
	service.InitHttpClient()

	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	c.Request.Header.Set("Content-Type", "application/json")
	common.SetContextKey(c, constant.ContextKeyChannelType, constant.ChannelTypeAnthropic)
	common.SetContextKey(c, constant.ContextKeyChannelBaseUrl, upstream.URL)
	common.SetContextKey(c, constant.ContextKeyChannelKey, "test-key")
	common.SetContextKey(c, constant.ContextKeyOriginalModel, "claude-sonnet-4-6-high")

	topP := 0.8
	temperature := 0.7
	maxTokens := uint(128)
	request := &dto.ClaudeRequest{
		Model:       "claude-sonnet-4-6-high",
		Messages:    []dto.ClaudeMessage{{Role: "user", Content: "hello"}},
		MaxTokens:   &maxTokens,
		Temperature: &temperature,
		TopP:        &topP,
	}
	info := &relaycommon.RelayInfo{
		OriginModelName: "claude-sonnet-4-6-high",
		RelayMode:       relayconstant.RelayModeUnknown,
		RelayFormat:     types.RelayFormatClaude,
		Request:         request,
	}

	relayErr := ClaudeHelper(c, info)
	require.NotNil(t, relayErr)
	result := <-upstreamResults
	require.NoError(t, result.err)
	assert.Equal(t, "claude-sonnet-4-6", result.request.Model)
	require.NotNil(t, result.request.Thinking)
	assert.Equal(t, "adaptive", result.request.Thinking.Type)
	assert.Equal(t, "high", gjson.GetBytes(result.request.OutputConfig, "effort").String())
	require.NotNil(t, result.request.Temperature)
	assert.Equal(t, 1.0, *result.request.Temperature)
	assert.Nil(t, result.request.TopP)
}
