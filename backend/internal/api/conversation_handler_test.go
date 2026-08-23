package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"freedinner/backend/internal/store"

	"github.com/gin-gonic/gin"
)

type fakeConversationStore struct {
	createdTitle  string
	listWebCalled bool
	messages      []store.Message
}

func (s *fakeConversationStore) Create(ctx context.Context, userID, title string) (store.Conversation, error) {
	s.createdTitle = title
	return store.Conversation{ID: "web-1", UserID: userID, Title: title, Source: "web_chat"}, nil
}

func (s *fakeConversationStore) ListWeb(ctx context.Context, userID string) ([]store.Conversation, error) {
	s.listWebCalled = true
	return []store.Conversation{
		{ID: "web-1", UserID: userID, Title: "Web", Source: "web_chat"},
	}, nil
}

func (s *fakeConversationStore) ListMessages(ctx context.Context, userID, conversationID string) ([]store.Message, error) {
	return s.messages, nil
}

type fakeConversationLLM struct {
	err error
}

func (l fakeConversationLLM) SendMessage(ctx context.Context, userID, conversationID, content string) (store.SendMessageResult, error) {
	if l.err != nil {
		return store.SendMessageResult{}, l.err
	}
	return store.SendMessageResult{
		TurnID:           "turn-1",
		UserMessage:      store.Message{ID: "msg-user", Role: "user", Content: content},
		AssistantMessage: store.Message{ID: "msg-assistant", Role: "assistant", Content: "ok"},
	}, nil
}

func TestConversationListUsesWebOnlyStore(t *testing.T) {
	gin.SetMode(gin.TestMode)
	conversations := &fakeConversationStore{}
	handler := &ConversationHandler{conversations: conversations, llm: fakeConversationLLM{}}
	router := authenticatedTestRouter()
	router.GET("/conversations", handler.List)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/conversations", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	if !conversations.listWebCalled {
		t.Fatal("expected handler to call ListWeb")
	}
	var body struct {
		Data []store.Conversation `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if len(body.Data) != 1 || body.Data[0].Source != "web_chat" {
		t.Fatalf("expected only web_chat conversations, got %#v", body.Data)
	}
}

func TestConversationSendRejectsReadonlyChannelConversation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := &ConversationHandler{
		conversations: &fakeConversationStore{},
		llm:           fakeConversationLLM{err: store.ErrConversationReadonlyInWeb},
	}
	router := authenticatedTestRouter()
	router.POST("/conversations/:conversation_id/messages", handler.SendMessage)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/conversations/channel-1/messages", strings.NewReader(`{"content":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	var body responseBody
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Error == nil || body.Error.Code != "CHANNEL_CONVERSATION_READONLY_IN_WEB" {
		t.Fatalf("unexpected error body: %#v", body)
	}
}

func TestConversationSendMapsMissingConversation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := &ConversationHandler{
		conversations: &fakeConversationStore{},
		llm:           fakeConversationLLM{err: store.ErrNotFound},
	}
	router := authenticatedTestRouter()
	router.POST("/conversations/:conversation_id/messages", handler.SendMessage)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/conversations/missing/messages", strings.NewReader(`{"content":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestConversationSendRejectsEmptyMessageBeforeLLM(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := &ConversationHandler{
		conversations: &fakeConversationStore{},
		llm:           fakeConversationLLM{err: errors.New("should not be called")},
	}
	router := authenticatedTestRouter()
	router.POST("/conversations/:conversation_id/messages", handler.SendMessage)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/conversations/web-1/messages", strings.NewReader(`{"content":"   "}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func authenticatedTestRouter() *gin.Engine {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(userIDContextKey, "user-1")
		c.Next()
	})
	return router
}
