package tool

import (
	"context"
	"encoding/json"
	"strings"

	"freedinner/backend/internal/agent"
	"freedinner/backend/internal/store"
)

func (s *Service) Route(ctx context.Context, input RouteInput) (agent.RouteResult, error) {
	tools, err := s.routableToolsForConversation(ctx, input.UserID, input.ConversationID)
	if err != nil {
		return agent.RouteResult{}, err
	}
	candidates := toToolDescriptors(tools)
	result := agent.RouteTools(input.Query, candidates)
	candidateJSON, _ := json.Marshal(result.Candidates)
	selectedJSON, _ := json.Marshal(result.Selected)
	_ = s.tools.CreateRouterLog(ctx, store.ToolRouterLogCreate{
		UserID:         input.UserID,
		ConversationID: input.ConversationID,
		MessageID:      input.MessageID,
		Query:          input.Query,
		CandidateTools: candidateJSON,
		SelectedTools:  selectedJSON,
		RouteReason:    &result.Reason,
		RiskLevel:      result.RiskLevel,
	})
	return result, nil
}

func (s *Service) routableTools(ctx context.Context, userID string) ([]store.ToolDefinition, error) {
	return s.routableToolsForConversation(ctx, userID, "")
}

func (s *Service) routableToolsForConversation(ctx context.Context, userID, conversationID string) ([]store.ToolDefinition, error) {
	if s.agents != nil {
		cfg, err := s.agents.GetDefault(ctx, userID)
		if err == nil {
			bound, err := s.tools.ListAgentBoundTools(ctx, userID, cfg.ID)
			if err != nil {
				return nil, err
			}
			if len(bound) > 0 {
				return s.filterContextualTools(ctx, userID, conversationID, bound), nil
			}
		}
	}
	tools, err := s.tools.ListTools(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.filterContextualTools(ctx, userID, conversationID, tools), nil
}

func (s *Service) filterContextualTools(ctx context.Context, userID, conversationID string, tools []store.ToolDefinition) []store.ToolDefinition {
	napcatToolsAllowed := s.napcatChannelContext(ctx, userID, conversationID) != nil
	filtered := make([]store.ToolDefinition, 0, len(tools))
	for _, toolDefinition := range tools {
		if strings.EqualFold(toolDefinition.Namespace, "napcatqq") && !napcatToolsAllowed {
			continue
		}
		filtered = append(filtered, toolDefinition)
	}
	return filtered
}

func (s *Service) napcatChannelContext(ctx context.Context, userID, conversationID string) *store.Conversation {
	if s.conversations == nil || strings.TrimSpace(conversationID) == "" {
		return nil
	}
	conversation, err := s.conversations.FindByID(ctx, userID, conversationID)
	if err != nil {
		return nil
	}
	if conversation.Source != "channel" || conversation.ChannelConnectionID == nil || strings.TrimSpace(*conversation.ChannelConnectionID) == "" {
		return nil
	}
	if s.channels == nil {
		return nil
	}
	provider, err := s.channels.FindProviderForConnection(ctx, userID, *conversation.ChannelConnectionID)
	if err != nil || !strings.EqualFold(provider.Name, "napcatqq") {
		return nil
	}
	return &conversation
}

func (s *Service) RouteAgentTools(ctx context.Context, input agent.ToolRouteInput) (agent.RouteResult, error) {
	return s.Route(ctx, RouteInput{
		UserID:         input.UserID,
		ConversationID: input.ConversationID,
		MessageID:      input.MessageID,
		Query:          input.Query,
	})
}

func toToolDescriptors(tools []store.ToolDefinition) []agent.ToolDescriptor {
	result := make([]agent.ToolDescriptor, 0, len(tools))
	for _, toolDefinition := range tools {
		result = append(result, agent.ToolDescriptor{
			ID:               toolDefinition.ID,
			Name:             toolDefinition.Name,
			DisplayName:      toolDefinition.DisplayName,
			Description:      toolDefinition.Description,
			PermissionLevel:  toolDefinition.PermissionLevel,
			RequiresApproval: toolDefinition.RequiresApproval,
			ParameterSchema:  toolDefinition.ParameterSchema,
		})
	}
	return result
}
