package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	channelsvc "freedinner/backend/internal/channel"
	"freedinner/backend/internal/store"

	"github.com/gin-gonic/gin"
)

type fakeChannelService struct {
	webhookConnectionID string
	webhookSecret       string
	webhookPayload      string
	transcriptCalled    bool
	draftInput          channelsvc.CreateOutboxDraftInput
	approvedOutboxID    string
	sentOutboxID        string
}

func (s *fakeChannelService) ListProviders(ctx context.Context) ([]store.ChannelProviderDefinition, error) {
	return nil, nil
}

func (s *fakeChannelService) CreateConnection(ctx context.Context, input channelsvc.CreateConnectionInput) (store.ChannelConnection, error) {
	return store.ChannelConnection{}, nil
}

func (s *fakeChannelService) ListConnections(ctx context.Context, userID string) ([]store.ChannelConnection, error) {
	return nil, nil
}

func (s *fakeChannelService) UpdateConnection(ctx context.Context, input channelsvc.UpdateConnectionInput) (store.ChannelConnection, error) {
	return store.ChannelConnection{}, nil
}

func (s *fakeChannelService) DeleteConnection(ctx context.Context, userID, connectionID string) error {
	return nil
}

func (s *fakeChannelService) ListEndpoints(ctx context.Context, userID, connectionID string) ([]store.ChannelConnectionEndpoint, error) {
	return nil, nil
}

func (s *fakeChannelService) UpsertPolicy(ctx context.Context, input channelsvc.UpsertPolicyInput) (store.ChannelPolicy, error) {
	return store.ChannelPolicy{}, nil
}

func (s *fakeChannelService) ListPolicies(ctx context.Context, userID, connectionID string) ([]store.ChannelPolicy, error) {
	return nil, nil
}

func (s *fakeChannelService) DeletePolicy(ctx context.Context, userID, connectionID, policyID string) error {
	return nil
}

func (s *fakeChannelService) HandleWebhook(ctx context.Context, connectionID, providedSecret string, rawPayload []byte) (channelsvc.WebhookResult, error) {
	s.webhookConnectionID = connectionID
	s.webhookSecret = providedSecret
	s.webhookPayload = string(rawPayload)
	return channelsvc.WebhookResult{
		InboxEvent: store.ChannelInboxEvent{
			ID:                  "inbox-1",
			ChannelConnectionID: connectionID,
			ShouldTriggerAgent:  true,
			Status:              "processed",
		},
	}, nil
}

func (s *fakeChannelService) ListExternalConversations(ctx context.Context, userID, connectionID string, limit int) ([]store.ExternalConversation, error) {
	return nil, nil
}

func (s *fakeChannelService) ListExternalConversationMessages(ctx context.Context, userID, connectionID, externalConversationID string) ([]store.Message, error) {
	s.transcriptCalled = true
	return []store.Message{
		{
			ID:             "msg-1",
			UserID:         userID,
			ConversationID: "conversation-channel-1",
			Role:           "user",
			Content:        "来自 QQ 群的消息",
			Metadata:       json.RawMessage(`{"source":"channel","provider":"qq","external_sender_id":"1106861129","external_sender_name":"Seia"}`),
		},
	}, nil
}

func (s *fakeChannelService) ListInboxEvents(ctx context.Context, userID, connectionID string, limit int) ([]store.ChannelInboxEvent, error) {
	return nil, nil
}

func (s *fakeChannelService) ListOutboxMessages(ctx context.Context, userID, connectionID string, status *string, limit int) ([]store.ChannelOutboxMessage, error) {
	return nil, nil
}

func (s *fakeChannelService) CreateOutboxDraft(ctx context.Context, input channelsvc.CreateOutboxDraftInput) (store.ChannelOutboxMessage, error) {
	s.draftInput = input
	return store.ChannelOutboxMessage{
		ID:                     "outbox-1",
		UserID:                 input.UserID,
		ChannelConnectionID:    input.ConnectionID,
		ExternalConversationID: &input.ExternalConversationID,
		MessageType:            "text",
		Content:                input.Content,
		RequiresApproval:       input.RequiresApproval == nil || *input.RequiresApproval,
		Status:                 "pending",
	}, nil
}

func (s *fakeChannelService) ApproveOutboxMessage(ctx context.Context, userID, outboxID string) (store.ChannelOutboxMessage, error) {
	s.approvedOutboxID = outboxID
	return store.ChannelOutboxMessage{ID: outboxID, UserID: userID, Status: "approved"}, nil
}

func (s *fakeChannelService) CancelOutboxMessage(ctx context.Context, userID, outboxID string) (store.ChannelOutboxMessage, error) {
	return store.ChannelOutboxMessage{ID: outboxID, UserID: userID, Status: "cancelled"}, nil
}

func (s *fakeChannelService) SendOutboxMessage(ctx context.Context, userID, outboxID string) (store.ChannelOutboxMessage, error) {
	s.sentOutboxID = outboxID
	return store.ChannelOutboxMessage{ID: outboxID, UserID: userID, Status: "sent"}, nil
}

func TestChannelWebhookPassesSecretAndRawPayloadToService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &fakeChannelService{}
	handler := &ChannelHandler{channels: service}
	router := gin.New()
	router.POST("/channels/:connection_id/webhook", handler.Webhook)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/channels/conn-1/webhook?token=query-secret", strings.NewReader(`{"post_type":"message"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	if service.webhookConnectionID != "conn-1" || service.webhookSecret != "query-secret" {
		t.Fatalf("unexpected webhook call: connection=%q secret=%q", service.webhookConnectionID, service.webhookSecret)
	}
	if service.webhookPayload != `{"post_type":"message"}` {
		t.Fatalf("unexpected payload: %s", service.webhookPayload)
	}
}

func TestChannelTranscriptEndpointReadsExternalConversationMessages(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &fakeChannelService{}
	handler := &ChannelHandler{channels: service}
	router := authenticatedTestRouter()
	router.GET("/me/channel-connections/:connection_id/external-conversations/:external_conversation_id/messages", handler.ExternalConversationMessages)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/me/channel-connections/conn-1/external-conversations/group-462934780/messages", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	if !service.transcriptCalled {
		t.Fatal("expected transcript endpoint to read external conversation messages")
	}
	var body struct {
		Data []store.Message `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if len(body.Data) != 1 || !strings.Contains(string(body.Data[0].Metadata), `"source":"channel"`) {
		t.Fatalf("expected channel transcript message, got %#v", body.Data)
	}
}

func TestChannelOutboxDraftEndpointDoesNotUseWebChatSend(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &fakeChannelService{}
	handler := &ChannelHandler{channels: service}
	router := authenticatedTestRouter()
	router.POST("/me/channel-connections/:connection_id/external-conversations/:external_conversation_id/outbox-drafts", handler.CreateOutboxDraft)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/me/channel-connections/conn-1/external-conversations/group-462934780/outbox-drafts", strings.NewReader(`{"content":"人工回复","message_type":"text"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	if service.draftInput.UserID != "user-1" || service.draftInput.ConnectionID != "conn-1" || service.draftInput.ExternalConversationID != "group-462934780" {
		t.Fatalf("unexpected draft input: %#v", service.draftInput)
	}
	if service.draftInput.Content != "人工回复" {
		t.Fatalf("unexpected draft content: %q", service.draftInput.Content)
	}
	var body struct {
		Data store.ChannelOutboxMessage `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Data.Status != "pending" || body.Data.Content != "人工回复" {
		t.Fatalf("expected pending outbox draft, got %#v", body.Data)
	}
}

func TestChannelOutboxApproveEndpointUsesChannelService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &fakeChannelService{}
	handler := &ChannelHandler{channels: service}
	router := authenticatedTestRouter()
	router.POST("/channel-outbox-messages/:outbox_id/approve", handler.ApproveOutbox)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/channel-outbox-messages/outbox-1/approve", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	if service.approvedOutboxID != "outbox-1" {
		t.Fatalf("expected approve to use channel service, got %q", service.approvedOutboxID)
	}
	var body struct {
		Data store.ChannelOutboxMessage `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Data.Status != "approved" {
		t.Fatalf("expected approved outbox, got %#v", body.Data)
	}
}

func TestChannelOutboxSendEndpointUsesChannelService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &fakeChannelService{}
	handler := &ChannelHandler{channels: service}
	router := authenticatedTestRouter()
	router.POST("/channel-outbox-messages/:outbox_id/send", handler.SendOutbox)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/channel-outbox-messages/outbox-1/send", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	if service.sentOutboxID != "outbox-1" {
		t.Fatalf("expected send to use channel service, got %q", service.sentOutboxID)
	}
	var body struct {
		Data store.ChannelOutboxMessage `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Data.Status != "sent" {
		t.Fatalf("expected sent outbox, got %#v", body.Data)
	}
}
