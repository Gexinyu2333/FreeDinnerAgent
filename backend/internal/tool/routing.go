package tool

import (
	"context"
	"encoding/json"

	"freedinner/backend/internal/agent"
	"freedinner/backend/internal/store"
)

func (s *Service) Route(ctx context.Context, input RouteInput) (agent.RouteResult, error) {
	tools, err := s.routableTools(ctx, input.UserID)
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
	if s.agents != nil {
		cfg, err := s.agents.GetDefault(ctx, userID)
		if err == nil {
			bound, err := s.tools.ListAgentBoundTools(ctx, userID, cfg.ID)
			if err != nil {
				return nil, err
			}
			if len(bound) > 0 {
				return bound, nil
			}
		}
	}
	return s.tools.ListTools(ctx, userID)
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
