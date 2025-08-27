package valueobjects

import (
	"bytes"
	"image"
	"image/jpeg"
	"testing"
)

func TestNewImageData(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name:    "empty data should fail",
			data:    []byte{},
			wantErr: true,
		},
		{
			name:    "nil data should fail",
			data:    nil,
			wantErr: true,
		},
		{
			name:    "invalid image data should fail",
			data:    []byte{0x00, 0x01, 0x02},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewImageData(tt.data, "image/jpeg")
			if (err != nil) != tt.wantErr {
				t.Errorf("NewImageData() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestImageData_ToJPEG(t *testing.T) {
	// Create a simple test image
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	var buf bytes.Buffer
	err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90})
	if err != nil {
		t.Fatalf("Failed to create test JPEG: %v", err)
	}

	imageData, err := NewImageData(buf.Bytes(), "image/jpeg")
	if err != nil {
		t.Fatalf("Failed to create ImageData: %v", err)
	}

	t.Run("JPEG to JPEG should return same instance", func(t *testing.T) {
		result, err := imageData.ToJPEG()
		if err != nil {
			t.Errorf("ToJPEG() error = %v", err)
		}
		if result != imageData {
			t.Errorf("Expected same instance for JPEG to JPEG conversion")
		}
	})

	t.Run("format should be JPEG", func(t *testing.T) {
		if imageData.Format() != JPEG {
			t.Errorf("Expected format JPEG, got %v", imageData.Format())
		}
	})

	t.Run("IsJPEG should return true", func(t *testing.T) {
		if !imageData.IsJPEG() {
			t.Errorf("IsJPEG() should return true for JPEG image")
		}
	})
}
