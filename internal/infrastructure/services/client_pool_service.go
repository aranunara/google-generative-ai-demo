package services

import (
	"context"
	"fmt"
	"sync"

	"cloud.google.com/go/vertexai/genai" // VertexAI用
	"google.golang.org/api/option"
	genai_std "google.golang.org/genai" // 標準GenAI用

	"tryon-demo/internal/domain/repositories"
)

// VertexAI Client Pool実装
type vertexAIClientPool struct {
	config *repositories.AIClientConfig
	client *genai.Client
	mutex  sync.RWMutex
}

// 新しいVertexAIクライアントプールを作成
func newVertexAIClientPool(config *repositories.AIClientConfig) repositories.VertexAIClientPool {
	return &vertexAIClientPool{
		config: config,
	}
}

func (p *vertexAIClientPool) GetVertexAIClient(ctx context.Context) (*genai.Client, error) {
	p.mutex.RLock()
	if p.client != nil {
		defer p.mutex.RUnlock()
		return p.client, nil
	}
	p.mutex.RUnlock()

	p.mutex.Lock()
	defer p.mutex.Unlock()

	// ダブルチェックロッキング
	if p.client != nil {
		return p.client, nil
	}

	// VertexAI クライアントを作成
	endpoint := fmt.Sprintf("%s-aiplatform.googleapis.com:443", p.config.Location)
	client, err := genai.NewClient(ctx, p.config.ProjectID, p.config.Location, option.WithEndpoint(endpoint))
	if err != nil {
		return nil, fmt.Errorf("failed to create VertexAI client: %w", err)
	}

	p.client = client
	return p.client, nil
}

func (p *vertexAIClientPool) Close() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.client != nil {
		err := p.client.Close()
		p.client = nil
		return err
	}
	return nil
}

// GenAI Client Pool実装
type genAIClientPool struct {
	config *repositories.AIClientConfig
	client *genai_std.Client
	mutex  sync.RWMutex
}

// 新しいGenAIクライアントプールを作成
func newGenAIClientPool(config *repositories.AIClientConfig) repositories.GenAIClientPool {
	return &genAIClientPool{
		config: config,
	}
}

func (p *genAIClientPool) GetGenAIClient(
	ctx context.Context,
	geminiApiKey string,
) (*genai_std.Client, error) {
	p.mutex.RLock()
	if p.client != nil {
		defer p.mutex.RUnlock()
		return p.client, nil
	}
	p.mutex.RUnlock()

	p.mutex.Lock()
	defer p.mutex.Unlock()

	// ダブルチェックロッキング
	if p.client != nil {
		return p.client, nil
	}

	// 標準GenAI クライアントを作成
	client, err := genai_std.NewClient(ctx, &genai_std.ClientConfig{
		APIKey: geminiApiKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create GenAI client: %w", err)
	}

	p.client = client

	return p.client, nil
}

func (p *genAIClientPool) Close() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.client != nil {
		// GenAI Clientはリソースクリーンアップ不要
		p.client = nil
	}
	return nil
}

// Client Pool Service実装
type clientPoolService struct {
	config       *repositories.AIClientConfig
	vertexAIPool repositories.VertexAIClientPool
	genAIPool    repositories.GenAIClientPool
}

// 新しいClient Pool Serviceを作成
func NewClientPoolService(projectID, location string) repositories.ClientPoolService {
	config := &repositories.AIClientConfig{
		ProjectID: projectID,
		Location:  location,
	}

	return &clientPoolService{
		config:       config,
		vertexAIPool: newVertexAIClientPool(config),
		genAIPool:    newGenAIClientPool(config),
	}
}

func (s *clientPoolService) VertexAIPool() repositories.VertexAIClientPool {
	return s.vertexAIPool
}

func (s *clientPoolService) GenAIPool() repositories.GenAIClientPool {
	return s.genAIPool
}

func (s *clientPoolService) Config() *repositories.AIClientConfig {
	return s.config
}

func (s *clientPoolService) Close() error {
	var errs []error

	if err := s.vertexAIPool.Close(); err != nil {
		errs = append(errs, fmt.Errorf("VertexAI pool close error: %w", err))
	}

	if err := s.genAIPool.Close(); err != nil {
		errs = append(errs, fmt.Errorf("GenAI pool close error: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("client pool close errors: %v", errs)
	}

	return nil
}
