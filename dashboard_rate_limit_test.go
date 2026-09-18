package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/router"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDashboardAssetsPreservePageRateLimitBudget(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousRedis, previousClient := common.RedisEnabled, common.RDB
	previousEnable, previousNum, previousDuration := common.GlobalWebRateLimitEnable, common.GlobalWebRateLimitNum, common.GlobalWebRateLimitDuration
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	common.RedisEnabled, common.RDB = true, client
	common.GlobalWebRateLimitEnable, common.GlobalWebRateLimitNum, common.GlobalWebRateLimitDuration = true, 2, 60
	t.Cleanup(func() {
		_ = client.Close()
		common.RedisEnabled, common.RDB = previousRedis, previousClient
		common.GlobalWebRateLimitEnable, common.GlobalWebRateLimitNum, common.GlobalWebRateLimitDuration = previousEnable, previousNum, previousDuration
	})
	engine := gin.New()
	require.NoError(t, engine.SetTrustedProxies(nil))
	router.SetWebRouter(engine, router.WebAssets{BuildFS: buildFS, IndexPage: indexPage}, func(c *gin.Context) { c.Next() })
	for _, path := range []string{"/index.html", "/index.html", "/index.html", "/pricing", "/channels"} {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, path, nil)
		request.RemoteAddr = "192.0.2.90:1234"
		engine.ServeHTTP(response, request)
		if path == "/index.html" {
			assert.Equal(t, http.StatusMovedPermanently, response.Code, path)
		} else {
			assert.Equal(t, http.StatusOK, response.Code, path)
		}
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/pricing", nil)
	request.RemoteAddr = "192.0.2.90:1234"
	engine.ServeHTTP(response, request)
	assert.Equal(t, http.StatusTooManyRequests, response.Code, "page requests remain limited")
	assert.Equal(t, "no-store", response.Header().Get("Cache-Control"))
}

func TestSessionRefreshDoesNotConsumeLoginRateLimitBudget(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousRedis, previousClient := common.RedisEnabled, common.RDB
	previousAPI := common.GlobalApiRateLimitEnable
	previousEnable, previousNum, previousDuration := common.CriticalRateLimitEnable, common.CriticalRateLimitNum, common.CriticalRateLimitDuration
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	common.RedisEnabled, common.RDB = true, client
	common.GlobalApiRateLimitEnable = false
	common.CriticalRateLimitEnable, common.CriticalRateLimitNum, common.CriticalRateLimitDuration = true, 2, 60
	t.Setenv("AUTH_REFRESH_RATE_LIMIT", "3")
	t.Setenv("AUTH_REFRESH_RATE_LIMIT_DURATION", "60")
	t.Cleanup(func() {
		_ = client.Close()
		common.RedisEnabled, common.RDB = previousRedis, previousClient
		common.GlobalApiRateLimitEnable = previousAPI
		common.CriticalRateLimitEnable, common.CriticalRateLimitNum, common.CriticalRateLimitDuration = previousEnable, previousNum, previousDuration
	})
	engine := gin.New()
	require.NoError(t, engine.SetTrustedProxies(nil))
	router.SetApiRouter(engine)
	for _, path := range []string{"/api/user/auth/refresh", "/api/user/auth/refresh", "/api/user/auth/refresh", "/api/user/login", "/api/user/login"} {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, path, strings.NewReader("{"))
		request.Header.Set("Content-Type", "application/json")
		request.RemoteAddr = "192.0.2.91:1234"
		engine.ServeHTTP(response, request)
		if path == "/api/user/auth/refresh" {
			assert.Equal(t, http.StatusUnauthorized, response.Code, path)
		} else {
			assert.Equal(t, http.StatusOK, response.Code, path)
		}
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/user/login", strings.NewReader("{"))
	request.RemoteAddr = "192.0.2.91:1234"
	engine.ServeHTTP(response, request)
	assert.Equal(t, http.StatusTooManyRequests, response.Code, "login attempts remain limited")
	response = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/api/user/auth/refresh", nil)
	request.RemoteAddr = "192.0.2.91:1234"
	engine.ServeHTTP(response, request)
	assert.Equal(t, http.StatusTooManyRequests, response.Code, "refresh has its own limit")
	assert.Equal(t, "60", response.Header().Get("Retry-After"))
}
