package services

import (
	"net/http"
	"strconv"

	"tryon-demo/internal/application/usecases"
)

type ParameterService struct{}

func NewParameterService() *ParameterService {
	return &ParameterService{}
}

func (s *ParameterService) ParseFromRequest(r *http.Request) *usecases.TryOnParametersInput {
	params := &usecases.TryOnParametersInput{
		AddWatermark:       s.getBool(r, "add_watermark", true),
		BaseSteps:          s.getInt(r, "base_steps", 32, 1, 100),
		PersonGeneration:   s.getString(r, "person_generation", "allow_adult"),
		SafetySetting:      s.getString(r, "safety_setting", "block_medium_and_above"),
		SampleCount:        s.getInt(r, "sample_count", 1, 1, 4),
		Seed:               s.getInt(r, "seed", 0, 0, 0),
		StorageURI:         s.getString(r, "storage_uri", ""),
		OutputMimeType:     s.getString(r, "output_mime_type", "image/png"),
		CompressionQuality: s.getInt(r, "compression_quality", 75, 0, 100),
	}

	// PNG選択時はCompressionQualityを0に設定（APIの制限）
	if params.OutputMimeType != "image/jpeg" {
		params.CompressionQuality = 0
	}

	// Watermarkが有効な場合、Seedを0に設定（APIの制限）
	if params.AddWatermark {
		params.Seed = 0
	}

	return params
}

func (s *ParameterService) getBool(r *http.Request, key string, defaultValue bool) bool {
	value := r.FormValue(key)
	if value == "" {
		return defaultValue
	}
	return value == "true"
}

func (s *ParameterService) getInt(r *http.Request, key string, defaultValue, min, max int) int {
	value := r.FormValue(key)
	if value == "" {
		return defaultValue
	}

	intVal, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}

	if max > 0 && (intVal < min || intVal > max) {
		return defaultValue
	}

	if min > 0 && intVal < min {
		return defaultValue
	}

	return intVal
}

func (s *ParameterService) getString(r *http.Request, key, defaultValue string) string {
	value := r.FormValue(key)
	if value == "" {
		return defaultValue
	}
	return value
}