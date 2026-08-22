package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"freedinner/backend/internal/agent"
	"freedinner/backend/internal/contextmgr"
	"freedinner/backend/internal/memory"
	"freedinner/backend/internal/store"
)

func (s *Service) findFallbackProvider(ctx context.Context, userID, currentProviderID string) (store.ModelProvider, string, bool) {
	if s.modelProviders == nil {
		return store.ModelProvider{}, "", false
	}
	providers, err := s.modelProviders.List(ctx, userID)
	if err != nil {
		return store.ModelProvider{}, "", false
	}
	for _, provider := range providers {
		if provider.ID == currentProviderID || provider.Status != "active" || provider.Provider != "openai" {
			continue
		}
		apiKey, err := s.crypto.Decrypt(provider.EncryptedChatAPIKey)
		if err != nil || strings.TrimSpace(apiKey) == "" {
			continue
		}
		return provider, apiKey, true
	}
	return store.ModelProvider{}, "", false
}

func (s *Service) observeMemory(ctx context.Context, in loopInput, action agent.Action) agent.Observation {
	if s.memories == nil || !in.cfg.MemoryEnabled {
		return agent.Observation{ActionType: agent.ActionMemorySearch, Text: "memory disabled", Failed: true}
	}
	result, err := s.memories.Retrieve(ctx, memory.RetrieveInput{
		UserID:           in.userID,
		ConversationID:   in.conversationID,
		MessageID:        &in.userMessage.ID,
		Query:            action.Query,
		MaxMemoryTokens:  900,
		LogRetrieval:     true,
		SemanticOnDemand: in.cfg.SemanticMemoryEnabled,
	})
	if err != nil {
		return agent.Observation{ActionType: agent.ActionMemorySearch, Text: err.Error(), Failed: true}
	}
	return agent.Observation{ActionType: agent.ActionMemorySearch, Text: renderMemoryObservation(result)}
}

func (s *Service) observeTool(ctx context.Context, in loopInput, stepID string, action agent.Action) (agent.Observation, bool) {
	if s.tools == nil || !in.cfg.ToolUseEnabled {
		return agent.Observation{ActionType: agent.ActionToolCall, Text: "tool use disabled", Failed: true}, false
	}
	_, _ = s.harness.AddEvent(ctx, event(in.turn, "tool_call_started", map[string]any{
		"loop_step_id": stepID,
		"tool_name":    action.ToolName,
	}))
	idempotencyKey := in.turn.ID + ":" + fmt.Sprint(stepID) + ":" + action.ToolName
	result, err := s.tools.ExecuteAgentTool(ctx, agent.ToolExecuteInput{
		UserID:         in.userID,
		ConversationID: in.conversationID,
		ToolName:       action.ToolName,
		Arguments:      action.Arguments,
		IdempotencyKey: &idempotencyKey,
		DryRun:         action.DryRun,
	})
	if result.ToolCall.ID != "" {
		_ = s.harness.SetLoopStepActionRef(ctx, stepID, in.userID, in.conversationID, &result.ToolCall.ID)
	}
	if err != nil && result.ApprovalRequest != nil {
		_, _ = s.harness.AddEvent(ctx, event(in.turn, "approval_requested", map[string]any{
			"tool_name": action.ToolName,
			"call_id":   result.ToolCall.ID,
		}))
		return agent.Observation{ActionType: agent.ActionToolCall, Text: "工具 " + action.ToolName + " 需要用户审批。", RefID: &result.ToolCall.ID}, true
	}
	if err != nil {
		_, _ = s.harness.AddEvent(ctx, event(in.turn, "tool_call_finished", map[string]any{
			"tool_name": action.ToolName,
			"status":    "failed",
			"error":     err.Error(),
		}))
		return agent.Observation{ActionType: agent.ActionToolCall, Text: "工具 " + action.ToolName + " 执行失败：" + err.Error(), RefID: refOrNil(result.ToolCall.ID), Failed: true}, false
	}
	text := compactObservation(result.Result)
	_, _ = s.harness.AddEvent(ctx, event(in.turn, "tool_call_finished", map[string]any{
		"tool_name": action.ToolName,
		"status":    "success",
		"call_id":   result.ToolCall.ID,
	}))
	return agent.Observation{ActionType: agent.ActionToolCall, Text: "工具 " + action.ToolName + " 返回：" + text, RefID: &result.ToolCall.ID}, false
}

func (s *Service) buildContextMemory(ctx context.Context, userID, conversationID string, messageID *string, cfg store.AgentConfig, query string) []contextmgr.MemoryChunk {
	if s.memories == nil || !cfg.MemoryEnabled {
		return nil
	}
	result, err := s.memories.Retrieve(ctx, memory.RetrieveInput{
		UserID:           userID,
		ConversationID:   conversationID,
		MessageID:        messageID,
		Query:            query,
		MaxMemoryTokens:  1200,
		LogRetrieval:     true,
		SemanticOnDemand: cfg.SemanticMemoryEnabled,
	})
	if err != nil {
		return nil
	}
	chunks := make([]contextmgr.MemoryChunk, 0, len(result.Chunks))
	for _, chunk := range result.Chunks {
		chunks = append(chunks, contextmgr.MemoryChunk{
			Layer:      chunk.Layer,
			RefID:      chunk.RefID,
			Content:    chunk.Content,
			TokenCount: chunk.TokenCount,
			LoadMode:   chunk.LoadMode,
		})
	}
	return chunks
}

func renderMemoryObservation(result memory.RetrieveResult) string {
	if len(result.Chunks) == 0 {
		return "没有检索到相关记忆。"
	}
	parts := make([]string, 0, len(result.Chunks))
	for _, chunk := range result.Chunks {
		parts = append(parts, "["+chunk.Layer+"] "+trimRunes(chunk.Content, 180))
	}
	return strings.Join(parts, "\n")
}

func compactObservation(raw json.RawMessage) string {
	text := string(raw)
	if strings.TrimSpace(text) == "" {
		return "{}"
	}
	return trimRunes(text, 800)
}
