package external

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"cloud.google.com/go/vertexai/genai"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"

	"tryon-demo/internal/domain/entities"
	"tryon-demo/internal/domain/valueobjects"
	"tryon-demo/model"
)

type VertexAIService struct {
	projectID string
	location  string
	vtoModel  string
	client    *genai.Client
	useSDK    bool
}

func NewVertexAIService(projectID, location, vtoModel string, useSDK bool) (*VertexAIService, error) {
	var client *genai.Client
	if useSDK {
		ctx := context.Background()
		endpoint := fmt.Sprintf("%s-aiplatform.googleapis.com:443", location)
		var err error
		client, err = genai.NewClient(ctx, projectID, location, option.WithEndpoint(endpoint))
		if err != nil {
			return nil, fmt.Errorf("failed to create genai client: %w", err)
		}
	}

	return &VertexAIService{
		projectID: projectID,
		location:  location,
		vtoModel:  vtoModel,
		client:    client,
		useSDK:    useSDK,
	}, nil
}

func (s *VertexAIService) GenerateTryOn(ctx context.Context, request *entities.TryOnRequest) (*entities.TryOnResult, error) {
	if s.useSDK {
		return s.generateWithSDK(ctx, request)
	}
	return s.generateWithREST(ctx, request)
}

func (s *VertexAIService) generateWithSDK(ctx context.Context, request *entities.TryOnRequest) (*entities.TryOnResult, error) {
	model := s.client.GenerativeModel(s.vtoModel)

	personPart := genai.ImageData("image/jpeg", request.PersonImage().Data())
	garmentPart := genai.ImageData("image/jpeg", request.GarmentImage().Data())

	prompt := []genai.Part{
		genai.Text("person:"),
		personPart,
		genai.Text("garment:"),
		garmentPart,
	}

	model.SetTemperature(0.4)
	model.SetTopK(32)
	model.SetTopP(1)
	model.SetMaxOutputTokens(2048)
	model.ResponseMIMEType = "image/jpeg"

	resp, err := model.GenerateContent(ctx, prompt...)
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	if len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates in response")
	}

	candidate := resp.Candidates[0]
	if candidate.Content == nil || len(candidate.Content.Parts) == 0 {
		return nil, fmt.Errorf("no content in response")
	}

	for _, part := range candidate.Content.Parts {
		if blob, ok := part.(genai.Blob); ok {
			if blob.MIMEType == "image/jpeg" || blob.MIMEType == "image/png" {
				imageData, err := valueobjects.NewImageData(blob.Data)
				if err != nil {
					return nil, fmt.Errorf("failed to create image data: %w", err)
				}
				return entities.NewTryOnResult(request.ID(), []*valueobjects.ImageData{imageData}), nil
			}
		}
	}

	return nil, fmt.Errorf("no image found in response")
}

func (s *VertexAIService) generateWithREST(ctx context.Context, request *entities.TryOnRequest) (*entities.TryOnResult, error) {
	accessToken, err := s.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	personB64 := request.PersonImage().ToBase64()
	garmentB64 := request.GarmentImage().ToBase64()

	params := request.Parameters()

	// outputOptionsを構築（CompressionQualityは条件付きで追加）
	outputOptions := map[string]interface{}{
		"mimeType": string(params.OutputMimeType()),
	}

	// CompressionQualityが0より大きい場合のみ追加
	if params.CompressionQuality() > 0 {
		outputOptions["compressionQuality"] = params.CompressionQuality()
	}

	// parametersを構築（条件付きパラメータは後から追加）
	parameters := map[string]interface{}{
		"addWatermark":     params.AddWatermark(),
		"baseSteps":        params.BaseSteps(),
		"personGeneration": string(params.PersonGeneration()),
		"safetySetting":    string(params.SafetySetting()),
		"sampleCount":      params.SampleCount(),
		"outputOptions":    outputOptions,
	}

	// Storage URIが空でない場合のみ追加
	if params.StorageURI() != "" {
		parameters["storageUri"] = params.StorageURI()
	}

	// Watermarkが無効かつSeedが0より大きい場合のみSeedを追加
	if !params.AddWatermark() && params.Seed() > 0 {
		parameters["seed"] = params.Seed()
	}

	apiRequest := map[string]interface{}{
		"instances": []map[string]interface{}{
			{
				"personImage": map[string]interface{}{
					"image": map[string]interface{}{
						"bytesBase64Encoded": personB64,
					},
				},
				"productImages": []map[string]interface{}{
					{
						"image": map[string]interface{}{
							"bytesBase64Encoded": garmentB64,
						},
					},
				},
			},
		},
		"parameters": parameters,
	}

	reqBody, err := json.Marshal(apiRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// デバッグ用：リクエストの詳細をログ出力（画像データは除く）
	debugRequest := make(map[string]interface{})
	for k, v := range apiRequest {
		if k == "instances" {
			// 画像データを除いたデバッグ用の簡略版
			debugRequest[k] = "【画像データ省略】"
		} else {
			debugRequest[k] = v
		}
	}
	debugJSON, _ := json.MarshalIndent(debugRequest, "", "  ")
	fmt.Printf("[DEBUG] API Request (without image data): %s\n", string(debugJSON))

	url := fmt.Sprintf("https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/google/models/%s:predict",
		s.location, s.projectID, s.location, s.vtoModel)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 300 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	predResp, err := s.parseResponse(respBody)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(predResp.Predictions) == 0 {
		return nil, fmt.Errorf("no predictions in response")
	}

	// Storage URIが指定されている場合の処理
	if params.StorageURI() != "" {
		// Storage URI指定時は200レスポンスが返ってきた時点で成功
		// 空のImageDataリストでTryOnResultを作成（フロントエンド側で保存成功を表示）
		log.Printf("[INFO] Images saved to Storage URI: %s", params.StorageURI())
		return entities.NewTryOnResult(request.ID(), []*valueobjects.ImageData{}), nil
	}

	// 通常の画像データ処理（Storage URI未指定時）
	var images []*valueobjects.ImageData
	for i, prediction := range predResp.Predictions {
		imageB64 := prediction.BytesBase64Encoded
		if imageB64 == "" {
			continue
		}

		imageBytes, err := base64.StdEncoding.DecodeString(imageB64)
		if err != nil {
			continue
		}

		imageData, err := valueobjects.NewImageData(imageBytes)
		if err != nil {
			continue
		}

		images = append(images, imageData)
		_ = i // avoid unused variable
	}

	if len(images) == 0 {
		return nil, fmt.Errorf("no valid image data found in response")
	}

	return entities.NewTryOnResult(request.ID(), images), nil
}

func (s *VertexAIService) getAccessToken(ctx context.Context) (string, error) {
	creds, err := google.FindDefaultCredentials(ctx,
		"https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return "", fmt.Errorf("failed to find default credentials: %w", err)
	}

	token, err := creds.TokenSource.Token()
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}

	return token.AccessToken, nil
}

func (s *VertexAIService) parseResponse(data []byte) (*model.VirtualTryOnResponse, error) {
	var response model.VirtualTryOnResponse
	err := json.Unmarshal(data, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func (s *VertexAIService) Close() error {
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}
