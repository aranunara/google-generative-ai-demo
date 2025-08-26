package valueobjects

import (
	"fmt"
)

type PersonGeneration string
type SafetySetting string
type MimeType string

const (
	AllowAdult PersonGeneration = "allow_adult"
	AllowAll   PersonGeneration = "allow_all"
	DontAllow  PersonGeneration = "dont_allow"
)

const (
	BlockMediumAndAbove SafetySetting = "block_medium_and_above"
	BlockLowAndAbove    SafetySetting = "block_low_and_above"
	BlockOnlyHigh       SafetySetting = "block_only_high"
	BlockNone           SafetySetting = "block_none"
)

const (
	MimeTypePNG  MimeType = "image/png"
	MimeTypeJPEG MimeType = "image/jpeg"
)

type TryOnParameters struct {
	addWatermark       bool
	baseSteps          int
	personGeneration   PersonGeneration
	safetySetting      SafetySetting
	sampleCount        int
	seed               int
	storageURI         string
	outputMimeType     MimeType
	compressionQuality int
}

func NewTryOnParameters(
	addWatermark bool,
	baseSteps int,
	personGeneration PersonGeneration,
	safetySetting SafetySetting,
	sampleCount int,
	seed int,
	storageURI string,
	outputMimeType MimeType,
	compressionQuality int,
) (*TryOnParameters, error) {
	if baseSteps < 1 || baseSteps > 100 {
		return nil, fmt.Errorf("baseSteps must be between 1 and 100, got %d", baseSteps)
	}

	if sampleCount < 1 || sampleCount > 4 {
		return nil, fmt.Errorf("sampleCount must be between 1 and 4, got %d", sampleCount)
	}

	if compressionQuality < 0 || compressionQuality > 100 {
		return nil, fmt.Errorf("compressionQuality must be between 0 and 100, got %d", compressionQuality)
	}

	return &TryOnParameters{
		addWatermark:       addWatermark,
		baseSteps:          baseSteps,
		personGeneration:   personGeneration,
		safetySetting:      safetySetting,
		sampleCount:        sampleCount,
		seed:               seed,
		storageURI:         storageURI,
		outputMimeType:     outputMimeType,
		compressionQuality: compressionQuality,
	}, nil
}

func DefaultTryOnParameters() *TryOnParameters {
	params, _ := NewTryOnParameters(
		true,
		32,
		AllowAdult,
		BlockMediumAndAbove,
		1,
		0,
		"",
		MimeTypePNG,
		75,
	)
	return params
}

func (p *TryOnParameters) AddWatermark() bool {
	return p.addWatermark
}

func (p *TryOnParameters) BaseSteps() int {
	return p.baseSteps
}

func (p *TryOnParameters) PersonGeneration() PersonGeneration {
	return p.personGeneration
}

func (p *TryOnParameters) SafetySetting() SafetySetting {
	return p.safetySetting
}

func (p *TryOnParameters) SampleCount() int {
	return p.sampleCount
}

func (p *TryOnParameters) Seed() int {
	return p.seed
}

func (p *TryOnParameters) StorageURI() string {
	return p.storageURI
}

func (p *TryOnParameters) OutputMimeType() MimeType {
	return p.outputMimeType
}

func (p *TryOnParameters) CompressionQuality() int {
	return p.compressionQuality
}