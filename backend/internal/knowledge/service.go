package knowledge

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"unicode/utf8"

	"freedinner/backend/internal/llm"
	"freedinner/backend/internal/secret"
	"freedinner/backend/internal/store"
)

const (
	defaultChunkRuneSize = 1200
	defaultSearchLimit   = 8
)

var ErrEmbeddingProviderRequired = errors.New("embedding provider required")

type Service struct {
	knowledge      *store.KnowledgeStore
	agentConfigs   *store.AgentConfigStore
	modelProviders *store.ModelProviderStore
	crypto         secret.Crypto
	openai         *llm.OpenAIClient
}

type IngestInput struct {
	UserID     string
	Title      string
	Content    string
	SourceType string
	SourceURI  *string
	Visibility string
}

type IngestResult struct {
	Document        store.KnowledgeDocument `json:"document"`
	Chunks          []store.KnowledgeChunk  `json:"chunks"`
	EmbeddingStatus string                  `json:"embedding_status"`
	EmbeddingError  *string                 `json:"embedding_error,omitempty"`
}

type SearchResult struct {
	Mode   string                 `json:"mode"`
	Chunks []store.KnowledgeChunk `json:"chunks"`
}

func NewService(
	knowledgeStore *store.KnowledgeStore,
	agentConfigs *store.AgentConfigStore,
	modelProviders *store.ModelProviderStore,
	crypto secret.Crypto,
	openai *llm.OpenAIClient,
) *Service {
	return &Service{
		knowledge:      knowledgeStore,
		agentConfigs:   agentConfigs,
		modelProviders: modelProviders,
		crypto:         crypto,
		openai:         openai,
	}
}

func (s *Service) Ingest(ctx context.Context, input IngestInput) (IngestResult, error) {
	cfg, err := s.agentConfigs.GetDefault(ctx, input.UserID)
	if err != nil {
		return IngestResult{}, err
	}

	visibility := normalizeVisibility(input.Visibility)
	sourceType := normalizeSourceType(input.SourceType)
	rawChunks := splitText(input.Content, defaultChunkRuneSize)
	embeddingStatus := "disabled"
	var embeddingError *string

	var provider store.ModelProvider
	var embeddingAPIKey string
	canEmbed := cfg.EmbeddingEnabled
	if canEmbed {
		provider, err = s.modelProviders.FindDefault(ctx, input.UserID)
		if err != nil || provider.DefaultEmbeddingModel == nil || provider.EncryptedEmbeddingAPIKey == nil || *provider.EncryptedEmbeddingAPIKey == "" {
			canEmbed = false
			embeddingStatus = "skipped_missing_provider"
		} else if provider.Provider != "openai" {
			canEmbed = false
			embeddingStatus = "skipped_unsupported_provider"
		} else {
			embeddingAPIKey, err = s.crypto.Decrypt(*provider.EncryptedEmbeddingAPIKey)
			if err != nil {
				canEmbed = false
				embeddingStatus = "failed"
				message := err.Error()
				embeddingError = &message
			} else {
				embeddingStatus = "generated"
			}
		}
	}

	chunks := make([]store.KnowledgeChunkCreate, 0, len(rawChunks))
	for index, content := range rawChunks {
		metadata, _ := json.Marshal(map[string]any{
			"chunk_runes": utf8.RuneCountInString(content),
		})
		chunk := store.KnowledgeChunkCreate{
			UserID:     input.UserID,
			Visibility: visibility,
			ChunkIndex: index,
			Content:    content,
			TokenCount: estimateTokens(content),
			Metadata:   metadata,
		}
		if canEmbed {
			response, err := s.openai.Embed(ctx, llm.EmbedRequest{
				APIKey:  embeddingAPIKey,
				BaseURL: provider.EmbeddingBaseURL,
				Model:   *provider.DefaultEmbeddingModel,
				Input:   content,
			})
			if err != nil {
				canEmbed = false
				embeddingStatus = "failed"
				message := err.Error()
				embeddingError = &message
			} else {
				vector := store.VectorLiteral(response.Vector)
				chunk.Embedding = &vector
			}
		}
		chunks = append(chunks, chunk)
	}

	metadata, _ := json.Marshal(map[string]any{
		"chunk_count": len(chunks),
	})
	document, createdChunks, err := s.knowledge.CreateDocument(ctx, store.KnowledgeDocumentCreate{
		UserID:      input.UserID,
		Title:       input.Title,
		SourceType:  sourceType,
		SourceURI:   input.SourceURI,
		Visibility:  visibility,
		ContentHash: contentHash(input.Content),
		Metadata:    metadata,
	}, chunks)
	if err != nil {
		return IngestResult{}, err
	}

	return IngestResult{
		Document:        document,
		Chunks:          createdChunks,
		EmbeddingStatus: embeddingStatus,
		EmbeddingError:  embeddingError,
	}, nil
}

func (s *Service) Search(ctx context.Context, userID, query string, limit int) (SearchResult, error) {
	if limit <= 0 || limit > 20 {
		limit = defaultSearchLimit
	}

	cfg, err := s.agentConfigs.GetDefault(ctx, userID)
	if err != nil {
		return SearchResult{}, err
	}

	if cfg.EmbeddingEnabled {
		provider, err := s.modelProviders.FindDefault(ctx, userID)
		if err == nil && provider.Provider == "openai" && provider.DefaultEmbeddingModel != nil &&
			provider.EncryptedEmbeddingAPIKey != nil && *provider.EncryptedEmbeddingAPIKey != "" {
			apiKey, err := s.crypto.Decrypt(*provider.EncryptedEmbeddingAPIKey)
			if err == nil {
				embedding, err := s.openai.Embed(ctx, llm.EmbedRequest{
					APIKey:  apiKey,
					BaseURL: provider.EmbeddingBaseURL,
					Model:   *provider.DefaultEmbeddingModel,
					Input:   query,
				})
				if err == nil {
					chunks, err := s.knowledge.VectorSearch(ctx, userID, store.VectorLiteral(embedding.Vector), limit)
					if err != nil {
						return SearchResult{}, err
					}
					return SearchResult{Mode: "vector", Chunks: chunks}, nil
				}
			}
		}
	}

	chunks, err := s.knowledge.KeywordSearch(ctx, userID, query, limit)
	if err != nil {
		return SearchResult{}, err
	}
	return SearchResult{Mode: "keyword", Chunks: chunks}, nil
}

func (s *Service) ListDocuments(ctx context.Context, userID string) ([]store.KnowledgeDocument, error) {
	return s.knowledge.ListDocuments(ctx, userID)
}

func splitText(content string, maxRunes int) []string {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil
	}
	runes := []rune(content)
	chunks := make([]string, 0, len(runes)/maxRunes+1)
	for start := 0; start < len(runes); start += maxRunes {
		end := start + maxRunes
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, strings.TrimSpace(string(runes[start:end])))
	}
	return chunks
}

func estimateTokens(content string) int {
	runes := utf8.RuneCountInString(content)
	if runes == 0 {
		return 0
	}
	return runes/4 + 1
}

func contentHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func normalizeVisibility(value string) string {
	if value == "public" {
		return "public"
	}
	return "private"
}

func normalizeSourceType(value string) string {
	switch value {
	case "upload", "url", "note", "manual":
		return value
	default:
		return "manual"
	}
}
