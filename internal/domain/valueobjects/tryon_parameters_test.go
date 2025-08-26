package valueobjects

import (
	"testing"
)

func TestNewTryOnParameters(t *testing.T) {
	tests := []struct {
		name               string
		baseSteps          int
		sampleCount        int
		compressionQuality int
		wantErr            bool
	}{
		{
			name:               "valid parameters",
			baseSteps:          32,
			sampleCount:        1,
			compressionQuality: 75,
			wantErr:            false,
		},
		{
			name:               "baseSteps too low",
			baseSteps:          0,
			sampleCount:        1,
			compressionQuality: 75,
			wantErr:            true,
		},
		{
			name:               "baseSteps too high",
			baseSteps:          101,
			sampleCount:        1,
			compressionQuality: 75,
			wantErr:            true,
		},
		{
			name:               "sampleCount too low",
			baseSteps:          32,
			sampleCount:        0,
			compressionQuality: 75,
			wantErr:            true,
		},
		{
			name:               "sampleCount too high",
			baseSteps:          32,
			sampleCount:        5,
			compressionQuality: 75,
			wantErr:            true,
		},
		{
			name:               "compressionQuality too low",
			baseSteps:          32,
			sampleCount:        1,
			compressionQuality: -1,
			wantErr:            true,
		},
		{
			name:               "compressionQuality too high",
			baseSteps:          32,
			sampleCount:        1,
			compressionQuality: 101,
			wantErr:            true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewTryOnParameters(
				true,
				tt.baseSteps,
				AllowAdult,
				BlockMediumAndAbove,
				tt.sampleCount,
				0,
				"",
				MimeTypePNG,
				tt.compressionQuality,
			)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewTryOnParameters() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultTryOnParameters(t *testing.T) {
	params := DefaultTryOnParameters()
	
	if params.BaseSteps() != 32 {
		t.Errorf("Expected BaseSteps 32, got %d", params.BaseSteps())
	}
	
	if params.SampleCount() != 1 {
		t.Errorf("Expected SampleCount 1, got %d", params.SampleCount())
	}
	
	if params.PersonGeneration() != AllowAdult {
		t.Errorf("Expected PersonGeneration AllowAdult, got %v", params.PersonGeneration())
	}
	
	if params.SafetySetting() != BlockMediumAndAbove {
		t.Errorf("Expected SafetySetting BlockMediumAndAbove, got %v", params.SafetySetting())
	}
	
	if params.OutputMimeType() != MimeTypePNG {
		t.Errorf("Expected OutputMimeType PNG, got %v", params.OutputMimeType())
	}
}