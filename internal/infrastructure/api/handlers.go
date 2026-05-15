package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"tryon-demo/internal/application/services"
	"tryon-demo/internal/application/usecases"
)

const maxFileSize = 10 * 1024 * 1024 // 10MB

type TryOnHandler struct {
	tryOnUseCase     *usecases.TryOnUseCase
	parameterService *services.ParameterService
	location         string // Vertex AIのリージョン情報
}

type ImagenHandler struct {
	imagenUseCase *usecases.ImagenUseCase
	location      string // Vertex AIのリージョン情報
}

type VeoHandler struct {
	veoUseCase *usecases.VeoUseCase
	location   string // Vertex AIのリージョン情報
}

// ImagenModel represents an available Imagen model
type ImagenModel struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// VeoModel represents an available Veo model
type VeoModel struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// サポートされるImagenモデル一覧
var supportedImagenModels = []ImagenModel{
	{
		ID:          "imagen-4.0-ultra-generate-001",
		Name:        "Imagen 4.0 Ultra",
		Description: "最高品質・最新モデル（処理時間長）",
	},
	{
		ID:          "imagen-4.0-fast-generate-001",
		Name:        "Imagen 4.0 Fast",
		Description: "高品質・高速処理",
	},
	{
		ID:          "imagen-4.0-generate-001",
		Name:        "Imagen 4.0",
		Description: "高品質・標準処理",
	},
	{
		ID:          "imagen-3.0-generate-002",
		Name:        "Imagen 3.0 v2",
		Description: "安定版（推奨）",
	},
}

func NewTryOnHandler(
	tryOnUseCase *usecases.TryOnUseCase,
	parameterService *services.ParameterService,
	location string,
) *TryOnHandler {
	return &TryOnHandler{
		tryOnUseCase:     tryOnUseCase,
		parameterService: parameterService,
		location:         location,
	}
}

func NewImagenHandler(
	imagenUseCase *usecases.ImagenUseCase,
	location string,
) *ImagenHandler {
	return &ImagenHandler{
		imagenUseCase: imagenUseCase,
		location:      location,
	}
}

func NewVeoHandler(
	veoUseCase *usecases.VeoUseCase,
	location string,
) *VeoHandler {
	return &VeoHandler{
		veoUseCase: veoUseCase,
		location:   location,
	}
}

// isValidImagenModel - 指定されたモデルIDが有効かどうかチェック
func (h *ImagenHandler) isValidImagenModel(modelID string) bool {
	for _, model := range supportedImagenModels {
		if model.ID == modelID {
			return true
		}
	}
	return false
}

// getDefaultImagenModel - デフォルトのImagenモデルIDを取得
func (h *ImagenHandler) getDefaultImagenModel() string {
	return "imagen-3.0-generate-002" // 安定版を推奨
}

// 画像生成を行わず、サンプル画像を返す
func (h *TryOnHandler) getSampleImages(sampleCount int) ([]usecases.ImageOutput, error) {
	log.Printf("[DEBUG] getSampleImages called with sampleCount: %d", sampleCount)

	// 同じ階層の「sample_images」ディレクトリからsampleCount分の画像を取得する
	files, err := os.ReadDir("static/sample_images/person")
	if err != nil {
		log.Printf("[ERROR] Failed to read sample_images directory: %v", err)
		return nil, err
	}

	log.Printf("[DEBUG] Found %d files in sample_images directory", len(files))

	// 画像ファイルのみをフィルタリング
	var imageFiles []os.DirEntry
	for _, file := range files {
		if !file.IsDir() {
			// 拡張子で画像ファイルを判定
			name := strings.ToLower(file.Name())
			if strings.HasSuffix(name, ".jpg") || strings.HasSuffix(name, ".jpeg") ||
				strings.HasSuffix(name, ".png") || strings.HasSuffix(name, ".gif") {
				imageFiles = append(imageFiles, file)
				log.Printf("[DEBUG] Found image file: %s", file.Name())
			}
		}
	}

	if len(imageFiles) == 0 {
		log.Printf("[ERROR] No image files found in sample_images directory")
		return nil, fmt.Errorf("sample_imagesディレクトリに画像ファイルが見つかりません")
	}

	log.Printf("[DEBUG] Filtered to %d image files", len(imageFiles))

	// 実際の画像ファイル数に基づいて処理
	var images []usecases.ImageOutput
	for i := range sampleCount {
		// ファイル数が不足している場合は循環して使用
		fileIndex := i % len(imageFiles)
		file := imageFiles[fileIndex]

		filePath := filepath.Join("static/sample_images/person", file.Name())
		log.Printf("[DEBUG] Reading file %d/%d: %s", i+1, sampleCount, filePath)

		imageData, err := os.ReadFile(filePath)
		if err != nil {
			log.Printf("[ERROR] Failed to read file %s: %v", filePath, err)
			return nil, err
		}

		log.Printf("[DEBUG] Successfully read file %s, size: %d bytes", file.Name(), len(imageData))

		// ファイル拡張子からMIMEタイプを決定
		mimeType := "image/jpeg" // デフォルト
		name := strings.ToLower(file.Name())
		if strings.HasSuffix(name, ".png") {
			mimeType = "image/png"
		} else if strings.HasSuffix(name, ".gif") {
			mimeType = "image/gif"
		}

		log.Printf("[DEBUG] Determined MIME type for %s: %s", file.Name(), mimeType)

		images = append(images, usecases.ImageOutput{
			Data: imageData,
			Type: mimeType,
		})
	}

	log.Printf("[DEBUG] Returning %d images", len(images))
	return images, nil
}

func (h *TryOnHandler) HandleTryOn(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxFileSize)
	if err := r.ParseMultipartForm(maxFileSize); err != nil {
		h.sendError(w, "画像が大きすぎます（10MBまで対応）", http.StatusRequestEntityTooLarge)
		return
	}

	personFile, personFileHeader, err := r.FormFile("person_image")
	if err != nil {
		h.sendError(w, "人物画像を選んでください", http.StatusBadRequest)
		return
	}
	// mimeTypeを取得
	personMimeType := personFileHeader.Header.Get("Content-Type")
	defer personFile.Close()

	// 複数ファイルを受け取るように修正
	garmentFiles := r.MultipartForm.File["garment_image"]
	if len(garmentFiles) == 0 {
		h.sendError(w, "衣服画像を選んでください", http.StatusBadRequest)
		return
	}

	var garmentFileData []usecases.GarmentImageData
	for _, file := range garmentFiles {
		garmentFile, err := file.Open()
		if err != nil {
			h.sendError(w, "衣服画像の読み込みに失敗しました", http.StatusInternalServerError)
			return
		}

		defer garmentFile.Close()
		data, err := io.ReadAll(garmentFile)
		if err != nil {
			h.sendError(w, "衣服画像の読み込みに失敗しました", http.StatusInternalServerError)
			return
		}
		slog.Info("garmentFileData", "garmentFileData", file.Header.Get("Content-Type"), "dataSize", len(data))

		garmentFileData = append(garmentFileData, usecases.GarmentImageData{
			Data:     data,
			MimeType: file.Header.Get("Content-Type"),
		})
	}

	personFileData, err := io.ReadAll(personFile)
	if err != nil {
		h.sendError(w, "人物画像の読み込みに失敗しました", http.StatusInternalServerError)
		return
	}

	slog.Info("personFileData", "personFileData", personMimeType, "dataSize", len(personFileData))

	parameters := h.parameterService.ParseFromRequest(r)

	input := usecases.TryOnInput{
		PersonImageData:  personFileData,
		PersonMimeType:   personMimeType,
		GarmentImageData: garmentFileData,
		Parameters:       parameters,
	}

	output, err := h.tryOnUseCase.Execute(r.Context(), input)
	if err != nil {
		log.Printf("Virtual Try-On failed: %v", err)

		if h.isQuotaError(err) {
			h.sendError(w, "現在サーバーが混雑しています。しばらく待ってから再試行してください。", http.StatusTooManyRequests)
			return
		}

		hint := "ヒント: 露出や著名人・ロゴ類・過度な加工を避け、人物と衣服がはっきり写る画像で再試行してください。"
		h.sendError(w, fmt.Sprintf("生成に失敗しました: %v %s", err, hint), http.StatusInternalServerError)
		return
	}

	if output == nil {
		log.Printf("Virtual Try-On returned nil output")
		h.sendError(w, "生成に失敗しました: 結果が取得できませんでした", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store, max-age=0")

	response := h.createResponse(output.Images)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode JSON response: %v", err)
		h.sendError(w, "レスポンスの生成に失敗しました", http.StatusInternalServerError)
		return
	}
}

func (h *TryOnHandler) createResponse(imagesOutput []usecases.ImageOutput) map[string]any {
	log.Printf("[DEBUG] createResponse called with %d images", len(imagesOutput))

	var images []map[string]string
	for i, img := range imagesOutput {
		// 空のImageOutputをスキップ（防御的プログラミング）
		if len(img.Data) == 0 {
			log.Printf("[WARNING] Skipping empty image at index %d", i)
			continue
		}

		log.Printf("[DEBUG] Processing image %d: size=%d bytes, type=%s", i, len(img.Data), img.Type)

		base64Data := base64.StdEncoding.EncodeToString(img.Data)
		log.Printf("[DEBUG] Base64 encoded length: %d characters", len(base64Data))

		// Base64データの最初の100文字をログ出力（デバッグ用）
		preview := base64Data
		if len(preview) > 100 {
			preview = preview[:100] + "..."
		}
		log.Printf("[DEBUG] Base64 preview: %s", preview)

		images = append(images, map[string]string{
			"id":   fmt.Sprintf("image_%d", i),
			"data": base64Data,
			"type": img.Type,
		})
	}

	log.Printf("[DEBUG] Final response will contain %d images", len(images))

	response := map[string]any{
		"success": true,
		"images":  images,
	}

	return response
}

func (h *TryOnHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func (h *TryOnHandler) isQuotaError(err error) bool {
	return fmt.Sprintf("%v", err) != "" &&
		(fmt.Sprintf("%v", err) == "service temporarily unavailable due to high demand")
}

func (h *TryOnHandler) sendError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// SampleImage represents a sample image metadata
type SampleImage struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	URL         string `json:"url"`
	Category    string `json:"category"`
}

// HandleSampleImages サンプル画像一覧を返すAPIエンドポイント
func (h *TryOnHandler) HandleSampleImages(w http.ResponseWriter, r *http.Request) {
	// カテゴリパラメータを取得（person または garment）
	category := r.URL.Query().Get("category")
	if category == "" {
		h.sendError(w, "categoryパラメータが必要です (person または garment)", http.StatusBadRequest)
		return
	}

	if category != "person" && category != "garment" {
		h.sendError(w, "categoryは 'person' または 'garment' である必要があります", http.StatusBadRequest)
		return
	}

	var samples []SampleImage

	if category == "person" {
		samples = []SampleImage{
			{
				ID:          "person_men",
				Name:        "男性 (一般)",
				Description: "カジュアルな服装の男性",
				URL:         "/api/sample-image?category=person&id=person_men",
				Category:    "person",
			},
			{
				ID:          "person_men_50",
				Name:        "男性 (50代)",
				Description: "フォーマルな服装の中年男性",
				URL:         "/api/sample-image?category=person&id=person_men_50",
				Category:    "person",
			},
			{
				ID:          "person_women_20",
				Name:        "女性 (20代)",
				Description: "カジュアルな服装の若い女性",
				URL:         "/api/sample-image?category=person&id=person_women_20",
				Category:    "person",
			},
			{
				ID:          "person_women_70",
				Name:        "女性 (70代)",
				Description: "エレガントな服装のシニア女性",
				URL:         "/api/sample-image?category=person&id=person_women_70",
				Category:    "person",
			},
		}
	} else {
		samples = []SampleImage{
			{
				ID:          "garment_tops",
				Name:        "トップス (ベーシック)",
				Description: "シンプルなデザインのトップス",
				URL:         "/api/sample-image?category=garment&id=garment_tops",
				Category:    "garment",
			},
			{
				ID:          "garment_tops_hade",
				Name:        "トップス (派手)",
				Description: "カラフルで目立つデザインのトップス",
				URL:         "/api/sample-image?category=garment&id=garment_tops_hade",
				Category:    "garment",
			},
			{
				ID:          "garment_pants",
				Name:        "パンツ",
				Description: "カジュアルなパンツ",
				URL:         "/api/sample-image?category=garment&id=garment_pants",
				Category:    "garment",
			},
			{
				ID:          "garment_shoes",
				Name:        "シューズ",
				Description: "スタイリッシュなシューズ",
				URL:         "/api/sample-image?category=garment&id=garment_shoes",
				Category:    "garment",
			},
			{
				ID:          "garment_shoes_double",
				Name:        "シューズ（両足）",
				Description: "スタイリッシュなシューズ（両足）",
				URL:         "/api/sample-image?category=garment&id=garment_shoes_double",
				Category:    "garment",
			},
			{
				ID:          "garment_neckless",
				Name:        "ネックレス",
				Description: "エレガントなネックレス",
				URL:         "/api/sample-image?category=garment&id=garment_neckless",
				Category:    "garment",
			},
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600") // 1時間キャッシュ

	response := map[string]interface{}{
		"success": true,
		"samples": samples,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode sample images response: %v", err)
		h.sendError(w, "レスポンスの生成に失敗しました", http.StatusInternalServerError)
		return
	}
}

func (h *TryOnHandler) HandleSampleImage(w http.ResponseWriter, r *http.Request) {
	// URLパラメータからカテゴリとIDを取得
	category := r.URL.Query().Get("category")
	id := r.URL.Query().Get("id")

	if category == "" || id == "" {
		h.sendError(w, "categoryとidパラメータが必要です", http.StatusBadRequest)
		return
	}

	if category != "person" && category != "garment" {
		h.sendError(w, "categoryは 'person' または 'garment' である必要があります", http.StatusBadRequest)
		return
	}

	// サンプル画像の定義からURLを取得
	var imageURL string

	if category == "person" {
		switch id {
		case "person_men":
			imageURL = "https://storage.googleapis.com/try-on-generated-central/sample/person/sample_men.png"
		case "person_men_50":
			imageURL = "https://storage.googleapis.com/try-on-generated-central/sample/person/sample_men_50.png"
		case "person_women_20":
			imageURL = "https://storage.googleapis.com/try-on-generated-central/sample/person/sample_women_20.png"
		case "person_women_70":
			imageURL = "https://storage.googleapis.com/try-on-generated-central/sample/person/sample_women_70.png"
		default:
			h.sendError(w, "無効なperson ID", http.StatusBadRequest)
			return
		}
	} else {
		switch id {
		case "garment_tops":
			imageURL = "https://storage.googleapis.com/try-on-generated-central/sample/garment/sample_tops.png"
		case "garment_tops_hade":
			imageURL = "https://storage.googleapis.com/try-on-generated-central/sample/garment/sample_tops_hade.png"
		case "garment_pants":
			imageURL = "https://storage.googleapis.com/try-on-generated-central/sample/garment/sample_pants.png"
		case "garment_shoes":
			imageURL = "https://storage.googleapis.com/try-on-generated-central/sample/garment/sample_shoes.png"
		case "garment_shoes_double":
			imageURL = "https://storage.googleapis.com/try-on-generated-central/sample/garment/sample_shoes_double.png"
		case "garment_neckless":
			imageURL = "https://storage.googleapis.com/try-on-generated-central/sample/garment/sample_neckless.png"
		default:
			h.sendError(w, "無効なgarment ID", http.StatusBadRequest)
			return
		}
	}

	// Google Cloud Storageから画像を取得してプロキシ
	resp, err := http.Get(imageURL)
	if err != nil {
		log.Printf("Failed to fetch sample image from %s: %v", imageURL, err)
		h.sendError(w, "サンプル画像の取得に失敗しました", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Sample image fetch failed with status %d from %s", resp.StatusCode, imageURL)
		h.sendError(w, "サンプル画像が見つかりません", http.StatusNotFound)
		return
	}

	// Content-Typeとキャッシュヘッダーを設定
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.Header().Set("Cache-Control", "public, max-age=3600") // 1時間キャッシュ
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET")

	// 画像データをストリーム転送
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		log.Printf("Failed to copy sample image data: %v", err)
		return
	}
}

func (h *TryOnHandler) HandleIndex(w http.ResponseWriter, r *http.Request) {
	renderStaticPage(w, tryOnIndexHTML)
}

// HandleImagen - imagen画像生成API
func (h *ImagenHandler) HandleImagen(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.sendError(w, "POST method required", http.StatusMethodNotAllowed)
		return
	}

	// パラメータの取得
	prompt := r.FormValue("prompt")
	if prompt == "" {
		h.sendError(w, "promptパラメータが必要です", http.StatusBadRequest)
		return
	}

	imagenModel := r.FormValue("imagenModel")
	if imagenModel == "" {
		imagenModel = h.getDefaultImagenModel()
	}

	// モデルIDのバリデーション
	if !h.isValidImagenModel(imagenModel) {
		log.Printf("[WARNING] Invalid modelo ID requested: %s", imagenModel)
		h.sendError(w, fmt.Sprintf("サポートされていないモデルです: %s", imagenModel), http.StatusBadRequest)
		return
	}

	// 詳細設定パラメータの取得と解析
	numberOfImages := 1
	if numStr := r.FormValue("numberOfImages"); numStr != "" {
		if num, err := strconv.Atoi(numStr); err == nil && num >= 1 && num <= 4 {
			numberOfImages = num
		}
	}

	aspectRatio := r.FormValue("aspectRatio")
	if aspectRatio == "" {
		aspectRatio = "1:1"
	}

	negativePrompt := r.FormValue("negativePrompt")

	seed := int64(0)
	if seedStr := r.FormValue("seed"); seedStr != "" {
		if seedVal, err := strconv.ParseInt(seedStr, 10, 64); err == nil {
			seed = seedVal
		}
	}

	includeRaiReason := r.FormValue("includeRaiReason") == "true"

	log.Printf("[INFO] Imagen generation request - prompt: %s, model: %s, numberOfImages: %d, aspectRatio: %s",
		prompt, imagenModel, numberOfImages, aspectRatio)

	input := usecases.ImagenInput{
		Prompt:           prompt,
		ImagenModel:      imagenModel,
		NumberOfImages:   numberOfImages,
		AspectRatio:      aspectRatio,
		NegativePrompt:   negativePrompt,
		Seed:             seed,
		IncludeRaiReason: includeRaiReason,
	}

	output, err := h.imagenUseCase.Execute(r.Context(), input)
	if err != nil {
		log.Printf("Imagen generation failed: %v", err)

		if h.isQuotaError(err) {
			h.sendError(w, "現在サーバーが混雑しています。しばらく待ってから再試行してください。", http.StatusTooManyRequests)
			return
		}

		h.sendError(w, fmt.Sprintf("画像生成に失敗しました: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store, max-age=0")

	response := h.createImagenResponse(output.Images)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode JSON response: %v", err)
		h.sendError(w, "レスポンスの生成に失敗しました", http.StatusInternalServerError)
		return
	}
}

// createImagenResponse - Imagen用のレスポンスを生成
func (h *ImagenHandler) createImagenResponse(imagesOutput []usecases.ImageOutput) map[string]any {
	log.Printf("[DEBUG] createImagenResponse called with %d images", len(imagesOutput))

	var images []map[string]string
	for i, img := range imagesOutput {
		// 空のImageOutputをスキップ（防御的プログラミング）
		if len(img.Data) == 0 {
			log.Printf("[WARNING] Skipping empty image at index %d", i)
			continue
		}

		log.Printf("[DEBUG] Processing image %d: size=%d bytes, type=%s", i, len(img.Data), img.Type)

		base64Data := base64.StdEncoding.EncodeToString(img.Data)
		log.Printf("[DEBUG] Base64 encoded length: %d characters", len(base64Data))

		images = append(images, map[string]string{
			"id":   fmt.Sprintf("imagen_%d", i),
			"data": base64Data,
			"type": img.Type,
		})
	}

	log.Printf("[DEBUG] Final response will contain %d images", len(images))

	response := map[string]any{
		"success": true,
		"images":  images,
	}

	return response
}

// isQuotaError - クォータエラーかどうかを判定
func (h *ImagenHandler) isQuotaError(err error) bool {
	return fmt.Sprintf("%v", err) != "" &&
		(fmt.Sprintf("%v", err) == "service temporarily unavailable due to high demand")
}

// sendError - エラーレスポンスを送信
func (h *ImagenHandler) sendError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// HandleImagenIndex - Imagen画像生成画面を表示
func (h *ImagenHandler) HandleImagenIndex(w http.ResponseWriter, r *http.Request) {
	// 現在のVertex AIリージョン情報をツールチップに含める
	locationInfo := fmt.Sprintf(" 現在のVertex AIリージョン: %s", h.location)

	// モデル選択肢を動的に生成
	var modelOptions strings.Builder
	for i, model := range supportedImagenModels {
		selected := ""
		if model.ID == h.getDefaultImagenModel() {
			selected = " selected"
		}

		modelOptions.WriteString(fmt.Sprintf(
			`<option value="%s"%s>%s</option>`,
			model.ID,
			selected,
			fmt.Sprintf("%s - %s", model.Name, model.Description),
		))
		if i < len(supportedImagenModels)-1 {
			modelOptions.WriteString("\n")
		}
	}
	renderPage(w, "imagen_index.html", pageData{LocationInfo: locationInfo, ModelOptions: modelOptions.String()})
}
