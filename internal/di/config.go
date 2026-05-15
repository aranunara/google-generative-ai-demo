package di

import (
	"errors"
	"os"
)

// Config はサーバ起動に必要な設定値を環境変数から読み込んだもの。
type Config struct {
	GenaiAPIKey string
	ProjectID   string
	Location    string
	VTOModel    string
	Port        string
}

// LoadConfig は環境変数から設定を読み込み、デフォルト適用と必須検証を行う。
// 必須項目が未設定の場合は現行と同一の文言の error を返す。
func LoadConfig() (*Config, error) {
	genaiAPIKey := os.Getenv("GEMINI_API_KEY")
	if genaiAPIKey == "" {
		return nil, errors.New("環境変数 GEMINI_API_KEY が未設定です")
	}

	projectID := os.Getenv("PROJECT_ID")
	if projectID == "" {
		return nil, errors.New("環境変数 PROJECT_ID が未設定です")
	}

	location := os.Getenv("LOCATION")
	if location == "" {
		location = "us-central1"
	}

	vtoModel := os.Getenv("VTO_MODEL")
	if vtoModel == "" {
		vtoModel = "virtual-try-on-preview-08-04"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	return &Config{
		GenaiAPIKey: genaiAPIKey,
		ProjectID:   projectID,
		Location:    location,
		VTOModel:    vtoModel,
		Port:        port,
	}, nil
}
