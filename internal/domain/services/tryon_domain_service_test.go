package services

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/jpeg"
	"strings"
	"testing"

	"tryon-demo/internal/domain/entities"
	"tryon-demo/internal/domain/valueobjects"
)

type mockAIService struct {
	result *entities.TryOnResult
	err    error
}

func (m *mockAIService) GenerateTryOn(ctx context.Context, request *entities.TryOnRequest) (*entities.TryOnResult, error) {
	return m.result, m.err
}

func TestTryOnDomainService_ProcessTryOn(t *testing.T) {
	personImage := createTestImageData(t)
	garmentImage := createTestImageData(t)
	
	validRequest, err := entities.NewTryOnRequest(personImage, garmentImage, nil)
	if err != nil {
		t.Fatalf("Failed to create valid request: %v", err)
	}

	t.Run("successful processing", func(t *testing.T) {
		mockAI := &mockAIService{
			result: entities.NewTryOnResult(validRequest.ID(), []*valueobjects.ImageData{personImage}),
			err:    nil,
		}
		
		service := NewTryOnDomainService(mockAI)
		result, err := service.ProcessTryOn(context.Background(), validRequest)
		
		if err != nil {
			t.Errorf("ProcessTryOn() error = %v", err)
		}
		if result == nil {
			t.Errorf("Expected result, got nil")
		}
		if !result.HasImages() {
			t.Errorf("Result should have images")
		}
	})

	t.Run("AI service error", func(t *testing.T) {
		mockAI := &mockAIService{
			result: nil,
			err:    errors.New("AI service failed"),
		}
		
		service := NewTryOnDomainService(mockAI)
		result, err := service.ProcessTryOn(context.Background(), validRequest)
		
		if err == nil {
			t.Errorf("Expected error, got nil")
		}
		if result != nil {
			t.Errorf("Expected nil result on error")
		}
	})

	t.Run("quota error handling", func(t *testing.T) {
		mockAI := &mockAIService{
			result: nil,
			err:    errors.New("quota exceeded"),
		}
		
		service := NewTryOnDomainService(mockAI)
		result, err := service.ProcessTryOn(context.Background(), validRequest)
		
		if err == nil {
			t.Errorf("Expected error, got nil")
		}
		if result != nil {
			t.Errorf("Expected nil result on error")
		}
		
		if !strings.Contains(err.Error(), "service temporarily unavailable due to high demand") {
			t.Errorf("Expected quota error message, got %v", err.Error())
		}
	})

	t.Run("no images generated", func(t *testing.T) {
		mockAI := &mockAIService{
			result: entities.NewTryOnResult(validRequest.ID(), []*valueobjects.ImageData{}),
			err:    nil,
		}
		
		service := NewTryOnDomainService(mockAI)
		result, err := service.ProcessTryOn(context.Background(), validRequest)
		
		if err == nil {
			t.Errorf("Expected error for no images")
		}
		if result != nil {
			t.Errorf("Expected nil result when no images generated")
		}
	})
}

func createTestImageData(t *testing.T) *valueobjects.ImageData {
	// Create minimal valid JPEG bytes
	jpegBytes := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0xFF, 0xD9}
	
	imageData, err := valueobjects.NewImageData(jpegBytes)
	if err != nil {
		// If that fails, create a simple test image
		img := image.NewRGBA(image.Rect(0, 0, 1, 1))
		var buf bytes.Buffer
		jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90})
		imageData, err = valueobjects.NewImageData(buf.Bytes())
		if err != nil {
			t.Fatalf("Failed to create test image data: %v", err)
		}
	}
	
	return imageData
}

