package entities

import (
	"bytes"
	"image"
	"image/jpeg"
	"testing"

	"tryon-demo/internal/domain/valueobjects"
)

func createTestImageData(t *testing.T) *valueobjects.ImageData {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	var buf bytes.Buffer
	err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90})
	if err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}

	imageData, err := valueobjects.NewImageData(buf.Bytes(), "image/jpeg")
	if err != nil {
		t.Fatalf("Failed to create ImageData: %v", err)
	}

	return imageData
}

func TestNewTryOnRequest(t *testing.T) {
	personImage := createTestImageData(t)
	garmentImage := createTestImageData(t)
	params := valueobjects.DefaultTryOnParameters()

	tests := []struct {
		name         string
		personImage  *valueobjects.ImageData
		garmentImage *valueobjects.ImageData
		parameters   *valueobjects.TryOnParameters
		wantErr      bool
	}{
		{
			name:         "valid request",
			personImage:  personImage,
			garmentImage: garmentImage,
			parameters:   params,
			wantErr:      false,
		},
		{
			name:         "nil person image should fail",
			personImage:  nil,
			garmentImage: garmentImage,
			parameters:   params,
			wantErr:      true,
		},
		{
			name:         "nil garment image should fail",
			personImage:  personImage,
			garmentImage: nil,
			parameters:   params,
			wantErr:      true,
		},
		{
			name:         "nil parameters should use default",
			personImage:  personImage,
			garmentImage: garmentImage,
			parameters:   nil,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request, err := NewTryOnRequest(tt.personImage, tt.garmentImage, tt.parameters)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewTryOnRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if request.ID() == "" {
					t.Errorf("Expected non-empty ID")
				}
				if request.PersonImage() != tt.personImage {
					t.Errorf("PersonImage not set correctly")
				}
				if request.GarmentImage() != tt.garmentImage {
					t.Errorf("GarmentImage not set correctly")
				}
				if request.Parameters() == nil {
					t.Errorf("Parameters should not be nil")
				}
			}
		})
	}
}

func TestTryOnRequest_PrepareImages(t *testing.T) {
	personImage := createTestImageData(t)
	garmentImage := createTestImageData(t)

	request, err := NewTryOnRequest(personImage, garmentImage, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	err = request.PrepareImages()
	if err != nil {
		t.Errorf("PrepareImages() error = %v", err)
	}

	if !request.PersonImage().IsJPEG() {
		t.Errorf("Person image should be JPEG after preparation")
	}

	if !request.GarmentImage().IsJPEG() {
		t.Errorf("Garment image should be JPEG after preparation")
	}
}
