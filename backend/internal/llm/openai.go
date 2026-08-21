package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultOpenAIBaseURL = "https://api.openai.com/v1"

type ChatMessage struct {
	Role    string
	Content string
}

type GenerateRequest struct {
	APIKey  string
	BaseURL *string
	Model   string
	Input   []ChatMessage
}

type GenerateResponse struct {
	Text         string
	InputTokens  int
	OutputTokens int
	TotalTokens  int
	LatencyMS    int
}

type EmbedRequest struct {
	APIKey  string
	BaseURL *string
	Model   string
	Input   string
}

type EmbedResponse struct {
	Vector    []float64
	Tokens    int
	LatencyMS int
}

type OpenAIClient struct {
	httpClient *http.Client
}

func NewOpenAIClient() *OpenAIClient {
	return &OpenAIClient{
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (c *OpenAIClient) Generate(ctx context.Context, req GenerateRequest) (GenerateResponse, error) {
	if strings.TrimSpace(req.APIKey) == "" {
		return GenerateResponse{}, errors.New("missing api key")
	}
	if strings.TrimSpace(req.Model) == "" {
		return GenerateResponse{}, errors.New("missing model")
	}

	startedAt := time.Now()
	body := openAIResponsesRequest{
		Model: req.Model,
		Input: make([]openAIInputMessage, 0, len(req.Input)),
	}
	for _, msg := range req.Input {
		if strings.TrimSpace(msg.Content) == "" {
			continue
		}
		body.Input = append(body.Input, openAIInputMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	endpoint := responsesEndpoint(req.BaseURL)
	if isChatCompletionsEndpoint(endpoint) {
		return c.generateChatCompletions(ctx, req, startedAt)
	}

	response, err := c.postResponses(ctx, endpoint, req.APIKey, body, startedAt)
	if err == nil {
		return response, nil
	}
	if req.BaseURL != nil && strings.TrimSpace(*req.BaseURL) != "" {
		return c.generateChatCompletions(ctx, req, startedAt)
	}
	return GenerateResponse{}, err
}

func (c *OpenAIClient) Embed(ctx context.Context, req EmbedRequest) (EmbedResponse, error) {
	if strings.TrimSpace(req.APIKey) == "" {
		return EmbedResponse{}, errors.New("missing api key")
	}
	if strings.TrimSpace(req.Model) == "" {
		return EmbedResponse{}, errors.New("missing embedding model")
	}
	if strings.TrimSpace(req.Input) == "" {
		return EmbedResponse{}, errors.New("missing embedding input")
	}

	startedAt := time.Now()
	body := openAIEmbeddingsRequest{
		Model: req.Model,
		Input: req.Input,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return EmbedResponse{}, err
	}

	endpoint := embeddingsEndpoint(req.BaseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return EmbedResponse{}, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+req.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return EmbedResponse{}, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return EmbedResponse{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return EmbedResponse{}, fmt.Errorf("openai embeddings status %d: %s", resp.StatusCode, string(respBody))
	}

	var decoded openAIEmbeddingsResponse
	if err := json.Unmarshal(respBody, &decoded); err != nil {
		return EmbedResponse{}, err
	}
	if len(decoded.Data) == 0 || len(decoded.Data[0].Embedding) == 0 {
		return EmbedResponse{}, errors.New("openai embeddings did not contain vector")
	}
	return EmbedResponse{
		Vector:    decoded.Data[0].Embedding,
		Tokens:    decoded.Usage.TotalTokens,
		LatencyMS: int(time.Since(startedAt).Milliseconds()),
	}, nil
}

func (c *OpenAIClient) postResponses(ctx context.Context, endpoint, apiKey string, body openAIResponsesRequest, startedAt time.Time) (GenerateResponse, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return GenerateResponse{}, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return GenerateResponse{}, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return GenerateResponse{}, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return GenerateResponse{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return GenerateResponse{}, fmt.Errorf("openai response status %d: %s", resp.StatusCode, string(respBody))
	}

	var decoded openAIResponsesResponse
	if err := json.Unmarshal(respBody, &decoded); err != nil {
		return GenerateResponse{}, err
	}

	text := strings.TrimSpace(decoded.OutputText)
	if text == "" {
		text = strings.TrimSpace(decoded.collectText())
	}
	if text == "" {
		return GenerateResponse{}, errors.New("openai response did not contain output text")
	}

	return GenerateResponse{
		Text:         text,
		InputTokens:  decoded.Usage.InputTokens,
		OutputTokens: decoded.Usage.OutputTokens,
		TotalTokens:  decoded.Usage.TotalTokens,
		LatencyMS:    int(time.Since(startedAt).Milliseconds()),
	}, nil
}

func (c *OpenAIClient) generateChatCompletions(ctx context.Context, req GenerateRequest, startedAt time.Time) (GenerateResponse, error) {
	body := openAIChatCompletionsRequest{
		Model:       req.Model,
		Messages:    make([]openAIChatMessage, 0, len(req.Input)),
		MaxTokens:   1000,
		Temperature: 0.7,
	}
	for _, msg := range req.Input {
		if strings.TrimSpace(msg.Content) == "" {
			continue
		}
		body.Messages = append(body.Messages, openAIChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return GenerateResponse{}, err
	}

	endpoint := chatCompletionsEndpoint(req.BaseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return GenerateResponse{}, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+req.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return GenerateResponse{}, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return GenerateResponse{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return GenerateResponse{}, fmt.Errorf("openai chat completions status %d: %s", resp.StatusCode, string(respBody))
	}

	var decoded openAIChatCompletionsResponse
	if err := json.Unmarshal(respBody, &decoded); err != nil {
		return GenerateResponse{}, err
	}
	if len(decoded.Choices) == 0 {
		return GenerateResponse{}, errors.New("openai chat completions did not contain choices")
	}
	text := strings.TrimSpace(decoded.Choices[0].Message.Content)
	if text == "" {
		return GenerateResponse{}, errors.New("openai chat completions did not contain output text")
	}

	return GenerateResponse{
		Text:         text,
		InputTokens:  decoded.Usage.PromptTokens,
		OutputTokens: decoded.Usage.CompletionTokens,
		TotalTokens:  decoded.Usage.TotalTokens,
		LatencyMS:    int(time.Since(startedAt).Milliseconds()),
	}, nil
}

func responsesEndpoint(baseURL *string) string {
	if baseURL == nil || strings.TrimSpace(*baseURL) == "" {
		return strings.TrimRight(defaultOpenAIBaseURL, "/") + "/responses"
	}
	return strings.TrimRight(strings.TrimSpace(*baseURL), "/") + "/responses"
}

func chatCompletionsEndpoint(baseURL *string) string {
	if baseURL == nil || strings.TrimSpace(*baseURL) == "" {
		return strings.TrimRight(defaultOpenAIBaseURL, "/") + "/chat/completions"
	}
	base := strings.TrimRight(strings.TrimSpace(*baseURL), "/")
	if isChatCompletionsEndpoint(base) {
		return base
	}
	return base + "/chat/completions"
}

func embeddingsEndpoint(baseURL *string) string {
	if baseURL == nil || strings.TrimSpace(*baseURL) == "" {
		return strings.TrimRight(defaultOpenAIBaseURL, "/") + "/embeddings"
	}
	base := strings.TrimRight(strings.TrimSpace(*baseURL), "/")
	if strings.HasSuffix(base, "/embeddings") {
		return base
	}
	return base + "/embeddings"
}

func isChatCompletionsEndpoint(endpoint string) bool {
	return strings.HasSuffix(strings.TrimRight(endpoint, "/"), "/chat/completions")
}

type openAIResponsesRequest struct {
	Model string               `json:"model"`
	Input []openAIInputMessage `json:"input"`
}

type openAIInputMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponsesResponse struct {
	OutputText string         `json:"output_text"`
	Output     []openAIOutput `json:"output"`
	Usage      openAIUsage    `json:"usage"`
}

type openAIOutput struct {
	Content []openAIOutputContent `json:"content"`
}

type openAIOutputContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type openAIUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

type openAIChatCompletionsRequest struct {
	Model       string              `json:"model"`
	Messages    []openAIChatMessage `json:"messages"`
	MaxTokens   int                 `json:"max_tokens"`
	Temperature float64             `json:"temperature"`
}

type openAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatCompletionsResponse struct {
	Choices []openAIChatChoice `json:"choices"`
	Usage   openAIChatUsage    `json:"usage"`
}

type openAIChatChoice struct {
	Message openAIChatMessage `json:"message"`
}

type openAIChatUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type openAIEmbeddingsRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type openAIEmbeddingsResponse struct {
	Data  []openAIEmbeddingData `json:"data"`
	Usage openAIEmbeddingUsage  `json:"usage"`
}

type openAIEmbeddingData struct {
	Embedding []float64 `json:"embedding"`
}

type openAIEmbeddingUsage struct {
	TotalTokens int `json:"total_tokens"`
}

func (r openAIResponsesResponse) collectText() string {
	var builder strings.Builder
	for _, output := range r.Output {
		for _, content := range output.Content {
			if content.Text == "" {
				continue
			}
			if builder.Len() > 0 {
				builder.WriteString("\n")
			}
			builder.WriteString(content.Text)
		}
	}
	return builder.String()
}
