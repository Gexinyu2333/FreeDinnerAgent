package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"freedinner/backend/internal/auth"

	"github.com/gin-gonic/gin"
)

func TestOKAndErrorResponseShape(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/ok", func(c *gin.Context) {
		OK(c, gin.H{"status": "ok"})
	})
	router.GET("/error", func(c *gin.Context) {
		Error(c, http.StatusBadRequest, "BAD_REQUEST", "bad input")
	})

	okRecorder := httptest.NewRecorder()
	router.ServeHTTP(okRecorder, httptest.NewRequest(http.MethodGet, "/ok", nil))
	if okRecorder.Code != http.StatusOK {
		t.Fatalf("/ok status = %d", okRecorder.Code)
	}
	var okBody responseBody
	if err := json.Unmarshal(okRecorder.Body.Bytes(), &okBody); err != nil {
		t.Fatalf("decode ok body: %v", err)
	}
	if okBody.Error != nil || okBody.Data == nil {
		t.Fatalf("unexpected ok body: %#v", okBody)
	}

	errorRecorder := httptest.NewRecorder()
	router.ServeHTTP(errorRecorder, httptest.NewRequest(http.MethodGet, "/error", nil))
	if errorRecorder.Code != http.StatusBadRequest {
		t.Fatalf("/error status = %d", errorRecorder.Code)
	}
	var errorBody responseBody
	if err := json.Unmarshal(errorRecorder.Body.Bytes(), &errorBody); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	if errorBody.Error == nil || errorBody.Error.Code != "BAD_REQUEST" || errorBody.Data != nil {
		t.Fatalf("unexpected error body: %#v", errorBody)
	}
}

func TestCORSHandlesPreflight(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(CORS())
	router.GET("/resource", func(c *gin.Context) {
		OK(c, "ok")
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodOptions, "/resource", nil))

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d", recorder.Code)
	}
	if recorder.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("missing CORS header: %#v", recorder.Header())
	}
}

func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	token, err := auth.GenerateAccessToken("secret", "user-1", "alice", time.Hour)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	router := gin.New()
	router.GET("/me", AuthMiddleware("secret"), func(c *gin.Context) {
		userID, ok := CurrentUserID(c)
		if !ok {
			t.Fatal("CurrentUserID should be set")
		}
		OK(c, gin.H{"user_id": userID})
	})

	missing := httptest.NewRecorder()
	router.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/me", nil))
	if missing.Code != http.StatusUnauthorized {
		t.Fatalf("missing auth status = %d", missing.Code)
	}

	authorized := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(authorized, req)
	if authorized.Code != http.StatusOK {
		t.Fatalf("authorized status = %d body=%s", authorized.Code, authorized.Body.String())
	}
}
