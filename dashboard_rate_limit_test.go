package main

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/router"
	"github.com/QuantumNous/new-api/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
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
			assert.Equal(t, "max-age=604800", response.Header().Get("Cache-Control"), path)
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

func TestDashboardFallbackRedisFailureIsNotCacheable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousRedis, previousClient := common.RedisEnabled, common.RDB
	previousEnable, previousNum, previousDuration := common.GlobalWebRateLimitEnable, common.GlobalWebRateLimitNum, common.GlobalWebRateLimitDuration
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	common.RedisEnabled, common.RDB = true, client
	common.GlobalWebRateLimitEnable, common.GlobalWebRateLimitNum, common.GlobalWebRateLimitDuration = true, 2, 60
	require.NoError(t, client.Close())
	t.Cleanup(func() {
		common.RedisEnabled, common.RDB = previousRedis, previousClient
		common.GlobalWebRateLimitEnable, common.GlobalWebRateLimitNum, common.GlobalWebRateLimitDuration = previousEnable, previousNum, previousDuration
	})

	engine := gin.New()
	require.NoError(t, engine.SetTrustedProxies(nil))
	router.SetWebRouter(engine, router.WebAssets{BuildFS: buildFS, IndexPage: indexPage}, func(c *gin.Context) { c.Next() })
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/pricing", nil)
	request.RemoteAddr = "192.0.2.92:1234"
	engine.ServeHTTP(response, request)

	assert.Equal(t, http.StatusInternalServerError, response.Code)
	assert.Equal(t, "no-store", response.Header().Get("Cache-Control"))
	assert.Empty(t, response.Header().Get("Cache-Version"))
	assert.NotContains(t, response.Header().Get("Cache-Control"), "max-age")
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
	t.Setenv("AUTH_REFRESH_IP_RATE_LIMIT", "3")
	t.Setenv("AUTH_REFRESH_IP_RATE_LIMIT_DURATION", "60")
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
			assert.Equal(t, "no-store", response.Header().Get("Cache-Control"), path)
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
	assert.Equal(t, "no-store", response.Header().Get("Cache-Control"))
}

func TestAuthRefreshEarlyFailuresAreNoStore(t *testing.T) {
	t.Run("origin guard", func(t *testing.T) {
		previousSecure := common.SessionCookieSecure
		previousTrustedURLs := common.SessionCookieTrustedURLs
		previousAPI := common.GlobalApiRateLimitEnable
		common.SessionCookieSecure = true
		common.SessionCookieTrustedURLs = nil
		common.GlobalApiRateLimitEnable = false
		t.Cleanup(func() {
			common.SessionCookieSecure = previousSecure
			common.SessionCookieTrustedURLs = previousTrustedURLs
			common.GlobalApiRateLimitEnable = previousAPI
		})

		engine := gin.New()
		router.SetApiRouter(engine)
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "https://panel.example.com/api/user/auth/refresh", nil)
		request.Host = "panel.example.com"
		engine.ServeHTTP(response, request)

		assert.Equal(t, http.StatusForbidden, response.Code)
		assert.Equal(t, "no-store", response.Header().Get("Cache-Control"))
		assert.NotContains(t, response.Header().Get("Cache-Control"), "max-age")
	})

	t.Run("refresh limiter backend", func(t *testing.T) {
		previousRedis, previousClient := common.RedisEnabled, common.RDB
		previousAPI := common.GlobalApiRateLimitEnable
		previousCritical := common.CriticalRateLimitEnable
		previousSecure := common.SessionCookieSecure
		server := miniredis.RunT(t)
		client := redis.NewClient(&redis.Options{Addr: server.Addr()})
		common.RedisEnabled, common.RDB = true, client
		common.GlobalApiRateLimitEnable = false
		common.CriticalRateLimitEnable = true
		common.SessionCookieSecure = false
		require.NoError(t, client.Close())
		t.Cleanup(func() {
			common.RedisEnabled, common.RDB = previousRedis, previousClient
			common.GlobalApiRateLimitEnable = previousAPI
			common.CriticalRateLimitEnable = previousCritical
			common.SessionCookieSecure = previousSecure
		})

		engine := gin.New()
		require.NoError(t, engine.SetTrustedProxies(nil))
		router.SetApiRouter(engine)
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/user/auth/refresh", nil)
		request.RemoteAddr = "192.0.2.93:1234"
		engine.ServeHTTP(response, request)

		assert.Equal(t, http.StatusInternalServerError, response.Code)
		assert.Equal(t, "no-store", response.Header().Get("Cache-Control"))
		assert.NotContains(t, response.Header().Get("Cache-Control"), "max-age")
	})
}

func TestForgedRefreshSIDsShareUnverifiedIPBaseline(t *testing.T) {
	tests := []struct {
		name        string
		redis       bool
		remoteAddr  string
		sessionBase string
	}{
		{name: "memory", remoteAddr: "192.0.2.94:1234", sessionBase: "018f47c0-11ee-7c3c-9d2d-0242ac12001"},
		{name: "redis", redis: true, remoteAddr: "192.0.2.95:1234", sessionBase: "018f47c0-11ee-7c3c-9d2d-0242ac12002"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			previousDB := model.DB
			previousRedis, previousClient := common.RedisEnabled, common.RDB
			previousAPI := common.GlobalApiRateLimitEnable
			previousCritical, previousCriticalNum, previousCriticalDuration := common.CriticalRateLimitEnable, common.CriticalRateLimitNum, common.CriticalRateLimitDuration
			previousSecure := common.SessionCookieSecure
			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
			require.NoError(t, err)
			require.NoError(t, db.AutoMigrate(&model.UserSession{}))
			model.DB = db
			common.RedisEnabled = test.redis
			common.RDB = nil
			common.GlobalApiRateLimitEnable = false
			common.CriticalRateLimitEnable, common.CriticalRateLimitNum, common.CriticalRateLimitDuration = true, 1, 60
			common.SessionCookieSecure = false
			t.Setenv("AUTH_REFRESH_RATE_LIMIT", "100")
			t.Setenv("AUTH_REFRESH_RATE_LIMIT_DURATION", "60")
			t.Setenv("AUTH_REFRESH_IP_RATE_LIMIT", "2")
			t.Setenv("AUTH_REFRESH_IP_RATE_LIMIT_DURATION", "60")
			if test.redis {
				server := miniredis.RunT(t)
				common.RDB = redis.NewClient(&redis.Options{Addr: server.Addr()})
			}
			t.Cleanup(func() {
				if test.redis && common.RDB != nil {
					_ = common.RDB.Close()
				}
				model.DB = previousDB
				common.RedisEnabled, common.RDB = previousRedis, previousClient
				common.GlobalApiRateLimitEnable = previousAPI
				common.CriticalRateLimitEnable, common.CriticalRateLimitNum, common.CriticalRateLimitDuration = previousCritical, previousCriticalNum, previousCriticalDuration
				common.SessionCookieSecure = previousSecure
			})

			engine := gin.New()
			require.NoError(t, engine.SetTrustedProxies(nil))
			router.SetApiRouter(engine)
			for index := range 3 {
				sid := test.sessionBase + strconv.Itoa(index)
				rawToken := sid + ".forged-secret"
				response := httptest.NewRecorder()
				request := httptest.NewRequest(http.MethodPost, "/api/user/auth/refresh", nil)
				request.RemoteAddr = test.remoteAddr
				request.AddCookie(&http.Cookie{Name: service.RefreshCookieName, Value: rawToken})
				engine.ServeHTTP(response, request)

				if index < 2 {
					assert.Equal(t, http.StatusUnauthorized, response.Code)
				} else {
					assert.Equal(t, http.StatusTooManyRequests, response.Code)
					retryAfter, err := strconv.Atoi(response.Header().Get("Retry-After"))
					require.NoError(t, err)
					assert.Positive(t, retryAfter)
				}
				assert.Equal(t, "no-store", response.Header().Get("Cache-Control"))
				assert.NotContains(t, response.Body.String(), sid)
				assert.NotContains(t, response.Body.String(), rawToken)
			}

			login := httptest.NewRecorder()
			loginRequest := httptest.NewRequest(http.MethodPost, "/api/user/login", strings.NewReader("{"))
			loginRequest.RemoteAddr = test.remoteAddr
			engine.ServeHTTP(login, loginRequest)
			assert.Equal(t, http.StatusOK, login.Code, "forged refreshes must not consume login budget")
		})
	}
}
