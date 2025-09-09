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

	log.Printf("[DEBUG] Response: %v", response)

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

	html := `<!DOCTYPE html>
<html lang="ja">
<head>
<meta charset="UTF-8"/>
<meta name="viewport" content="width=device-width, initial-scale=1.0"/>
<title>Vertex AI Virtual Try-On</title>
<script src="https://cdn.tailwindcss.com"></script>
<style>
body { font-family: Inter, system-ui, -apple-system, Segoe UI, Roboto, sans-serif; }
.preview-box{width:100%;height:320px;background:#f3f4f6;border:2px dashed #d1d5db;display:flex;align-items:center;justify-content:center;overflow:hidden}
.preview-box img{max-width:100%;max-height:100%;object-fit:contain}
.loader{border:8px solid #f3f3f3;border-top:8px solid #6366f1;border-radius:50%;width:56px;height:56px;animation:spin 1.2s linear infinite}
@keyframes spin{0%{transform:rotate(0)}100%{transform:rotate(360deg)}}
.tooltip { position: relative; display: inline-block; }
.tooltip .tooltiptext { visibility: hidden; width: 280px; background-color: #333; color: #fff; text-center; border-radius: 6px; padding: 8px; position: absolute; z-index: 1; bottom: 125%; left: 50%; margin-left: -140px; opacity: 0; transition: opacity 0.3s; font-size: 12px; line-height: 1.4; }
.tooltip .tooltiptext::after { content: ""; position: absolute; top: 100%; left: 50%; margin-left: -5px; border-width: 5px; border-style: solid; border-color: #333 transparent transparent transparent; }
.tooltip:hover .tooltiptext { visibility: visible; opacity: 1; }
.info-icon { display: inline-flex; align-items: center; justify-content: center; width: 16px; height: 16px; border-radius: 50%; background-color: #6366f1; color: white; font-size: 12px; font-weight: bold; margin-left: 4px; cursor: help; }
.image-upload-area {
    border: 2px dashed #d1d5db;
    background: #f9fafb;
    border-radius: 8px;
    transition: all 0.3s ease;
    cursor: pointer;
    min-height: 200px;
}
.image-upload-area:hover {
    border-color: #6366f1;
    background: #f3f4f6;
}
.image-upload-area.drag-over {
    border-color: #6366f1;
    background: #eef2ff;
}
</style>
</head>
<body class="bg-gray-50 text-gray-800">
<div class="container mx-auto p-4 md:p-8 max-w-5xl">
<!-- ナビゲーションバー -->
<nav class="bg-white shadow-sm rounded-lg mb-6 p-4">
<div class="flex flex-wrap justify-center gap-3">
<button onclick="location.href='/'" class="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors font-medium shadow-sm">
<svg class="w-4 h-4 inline mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"/></svg>
Virtual Try-On
</button>
<button onclick="location.href='/imagen'" class="px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 transition-colors font-medium shadow-sm">
<svg class="w-4 h-4 inline mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"/></svg>
Imagen画像生成
</button>
<button onclick="location.href='/veo'" class="px-4 py-2 bg-purple-600 text-white rounded-lg hover:bg-purple-700 transition-colors font-medium shadow-sm">
<svg class="w-4 h-4 inline mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z"/></svg>
Veo動画生成
</button>
<button onclick="location.href='/nanobanana/image-editing'" class="px-4 py-2 bg-orange-600 text-white rounded-lg hover:bg-orange-700 transition-colors font-medium shadow-sm">
<svg class="w-4 h-4 inline mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/></svg>
Nanobanana画像編集
</button>
</div>
</nav>

<header class="text-center mb-8">
<h1 class="text-3xl md:text-4xl font-bold text-gray-900">Vertex AI Virtual Try-On</h1>
<p class="text-gray-600 mt-2">人物と衣服の画像をアップロードして、着せ替えを試そう。</p>
</header>
<main class="bg-white p-6 md:p-8 rounded-2xl shadow-lg">
<form id="tryon-form">
<div class="grid grid-cols-1 md:grid-cols-2 gap-6 mb-6">
<div>
<label class="block text-lg font-semibold mb-2 text-gray-700">1. 人物画像をアップロード</label>
<div class="image-upload-area p-8 text-center" id="person-upload-area">
<input type="file" id="person-image" name="person_image" accept="image/*" class="hidden" required>
<div id="person-upload-content">
<svg class="mx-auto h-12 w-12 text-gray-400 mb-4" stroke="currentColor" fill="none" viewBox="0 0 48 48">
<path d="M28 8H12a4 4 0 00-4 4v20m32-12v8m0 0v8a4 4 0 01-4 4H12a4 4 0 01-4-4v-4m32-4l-3.172-3.172a4 4 0 00-5.656 0L28 28M8 32l9.172-9.172a4 4 0 015.656 0L28 28m0 0l4 4m4-24h8m-4-4v8m-12 4h.02" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
</svg>
<p class="text-lg text-gray-600 mb-2">人物画像をドラッグ&ドロップするか、クリックして選択</p>
<p class="text-sm text-gray-500 mb-4">JPG, PNG形式をサポート</p>
<button type="button" id="person-sample-btn" class="inline-flex items-center px-4 py-2 rounded-lg bg-gradient-to-r from-green-500 to-teal-500 text-white shadow hover:shadow-lg">
<svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"/></svg>
<span>サンプルから選択</span>
</button>
</div>
<div id="person-preview" class="hidden">
<img id="person-preview-image" class="max-w-full max-h-64 mx-auto rounded-lg">
<p class="text-sm text-gray-600 mt-2">別の画像に変更するにはクリック</p>
</div>
</div>
<span id="person-name" class="text-sm text-gray-500 mt-2"></span>
</div>
<div>
<label class="block text-lg font-semibold mb-2 text-gray-700">2. 衣服画像をアップロード（最大5枚まで）</label>
<div class="image-upload-area p-8 text-center" id="garment-upload-area">
<input type="file" id="garment-image" name="garment_image" accept="image/*" multiple class="hidden" required>
<div id="garment-upload-content">
<svg class="mx-auto h-12 w-12 text-gray-400 mb-4" stroke="currentColor" fill="none" viewBox="0 0 48 48">
<path d="M28 8H12a4 4 0 00-4 4v20m32-12v8m0 0v8a4 4 0 01-4 4H12a4 4 0 01-4-4v-4m32-4l-3.172-3.172a4 4 0 00-5.656 0L28 28M8 32l9.172-9.172a4 4 0 015.656 0L28 28m0 0l4 4m4-24h8m-4-4v8m-12 4h.02" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
</svg>
<p class="text-lg text-gray-600 mb-2">衣服画像をドラッグ&ドロップするか、クリックして選択</p>
<p class="text-sm text-gray-500 mb-4">JPG, PNG形式をサポート（最大5枚まで）</p>
<button type="button" id="garment-sample-btn" class="inline-flex items-center px-4 py-2 rounded-lg bg-gradient-to-r from-green-500 to-teal-500 text-white shadow hover:shadow-lg">
<svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"/></svg>
<span>サンプルから選択</span>
</button>
</div>
<div id="garment-preview" class="hidden">
<div id="garment-preview-grid" class="grid grid-cols-2 md:grid-cols-3 gap-4">
<!-- 複数画像のプレビューがここに表示される -->
</div>
<p class="text-sm text-gray-600 mt-2">別の画像に変更するにはクリック</p>
</div>
</div>
<span id="garment-name" class="text-sm text-gray-500 mt-2"></span>
</div>
</div>
<!-- 詳細設定セクション -->
<div id="advanced-settings" class="mb-6 p-4 bg-gray-50 rounded-lg" style="display: none;">
<h3 class="text-lg font-semibold mb-4 text-gray-700">詳細設定</h3>
<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
<div>
<label class="block text-sm font-medium mb-1 text-gray-600">
Watermark追加
<div class="tooltip">
<span class="info-icon">?</span>
<span class="tooltiptext">生成画像にウォーターマークを追加するかどうかを設定します。有効にすると画像の品質保護に役立ちますが、Seedによる結果の再現性は無効になります。</span>
</div>
</label>
<select name="add_watermark" id="watermark-select" class="w-full px-3 py-2 border border-gray-300 rounded-md">
<option value="true">有効</option>
<option value="false">無効</option>
</select>
</div>
<div>
<label class="block text-sm font-medium mb-1 text-gray-600">
Base Steps
<div class="tooltip">
<span class="info-icon">?</span>
<span class="tooltiptext">AI生成プロセスのステップ数です。値が大きいほど詳細で高品質な結果が得られますが、生成時間も長くなります。推奨値: 32</span>
</div>
</label>
<input type="number" name="base_steps" min="1" max="100" value="32" class="w-full px-3 py-2 border border-gray-300 rounded-md">
</div>
<div>
<label class="block text-sm font-medium mb-1 text-gray-600">
Person Generation
<div class="tooltip">
<span class="info-icon">?</span>
<span class="tooltiptext">人物の生成に関する制限設定です。「成人のみ許可」は成人の人物のみ生成、「全年齢許可」はすべての年齢層、「人物生成禁止」は人物の生成を完全に無効化します。</span>
</div>
</label>
<select name="person_generation" class="w-full px-3 py-2 border border-gray-300 rounded-md">
<option value="allow_adult">成人のみ許可</option>
<option value="allow_all">全年齢許可</option>
<option value="dont_allow">人物生成禁止</option>
</select>
</div>
<div>
<label class="block text-sm font-medium mb-1 text-gray-600">
Safety Setting
<div class="tooltip">
<span class="info-icon">?</span>
<span class="tooltiptext">コンテンツの安全性フィルターレベルです。「中程度以上をブロック」が推奨設定で、不適切なコンテンツを効果的にブロックします。より厳格または緩和された設定も選択可能です。</span>
</div>
</label>
<select name="safety_setting" class="w-full px-3 py-2 border border-gray-300 rounded-md">
<option value="block_medium_and_above">中程度以上をブロック</option>
<option value="block_low_and_above">低レベル以上をブロック</option>
<option value="block_only_high">高レベルのみブロック</option>
<option value="block_none">ブロックなし</option>
</select>
</div>
<div>
<label class="block text-sm font-medium mb-1 text-gray-600">
Sample Count
<div class="tooltip">
<span class="info-icon">?</span>
<span class="tooltiptext">一度に生成する画像の枚数です（1-4枚）。複数生成すると異なるバリエーションが得られますが、生成時間とコストが増加します。</span>
</div>
</label>
<input type="number" name="sample_count" min="1" max="4" value="1" class="w-full px-3 py-2 border border-gray-300 rounded-md">
</div>
<div>
<label class="block text-sm font-medium mb-1 text-gray-600">
Seed (オプション)
<div class="tooltip">
<span class="info-icon">?</span>
<span class="tooltiptext">生成結果の再現性を制御する数値です。同じSeed値を使用すると同じ結果が得られます。ランダムな結果を得たい場合は0にしてください。※Watermark有効時は使用できません。</span>
</div>
</label>
<input type="number" name="seed" value="0" class="w-full px-3 py-2 border border-gray-300 rounded-md" id="seed-input">
<small class="text-xs text-orange-600 mt-1 hidden" id="seed-warning">※ Watermark有効時はSeedは無効になります</small>
</div>
<div>
<label class="block text-sm font-medium mb-1 text-gray-600">
Output MIME Type
<div class="tooltip">
<span class="info-icon">?</span>
<span class="tooltiptext">出力画像の形式です。PNG：透明度保持、高品質、ファイルサイズ大。JPEG：ファイルサイズ小、圧縮による若干の品質劣化あり、圧縮品質調整可能。</span>
</div>
</label>
<select name="output_mime_type" id="mime-type-select" class="w-full px-3 py-2 border border-gray-300 rounded-md">
<option value="image/png">PNG</option>
<option value="image/jpeg">JPEG</option>
</select>
</div>
<div>
<label class="block text-sm font-medium mb-1 text-gray-600">
Compression Quality (JPEG用)
<div class="tooltip">
<span class="info-icon">?</span>
<span class="tooltiptext">JPEG画像の圧縮品質です（0-100）。値が高いほど高品質ですがファイルサイズが大きくなります。推奨値：75。※PNG選択時は無効です。</span>
</div>
</label>
<input type="number" name="compression_quality" min="0" max="100" value="75" class="w-full px-3 py-2 border border-gray-300 rounded-md" id="compression-quality-input">
<small class="text-xs text-orange-600 mt-1 hidden" id="compression-warning">※ PNG選択時は圧縮品質は無効になります</small>
</div>

</div>
</div>
<div class="text-center mb-6">
<button type="button" id="toggle-advanced" class="text-sm text-indigo-600 hover:text-indigo-800 mb-4">
詳細設定を表示
</button>
</div>

<!-- 実行ボタン（メイン） -->
<div class="text-center mb-8">
<button type="submit" id="submit-btn"
class="bg-gradient-to-r from-indigo-500 to-blue-600 text-white font-bold py-4 px-12 rounded-full hover:shadow-xl transform hover:-translate-y-0.5 transition-all text-lg">
着せ替えを実行
</button>
</div>

<!-- クリアボタン（サブ） -->
<div class="text-center">
<button type="button" id="clear-btn"
class="px-6 py-2 text-sm rounded-lg border border-gray-300 text-gray-600 hover:bg-gray-50 hover:border-gray-400 transition-all">
<svg class="w-4 h-4 inline mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/></svg>
全てクリア
</button>
</div>
</form>
<div id="result-section" class="mt-10 hidden">
<h2 class="text-2xl font-bold text-center mb-4 text-gray-800">生成結果</h2>
<div id="result-display" class="preview-box rounded-lg bg-green-50"></div>
<div id="multiple-results" class="grid grid-cols-1 md:grid-cols-2 gap-4 mt-4" style="display: none;"></div>
</div>
<div id="error-message" class="mt-6 hidden bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded-lg"></div>
</main>

<!-- サンプル画像選択モーダル -->
<div id="sample-modal" class="fixed inset-0 bg-black bg-opacity-50 hidden z-50">
<div class="flex items-center justify-center min-h-screen p-4">
<div class="bg-white rounded-lg max-w-5xl w-full max-h-[95vh] overflow-hidden flex flex-col">
<div class="p-6 border-b border-gray-200">
<div class="flex justify-between items-center">
<h2 id="modal-title" class="text-2xl font-bold text-gray-800">サンプル画像を選択</h2>
<button id="close-modal" class="text-gray-500 hover:text-gray-700 text-2xl font-bold">&times;</button>
</div>
<div id="selection-info" class="mt-2 text-sm text-gray-600">
<span id="selected-count">0</span>枚選択中（最大5枚まで）
</div>
</div>
<div class="flex-1 overflow-y-auto p-6">
<div id="sample-grid" class="grid grid-cols-2 md:grid-cols-3 gap-6">
<!-- サンプル画像がここに動的に追加される -->
</div>
</div>
<div class="p-6 border-t border-gray-200 text-center">
<button id="cancel-sample" class="px-6 py-2 bg-gray-300 text-gray-700 rounded-lg hover:bg-gray-400 mr-4">キャンセル</button>
<button id="confirm-sample" class="px-6 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 disabled:bg-gray-400 disabled:cursor-not-allowed" disabled>選択完了</button>
</div>
</div>
</div>
</div>


</div>
<script>
const form = document.getElementById('tryon-form');
const personInput = document.getElementById('person-image');
const garmentInput = document.getElementById('garment-image');
const personUploadArea = document.getElementById('person-upload-area');
const garmentUploadArea = document.getElementById('garment-upload-area');
const personUploadContent = document.getElementById('person-upload-content');
const garmentUploadContent = document.getElementById('garment-upload-content');
const personPreview = document.getElementById('person-preview');
const garmentPreview = document.getElementById('garment-preview');
const personPreviewImage = document.getElementById('person-preview-image');
const garmentPreviewGrid = document.getElementById('garment-preview-grid');
const personName = document.getElementById('person-name');
const garmentName = document.getElementById('garment-name');
const resultSection = document.getElementById('result-section');
const resultDisplay = document.getElementById('result-display');
const multipleResults = document.getElementById('multiple-results');
const errorMessage = document.getElementById('error-message');
const submitBtn = document.getElementById('submit-btn');
const clearBtn = document.getElementById('clear-btn');
const toggleAdvancedBtn = document.getElementById('toggle-advanced');
const advancedSettings = document.getElementById('advanced-settings');
const watermarkSelect = document.getElementById('watermark-select');
const seedInput = document.getElementById('seed-input');
const seedWarning = document.getElementById('seed-warning');
const mimeTypeSelect = document.getElementById('mime-type-select');
const compressionQualityInput = document.getElementById('compression-quality-input');
const compressionWarning = document.getElementById('compression-warning');

// サンプル画像選択関連の要素
const personSampleBtn = document.getElementById('person-sample-btn');
const garmentSampleBtn = document.getElementById('garment-sample-btn');
const sampleModal = document.getElementById('sample-modal');
const modalTitle = document.getElementById('modal-title');
const sampleGrid = document.getElementById('sample-grid');
const closeModal = document.getElementById('close-modal');
const cancelSample = document.getElementById('cancel-sample');
const confirmSample = document.getElementById('confirm-sample');
const selectedCount = document.getElementById('selected-count');

// 現在選択中のサンプル画像を追跡するための変数
let currentPersonSample = null;
let currentGarmentSample = null;
let currentModalCategory = null;
let selectedSamples = []; // モーダル内で選択された画像を追跡

// アップロードされたファイルを追跡するための変数
let uploadedGarmentFiles = []; // アップロードされた服装画像ファイルを追跡

// ファイルアップロード関連の処理
function setupDragAndDrop(uploadArea, input, uploadContent, previewElement, previewImage, nameLabel) {
    uploadArea.addEventListener('click', () => {
        input.click();
    });

    uploadArea.addEventListener('dragover', (e) => {
        e.preventDefault();
        uploadArea.classList.add('drag-over');
    });

    uploadArea.addEventListener('dragleave', (e) => {
        e.preventDefault();
        uploadArea.classList.remove('drag-over');
    });

    uploadArea.addEventListener('drop', (e) => {
        e.preventDefault();
        uploadArea.classList.remove('drag-over');
        
        const files = e.dataTransfer.files;
        if (files.length > 0) {
            if (input.id === 'garment-image') {
                handleGarmentFileSelect(Array.from(files), input, uploadContent, previewElement, previewImage, nameLabel);
            } else {
                handleFileSelect(files[0], input, uploadContent, previewElement, previewImage, nameLabel);
            }
        }
    });

    input.addEventListener('change', (e) => {
        if (e.target.files.length > 0) {
            if (input.id === 'garment-image') {
                handleGarmentFileSelect(Array.from(e.target.files), input, uploadContent, previewElement, previewImage, nameLabel);
            } else {
                handleFileSelect(e.target.files[0], input, uploadContent, previewElement, previewImage, nameLabel);
            }
        }
    });
}

function handleFileSelect(file, input, uploadContent, previewElement, previewImage, nameLabel) {
    if (!file.type.startsWith('image/')) {
        errorMessage.textContent = '画像ファイルを選択してください';
        errorMessage.classList.remove('hidden');
        return;
    }
    
    errorMessage.classList.add('hidden');
    
    const reader = new FileReader();
    reader.onload = (e) => {
        previewImage.src = e.target.result;
        uploadContent.classList.add('hidden');
        previewElement.classList.remove('hidden');
        nameLabel.textContent = file.name;
    };
    reader.readAsDataURL(file);
}

function handleGarmentFileSelect(files, input, uploadContent, previewElement, previewGrid, nameLabel) {
    // 既存のファイル数と新しいファイル数の合計が5枚を超えるかチェック
    if (uploadedGarmentFiles.length + files.length > 5) {
        errorMessage.textContent = '衣服画像は最大5枚まで選択できます（現在' + uploadedGarmentFiles.length + '枚選択中）';
        errorMessage.classList.remove('hidden');
        return;
    }
    
    // 画像ファイルかチェック
    for (let file of files) {
        if (!file.type.startsWith('image/')) {
            errorMessage.textContent = '画像ファイルを選択してください';
            errorMessage.classList.remove('hidden');
            return;
        }
    }
    
    errorMessage.classList.add('hidden');
    
    // 新しいファイルを既存のリストに追加
    uploadedGarmentFiles.push(...files);
    
    // プレビューグリッドを更新
    updateGarmentFilePreview();
    
    uploadContent.classList.add('hidden');
    previewElement.classList.remove('hidden');
    nameLabel.textContent = uploadedGarmentFiles.length + '枚の画像を選択中';
}

function updateGarmentFilePreview() {
    garmentPreviewGrid.innerHTML = '';
    
    uploadedGarmentFiles.forEach((file, index) => {
        const reader = new FileReader();
        reader.onload = (e) => {
            const imgContainer = document.createElement('div');
            imgContainer.className = 'relative';
            
            const img = document.createElement('img');
            img.src = e.target.result;
            img.alt = file.name;
            img.className = 'w-full h-24 object-cover rounded-lg border';
            
            const removeBtn = document.createElement('button');
            removeBtn.textContent = '×';
            removeBtn.className = 'absolute -top-2 -right-2 bg-red-500 text-white rounded-full w-6 h-6 flex items-center justify-center text-xs hover:bg-red-600';
            removeBtn.onclick = (event) => {
                event.preventDefault();
                event.stopPropagation();
                // ファイルを配列から削除
                uploadedGarmentFiles.splice(index, 1);
                updateGarmentFilePreview();
                updateGarmentFileList();
            };
            
            imgContainer.appendChild(img);
            imgContainer.appendChild(removeBtn);
            garmentPreviewGrid.appendChild(imgContainer);
        };
        reader.readAsDataURL(file);
    });
}

function updateGarmentFileList() {
    if (uploadedGarmentFiles.length === 0) {
        garmentUploadContent.classList.remove('hidden');
        garmentPreview.classList.add('hidden');
        garmentName.textContent = '';
    } else {
        garmentName.textContent = uploadedGarmentFiles.length + '枚の画像を選択中';
    }
}

// サンプル画像関連の関数
async function loadSampleImages(category) {
    try {
        const response = await fetch('/api/sample-images?category=' + category);
        if (!response.ok) {
            throw new Error('Failed to load sample images');
        }
        const data = await response.json();
        return data.samples || [];
    } catch (error) {
        console.error('Error loading sample images:', error);
        return [];
    }
}

function showSampleModal(category) {
    currentModalCategory = category;
    modalTitle.textContent = category === 'person' ? '人物画像を選択' : '衣服画像を選択';
    selectedSamples = []; // 選択をリセット
    updateSelectionInfo();
    sampleModal.classList.remove('hidden');
    
    // サンプル画像を読み込んで表示
    loadSampleImages(category).then(samples => {
        sampleGrid.innerHTML = '';
        samples.forEach(sample => {
            const sampleItem = document.createElement('div');
            sampleItem.className = 'border rounded-lg p-4 cursor-pointer hover:border-blue-500 hover:shadow-md transition-all relative';
            sampleItem.innerHTML = 
                '<div class="absolute top-2 left-2 z-10">' +
                '<input type="checkbox" class="sample-checkbox w-4 h-4 text-indigo-600 bg-gray-100 border-gray-300 rounded focus:ring-indigo-500" data-sample-id="' + sample.id + '">' +
                '</div>' +
                '<div class="w-full h-48 bg-gray-100 rounded mb-3 flex items-center justify-center overflow-hidden">' +
                '<img src="' + sample.url + '" alt="' + sample.name + '" class="max-w-full max-h-full object-contain">' +
                '</div>' +
                '<h3 class="font-semibold text-sm text-gray-800 mb-1">' + sample.name + '</h3>' +
                '<p class="text-xs text-gray-600 leading-relaxed">' + sample.description + '</p>';
            
            const checkbox = sampleItem.querySelector('.sample-checkbox');
            checkbox.addEventListener('change', (e) => {
                e.stopPropagation();
                toggleSampleSelection(sample, checkbox.checked);
            });
            
            sampleItem.addEventListener('click', (e) => {
                if (e.target !== checkbox) {
                    checkbox.checked = !checkbox.checked;
                    toggleSampleSelection(sample, checkbox.checked);
                }
            });
            
            sampleGrid.appendChild(sampleItem);
        });
    });
}

function toggleSampleSelection(sample, isSelected) {
    if (isSelected) {
        // 最大選択数の制限チェック
        const maxCount = currentModalCategory === 'person' ? 1 : 5;
        const categoryName = currentModalCategory === 'person' ? '人物画像' : '衣服画像';
        
        if (selectedSamples.length >= maxCount) {
            errorMessage.textContent = categoryName + 'は最大' + maxCount + '枚まで選択できます';
            errorMessage.classList.remove('hidden');
            // チェックボックスを元に戻す
            const checkbox = document.querySelector('[data-sample-id="' + sample.id + '"]');
            if (checkbox) checkbox.checked = false;
            return;
        }
        selectedSamples.push(sample);
    } else {
        selectedSamples = selectedSamples.filter(s => s.id !== sample.id);
    }
    updateSelectionInfo();
}

function updateSelectionInfo() {
    selectedCount.textContent = selectedSamples.length;
    confirmSample.disabled = selectedSamples.length === 0;
    
    // 選択情報の表示を更新
    const maxCount = currentModalCategory === 'person' ? 1 : 5;
    const categoryName = currentModalCategory === 'person' ? '人物画像' : '衣服画像';
    const infoText = selectedSamples.length + '枚選択中（最大' + maxCount + '枚まで）';
    selectedCount.parentNode.innerHTML = '<span id="selected-count">' + selectedSamples.length + '</span>枚選択中（最大' + maxCount + '枚まで）';
}

function confirmSampleSelection() {
    if (currentModalCategory === 'person') {
        // 人物画像の場合は1つだけ選択
        if (selectedSamples.length > 0) {
            const sample = selectedSamples[0];
            currentPersonSample = sample;
            personPreviewImage.src = sample.url;
            personPreviewImage.alt = sample.name;
            personUploadContent.classList.add('hidden');
            personPreview.classList.remove('hidden');
            personName.textContent = sample.name + ' (サンプル)';
            personInput.value = '';
            personInput.removeAttribute('required');
        }
    } else {
        // 服装画像の場合は複数選択
        currentGarmentSample = [...selectedSamples];
        
        // プレビューグリッドを更新
        garmentPreviewGrid.innerHTML = '';
        currentGarmentSample.forEach((s, index) => {
            const imgContainer = document.createElement('div');
            imgContainer.className = 'relative';
            
            const img = document.createElement('img');
            img.src = s.url;
            img.alt = s.name;
            img.className = 'w-full h-24 object-cover rounded-lg border';
            
            const removeBtn = document.createElement('button');
            removeBtn.textContent = '×';
            removeBtn.className = 'absolute -top-2 -right-2 bg-red-500 text-white rounded-full w-6 h-6 flex items-center justify-center text-xs hover:bg-red-600';
            removeBtn.onclick = (event) => {
                event.preventDefault();
                event.stopPropagation();
                currentGarmentSample.splice(index, 1);
                updateGarmentSampleList();
            };
            
            imgContainer.appendChild(img);
            imgContainer.appendChild(removeBtn);
            garmentPreviewGrid.appendChild(imgContainer);
        });
        
        garmentUploadContent.classList.add('hidden');
        garmentPreview.classList.remove('hidden');
        garmentName.textContent = currentGarmentSample.length + '枚のサンプル画像を選択中';
        garmentInput.value = '';
        garmentInput.removeAttribute('required');
        
        // アップロードされたファイルをクリア
        uploadedGarmentFiles = [];
    }
    
    // モーダルを閉じる
    sampleModal.classList.add('hidden');
}

function selectSampleImage(sample, category) {
    // 旧来の単一選択処理（後方互換性のため残す）
    if (category === 'person') {
        currentPersonSample = sample;
        personPreviewImage.src = sample.url;
        personPreviewImage.alt = sample.name;
        personUploadContent.classList.add('hidden');
        personPreview.classList.remove('hidden');
        personName.textContent = sample.name + ' (サンプル)';
        personInput.value = '';
        personInput.removeAttribute('required');
        sampleModal.classList.add('hidden');
    }
}

function updateGarmentSampleList() {
    if (!currentGarmentSample || currentGarmentSample.length === 0) {
        garmentUploadContent.classList.remove('hidden');
        garmentPreview.classList.add('hidden');
        garmentName.textContent = '';
        currentGarmentSample = null;
    } else {
        garmentName.textContent = currentGarmentSample.length + '枚のサンプル画像を選択中';
    }
}

function closeSampleModal() {
    sampleModal.classList.add('hidden');
    currentModalCategory = null;
}

// サンプル画像ボタンのイベントリスナー
personSampleBtn.addEventListener('click', (e) => {
    e.stopPropagation(); // 親要素のクリックイベントを防ぐ
    showSampleModal('person');
});

garmentSampleBtn.addEventListener('click', (e) => {
    e.stopPropagation(); // 親要素のクリックイベントを防ぐ  
    showSampleModal('garment');
});

// モーダル関連のイベントリスナー
closeModal.addEventListener('click', closeSampleModal);
cancelSample.addEventListener('click', closeSampleModal);
confirmSample.addEventListener('click', confirmSampleSelection);

// モーダルの背景クリックで閉じる
sampleModal.addEventListener('click', (event) => {
    if (event.target === sampleModal) {
        closeSampleModal();
    }
});

// 詳細設定の表示/非表示切り替え
toggleAdvancedBtn.addEventListener('click', () => {
    if (advancedSettings.style.display === 'none') {
        advancedSettings.style.display = 'block';
        toggleAdvancedBtn.textContent = '詳細設定を非表示';
    } else {
        advancedSettings.style.display = 'none';
        toggleAdvancedBtn.textContent = '詳細設定を表示';
    }
});

// Watermarkの状態に応じてSeedの有効/無効を切り替え
function updateSeedAvailability() {
    const isWatermarkEnabled = watermarkSelect.value === 'true';
    if (isWatermarkEnabled) {
        seedInput.disabled = true;
        seedInput.value = '0';
        seedInput.classList.add('bg-gray-100', 'cursor-not-allowed');
        seedWarning.classList.remove('hidden');
    } else {
        seedInput.disabled = false;
        seedInput.classList.remove('bg-gray-100', 'cursor-not-allowed');
        seedWarning.classList.add('hidden');
    }
}

// MIME Typeの状態に応じてCompression Qualityの有効/無効を切り替え
function updateCompressionQualityAvailability() {
    const isMimeTypeJPEG = mimeTypeSelect.value === 'image/jpeg';
    if (!isMimeTypeJPEG) {
        compressionQualityInput.disabled = true;
        compressionQualityInput.classList.add('bg-gray-100', 'cursor-not-allowed');
        compressionWarning.classList.remove('hidden');
    } else {
        compressionQualityInput.disabled = false;
        compressionQualityInput.classList.remove('bg-gray-100', 'cursor-not-allowed');
        compressionWarning.classList.add('hidden');
    }
}

watermarkSelect.addEventListener('change', updateSeedAvailability);
mimeTypeSelect.addEventListener('change', updateCompressionQualityAvailability);

// 初期状態を設定
updateSeedAvailability();
updateCompressionQualityAvailability();

setupDragAndDrop(personUploadArea, personInput, personUploadContent, personPreview, personPreviewImage, personName);
setupDragAndDrop(garmentUploadArea, garmentInput, garmentUploadContent, garmentPreview, garmentPreviewGrid, garmentName);

clearBtn.addEventListener('click', () => {
    personInput.value = '';
    garmentInput.value = '';
    personName.textContent = '';
    garmentName.textContent = '';
    
    // 画像プレビューをリセット
    personUploadContent.classList.remove('hidden');
    personPreview.classList.add('hidden');
    garmentUploadContent.classList.remove('hidden');
    garmentPreview.classList.add('hidden');
    garmentPreviewGrid.innerHTML = '';
    
    resultDisplay.innerHTML = '';
    multipleResults.innerHTML = '';
    multipleResults.style.display = 'none';
    resultDisplay.style.display = 'flex';
    resultSection.classList.add('hidden');
    errorMessage.classList.add('hidden');
    
    // サンプル画像の選択もリセット
    currentPersonSample = null;
    currentGarmentSample = null;
    
    // アップロードされたファイルもリセット
    uploadedGarmentFiles = [];
    
    // required属性を復元
    personInput.setAttribute('required', 'required');
    garmentInput.setAttribute('required', 'required');
    
    // 詳細設定もリセット
    document.querySelector('select[name="add_watermark"]').value = 'true';
    document.querySelector('input[name="base_steps"]').value = '32';
    document.querySelector('select[name="person_generation"]').value = 'allow_adult';
    document.querySelector('select[name="safety_setting"]').value = 'block_medium_and_above';
    document.querySelector('input[name="sample_count"]').value = '1';
    document.querySelector('input[name="seed"]').value = '0';
    document.querySelector('select[name="output_mime_type"]').value = 'image/png';
    document.querySelector('input[name="compression_quality"]').value = '75';

    
    // UI状態も更新
    updateSeedAvailability();
    updateCompressionQualityAvailability();
});

form.addEventListener('submit', async (event) => {
    event.preventDefault();
    
    // サンプル画像とファイルアップロードのどちらを使用するかをチェック
    const p = personInput.files[0];
    const garmentFiles = uploadedGarmentFiles.length > 0 ? uploadedGarmentFiles : Array.from(garmentInput.files);
    
    const hasPersonImage = p || currentPersonSample;
    const hasGarmentImage = garmentFiles.length > 0 || (currentGarmentSample && currentGarmentSample.length > 0);
    
    if (!hasPersonImage || !hasGarmentImage) {
        errorMessage.textContent = '人物画像と衣服画像の両方を選択してください（ファイルアップロードまたはサンプルから）';
        errorMessage.classList.remove('hidden');
        return;
    }

    const MAX = 10 * 1024 * 1024;
    // ファイルアップロード使用時のみサイズチェック
    if (p && p.size > MAX) {
        errorMessage.textContent = '人物画像が大きすぎます（10MBまで対応）';
        errorMessage.classList.remove('hidden');
        return;
    }
    
    for (let file of garmentFiles) {
        if (file.size > MAX) {
            errorMessage.textContent = '衣服画像が大きすぎます（10MBまで対応）';
            errorMessage.classList.remove('hidden');
            return;
        }
    }

    submitBtn.disabled = true;
    submitBtn.textContent = '生成中...';
    resultSection.classList.remove('hidden');
    
    // 前の結果をクリアしてローディングアニメーションを表示
    resultDisplay.innerHTML = '<div class="loader"></div>';
    resultDisplay.style.display = 'flex';
    multipleResults.innerHTML = '';
    multipleResults.style.display = 'none';
    errorMessage.classList.add('hidden');

    const formData = new FormData();
    
    const formElements = document.querySelectorAll('#advanced-settings input, #advanced-settings select');
    
    // サンプル画像を使用する場合の処理
    if (currentPersonSample && !p) {
        // サンプル画像のURLから画像データを取得してFormDataに追加
        try {
            const response = await fetch(currentPersonSample.url);
            const blob = await response.blob();
            formData.append('person_image', blob, 'sample_person.png');
        } catch (error) {
            console.error('Failed to load person sample image:', error);
            errorMessage.textContent = '人物サンプル画像の読み込みに失敗しました';
            errorMessage.classList.remove('hidden');
            return;
        }
    } else {
        formData.append('person_image', p);
    }
    
    if (currentGarmentSample && currentGarmentSample.length > 0 && garmentFiles.length === 0) {
        // サンプル画像のURLから画像データを取得してFormDataに追加
        try {
            for (let i = 0; i < currentGarmentSample.length; i++) {
                const sample = currentGarmentSample[i];
                const response = await fetch(sample.url);
                const blob = await response.blob();
                formData.append('garment_image', blob, 'sample_garment_' + i + '.png');
            }
        } catch (error) {
            console.error('Failed to load garment sample image:', error);
            errorMessage.textContent = '衣服サンプル画像の読み込みに失敗しました';
            errorMessage.classList.remove('hidden');
            return;
        }
    } else {
        // ファイルアップロードの場合
        for (let file of garmentFiles) {
            formData.append('garment_image', file);
        }
    }
    
    // フォームの全てのパラメータを追加
    formElements.forEach(element => {
        if (element.name && element.value) {
            formData.append(element.name, element.value);
        }
    });

    try {
        const resp = await fetch('/tryon', { method: 'POST', body: formData });
        if (!resp.ok) {
            let msg = 'HTTP ' + resp.status;
            try {
                const j = await resp.json();
                if (j && j.error) msg = j.error;
            } catch {}
            throw new Error(msg);
        }
        
        const contentType = resp.headers.get('content-type');
        if (contentType && contentType.includes('application/json')) {
                    const data = await resp.json();
        if (data.success) {
            if (data.images && data.images.length > 0) {
                    resultDisplay.style.display = 'none';
                    multipleResults.style.display = 'grid';
                    multipleResults.innerHTML = '';
                    
                    data.images.forEach((img, index) => {
                    const imgContainer = document.createElement('div');
                    imgContainer.className = 'relative';
                    
                    const imgElement = document.createElement('img');
                    imgElement.src = 'data:' + img.type + ';base64,' + img.data;
                    imgElement.alt = 'Result ' + (index + 1);
                    imgElement.className = 'w-full h-auto rounded-lg shadow-md';
                    
                    const label = document.createElement('div');
                    label.textContent = '画像 ' + (index + 1);
                    label.className = 'absolute top-2 left-2 bg-black bg-opacity-50 text-white px-2 py-1 rounded text-sm';
                    
                    const saveBtn = document.createElement('button');
                    saveBtn.textContent = '保存';
                    saveBtn.className = 'absolute top-2 right-2 bg-blue-500 hover:bg-blue-600 text-white px-2 py-1 rounded text-sm transition-colors';
                    saveBtn.onclick = (event) => {
                        event.preventDefault();
                        event.stopPropagation();
                        
                        // ボタンの状態を保存中に変更
                        const originalText = saveBtn.textContent;
                        const originalClass = saveBtn.className;
                        saveBtn.textContent = '保存中...';
                        saveBtn.className = 'absolute top-2 right-2 bg-gray-400 text-white px-2 py-1 rounded text-sm cursor-not-allowed';
                        saveBtn.disabled = true;
                        
                        // 非同期でダウンロード処理を実行
                        setTimeout(() => {
                            try {
                                const link = document.createElement('a');
                                link.href = imgElement.src;
                                // タイムスタンプ付きファイル名を生成
                                const timestamp = new Date().toISOString().replace(/[:.]/g, '-').slice(0, 19);
                                const extension = (img.type === 'image/jpeg') ? 'jpg' : 'png';
                                link.download = 'tryon-result-' + timestamp + '.' + extension;
                                document.body.appendChild(link);
                                link.click();
                                document.body.removeChild(link);
                            } catch (error) {
                                console.error('Download failed:', error);
                            } finally {
                                // ボタンの状態を元に戻す
                                setTimeout(() => {
                                    saveBtn.textContent = originalText;
                                    saveBtn.className = originalClass;
                                    saveBtn.disabled = false;
                                }, 500);
                            }
                        }, 100);
                    };
                    
                    imgContainer.appendChild(imgElement);
                    imgContainer.appendChild(label);
                    imgContainer.appendChild(saveBtn);
                    multipleResults.appendChild(imgContainer);
                });
                } else {
                    throw new Error('画像の生成に失敗しました');
                }
            } else {
                throw new Error('生成に失敗しました');
            }
        } else {
            const blob = await resp.blob();
            resultDisplay.style.display = 'flex';
            multipleResults.style.display = 'none';
            
            const imgContainer = document.createElement('div');
            imgContainer.className = 'relative w-full h-full flex items-center justify-center';
            
            const imgElement = document.createElement('img');
            imgElement.src = URL.createObjectURL(blob);
            imgElement.alt = 'Result';
            imgElement.className = 'max-w-full max-h-full object-contain rounded-lg shadow-md';
            
            const saveBtn = document.createElement('button');
            saveBtn.textContent = '保存';
            saveBtn.className = 'absolute top-2 right-2 bg-blue-500 hover:bg-blue-600 text-white px-3 py-1 rounded text-sm transition-colors';
            saveBtn.onclick = (event) => {
                event.preventDefault();
                event.stopPropagation();
                
                // ボタンの状態を保存中に変更
                const originalText = saveBtn.textContent;
                const originalClass = saveBtn.className;
                saveBtn.textContent = '保存中...';
                saveBtn.className = 'absolute top-2 right-2 bg-gray-400 text-white px-3 py-1 rounded text-sm cursor-not-allowed';
                saveBtn.disabled = true;
                
                // 非同期でダウンロード処理を実行
                setTimeout(() => {
                    try {
                        const link = document.createElement('a');
                        link.href = imgElement.src;
                        // タイムスタンプ付きファイル名を生成
                        const timestamp = new Date().toISOString().replace(/[:.]/g, '-').slice(0, 19);
                        link.download = 'tryon-result-' + timestamp + '.jpg';
                        document.body.appendChild(link);
                        link.click();
                        document.body.removeChild(link);
                    } catch (error) {
                        console.error('Download failed:', error);
                    } finally {
                        // ボタンの状態を元に戻す
                        setTimeout(() => {
                            saveBtn.textContent = originalText;
                            saveBtn.className = originalClass;
                            saveBtn.disabled = false;
                        }, 500);
                    }
                }, 100);
            };
            
            imgContainer.appendChild(imgElement);
            imgContainer.appendChild(saveBtn);
            resultDisplay.innerHTML = '';
            resultDisplay.appendChild(imgContainer);
        }
    } catch (err) {
        console.error(err);
        resultDisplay.style.display = 'flex';
        multipleResults.style.display = 'none';
        resultDisplay.innerHTML = '<span class="text-red-500 flex items-center justify-center">生成に失敗しました</span>';
        errorMessage.textContent = 'エラー: ' + err.message;
        errorMessage.classList.remove('hidden');
    } finally {
        submitBtn.disabled = false;
        submitBtn.textContent = '着せ替えを実行';
    }
});
</script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, max-age=0")
	w.Write([]byte(html))
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

	html := `<!DOCTYPE html>
<html lang="ja">
<head>
<meta charset="UTF-8"/>
<meta name="viewport" content="width=device-width, initial-scale=1.0"/>
<title>Vertex AI Imagen - 画像生成</title>
<script src="https://cdn.tailwindcss.com"></script>
<style>
body { font-family: Inter, system-ui, -apple-system, Segoe UI, Roboto, sans-serif; }
.loader{border:8px solid #f3f3f3;border-top:8px solid #6366f1;border-radius:50%;width:56px;height:56px;animation:spin 1.2s linear infinite}
@keyframes spin{0%{transform:rotate(0)}100%{transform:rotate(360deg)}}
.tooltip { position: relative; display: inline-block; }
.tooltip .tooltiptext { visibility: hidden; width: 280px; background-color: #333; color: #fff; text-center; border-radius: 6px; padding: 8px; position: absolute; z-index: 1; bottom: 125%; left: 50%; margin-left: -140px; opacity: 0; transition: opacity 0.3s; font-size: 12px; line-height: 1.4; }
.tooltip .tooltiptext::after { content: ""; position: absolute; top: 100%; left: 50%; margin-left: -5px; border-width: 5px; border-style: solid; border-color: #333 transparent transparent transparent; }
.tooltip:hover .tooltiptext { visibility: visible; opacity: 1; }
.info-icon { display: inline-flex; align-items: center; justify-content: center; width: 16px; height: 16px; border-radius: 50%; background-color: #6366f1; color: white; font-size: 12px; font-weight: bold; margin-left: 4px; cursor: help; }
.result-preview{width:100%;height:400px;background:#f3f4f6;border:2px dashed #d1d5db;display:flex;align-items:center;justify-content:center;overflow:hidden;border-radius:8px}
.result-preview img{max-width:100%;max-height:100%;object-fit:contain}
</style>
</head>
<body class="bg-gray-50 text-gray-800">
<div class="container mx-auto p-4 md:p-8 max-w-4xl">
<!-- ナビゲーションバー -->
<nav class="bg-white shadow-sm rounded-lg mb-6 p-4">
<div class="flex flex-wrap justify-center gap-3">
<button onclick="location.href='/'" class="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors font-medium shadow-sm">
<svg class="w-4 h-4 inline mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"/></svg>
Virtual Try-On
</button>
<button onclick="location.href='/imagen'" class="px-4 py-2 bg-green-700 text-white rounded-lg shadow-md font-medium ring-2 ring-green-300">
<svg class="w-4 h-4 inline mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"/></svg>
Imagen画像生成
</button>
<button onclick="location.href='/veo'" class="px-4 py-2 bg-purple-600 text-white rounded-lg hover:bg-purple-700 transition-colors font-medium shadow-sm">
<svg class="w-4 h-4 inline mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z"/></svg>
Veo動画生成
</button>
<button onclick="location.href='/nanobanana/image-editing'" class="px-4 py-2 bg-orange-600 text-white rounded-lg hover:bg-orange-700 transition-colors font-medium shadow-sm">
<svg class="w-4 h-4 inline mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/></svg>
Nanobanana画像編集
</button>
</div>
</nav>

<header class="text-center mb-8">
<h1 class="text-3xl md:text-4xl font-bold text-gray-900">Vertex AI Imagen</h1>
<p class="text-gray-600 mt-2">テキストプロンプトから画像を生成します</p>
</header>
<main class="bg-white p-6 md:p-8 rounded-2xl shadow-lg">
<form id="imagen-form">
<div class="space-y-6 mb-6">
<div>
<label class="block text-lg font-semibold mb-2 text-gray-700">
プロンプト
<div class="tooltip">
<span class="info-icon">?</span>
<span class="tooltiptext">生成したい画像の詳細な説明を入力してください。具体的で詳細な説明ほど、意図した画像が生成されやすくなります。</span>
</div>
</label>
<textarea id="prompt" name="prompt" required rows="4" 
class="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500" 
placeholder="例: A beautiful landscape with mountains and a lake during sunset, highly detailed, photorealistic"></textarea>
</div>
<div>
<label class="block text-lg font-semibold mb-2 text-gray-700">
Imagenモデル
<div class="tooltip">
<span class="info-icon">?</span>
<span class="tooltiptext">使用するImagenモデルを選択してください。新しいバージョンほど高品質な画像を生成できますが、処理時間が長くなる場合があります。` + locationInfo + `</span>
</div>
</label>
<select id="imagenModel" name="imagenModel" class="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500">
` + modelOptions.String() + `
</select>
</div>
</div>
<!-- 詳細設定セクション -->
<div id="imagen-advanced-settings" class="mb-6 p-4 bg-gray-50 rounded-lg" style="display: none;">
<h3 class="text-lg font-semibold mb-4 text-gray-700">詳細設定</h3>
<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
<div>
<label class="block text-sm font-medium mb-1 text-gray-600">
生成画像数
<div class="tooltip">
<span class="info-icon">?</span>
<span class="tooltiptext">一度に生成する画像の枚数です（1-4枚）。複数生成すると異なるバリエーションが得られますが、生成時間とコストが増加します。</span>
</div>
</label>
<input type="number" name="numberOfImages" min="1" max="4" value="1" class="w-full px-3 py-2 border border-gray-300 rounded-md">
</div>
<div>
<label class="block text-sm font-medium mb-1 text-gray-600">
アスペクト比
<div class="tooltip">
<span class="info-icon">?</span>
<span class="tooltiptext">生成される画像の縦横比を指定します。用途に応じて最適な比率を選択してください。</span>
</div>
</label>
<select name="aspectRatio" class="w-full px-3 py-2 border border-gray-300 rounded-md">
<option value="1:1">1:1 (正方形)</option>
<option value="3:4">3:4 (縦長)</option>
<option value="4:3">4:3 (横長)</option>
<option value="9:16">9:16 (縦長・モバイル向け)</option>
<option value="16:9">16:9 (横長・ワイド)</option>
</select>
</div>
<div>
<label class="block text-sm font-medium mb-1 text-gray-600">
ネガティブプロンプト
<div class="tooltip">
<span class="info-icon">?</span>
<span class="tooltiptext">生成画像に含めたくない要素を指定できます。例：「blurry, low quality, distorted」</span>
</div>
</label>
<input type="text" name="negativePrompt" placeholder="除外したい要素を入力" class="w-full px-3 py-2 border border-gray-300 rounded-md">
</div>
<div>
<label class="block text-sm font-medium mb-1 text-gray-600">
シード値
<div class="tooltip">
<span class="info-icon">?</span>
<span class="tooltiptext">再現性のある結果を得るための数値です。同じシード値を使用すると同じ結果が得られます。0の場合はランダムになります。</span>
</div>
</label>
<input type="number" name="seed" value="0" class="w-full px-3 py-2 border border-gray-300 rounded-md">
</div>
<div class="md:col-span-2">
<label class="inline-flex items-center">
<input type="checkbox" name="includeRaiReason" class="rounded border-gray-300 text-indigo-600 shadow-sm focus:border-indigo-300 focus:ring focus:ring-indigo-200 focus:ring-opacity-50">
<span class="ml-2 text-sm text-gray-600">
AI安全性チェック結果を含める
<div class="tooltip inline">
<span class="info-icon">?</span>
<span class="tooltiptext">画像がResponsible AIチェックに失敗した場合、その理由を含めるかどうかを指定します。</span>
</div>
</span>
</label>
</div>
</div>
</div>
<div class="text-center mb-6">
<button type="button" id="toggle-imagen-advanced" class="text-sm text-indigo-600 hover:text-indigo-800 mb-4">
詳細設定を表示
</button>
</div>

<!-- 実行ボタン（メイン） -->
<div class="text-center mb-8">
<button type="submit" id="submit-btn"
class="bg-gradient-to-r from-indigo-500 to-blue-600 text-white font-bold py-4 px-12 rounded-full hover:shadow-xl transform hover:-translate-y-0.5 transition-all text-lg">
画像を生成
</button>
</div>

<!-- クリアボタン（サブ） -->
<div class="text-center">
<button type="button" id="clear-imagen-btn"
class="px-6 py-2 text-sm rounded-lg border border-gray-300 text-gray-600 hover:bg-gray-50 hover:border-gray-400 transition-all">
<svg class="w-4 h-4 inline mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/></svg>
プロンプトクリア
</button>
</div>
</form>
<div id="result-section" class="mt-10 hidden">
<h2 class="text-2xl font-bold text-center mb-4 text-gray-800">生成結果</h2>
<div id="result-display" class="result-preview"></div>
<div id="multiple-results" class="grid grid-cols-1 md:grid-cols-2 gap-4 mt-4" style="display: none;"></div>
</div>
<div id="error-message" class="mt-6 hidden bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded-lg"></div>
</main>

</div>
<script>
const form = document.getElementById('imagen-form');
const promptInput = document.getElementById('prompt');
const imagenModelSelect = document.getElementById('imagenModel');
const resultSection = document.getElementById('result-section');
const resultDisplay = document.getElementById('result-display');
const multipleResults = document.getElementById('multiple-results');
const errorMessage = document.getElementById('error-message');
const submitBtn = document.getElementById('submit-btn');
const clearImagenBtn = document.getElementById('clear-imagen-btn');
const toggleAdvancedBtn = document.getElementById('toggle-imagen-advanced');
const advancedSettings = document.getElementById('imagen-advanced-settings');

// 詳細設定の表示/非表示切り替え
toggleAdvancedBtn.addEventListener('click', () => {
    if (advancedSettings.style.display === 'none') {
        advancedSettings.style.display = 'block';
        toggleAdvancedBtn.textContent = '詳細設定を非表示';
    } else {
        advancedSettings.style.display = 'none';
        toggleAdvancedBtn.textContent = '詳細設定を表示';
    }
});

// クリアボタン
clearImagenBtn.addEventListener('click', () => {
    if (confirm('プロンプトと設定をクリアしますか？')) {
        promptInput.value = '';
        imagenModelSelect.selectedIndex = 0;
        resultDisplay.innerHTML = '';
        resultSection.classList.add('hidden');
        errorMessage.classList.add('hidden');
        multipleResults.innerHTML = '';
        multipleResults.style.display = 'none';
        
        // 詳細設定もリセット
        document.querySelector('input[name="numberOfImages"]').value = '1';
        document.querySelector('select[name="aspectRatio"]').selectedIndex = 0;
        document.querySelector('input[name="negativePrompt"]').value = '';
        document.querySelector('input[name="seed"]').value = '0';
        document.querySelector('input[name="includeRaiReason"]').checked = false;
    }
});

form.addEventListener('submit', async (event) => {
    event.preventDefault();
    
    const prompt = promptInput.value.trim();
    if (!prompt) {
        errorMessage.textContent = 'プロンプトを入力してください';
        errorMessage.classList.remove('hidden');
        return;
    }

    submitBtn.disabled = true;
    submitBtn.textContent = '生成中...';
    resultSection.classList.remove('hidden');
    
    // 前の結果をクリアしてローディングアニメーションを表示
    resultDisplay.innerHTML = '<div class="loader"></div>';
    resultDisplay.style.display = 'flex';
    multipleResults.innerHTML = '';
    multipleResults.style.display = 'none';
    errorMessage.classList.add('hidden');

    const formData = new FormData();
    formData.append('prompt', prompt);
    formData.append('imagenModel', imagenModelSelect.value);
    
    // 詳細設定パラメータを追加
    const formElements = document.querySelectorAll('#imagen-advanced-settings input, #imagen-advanced-settings select');
    formElements.forEach(element => {
        if (element.type === 'checkbox') {
            formData.append(element.name, element.checked ? 'true' : 'false');
        } else if (element.name && element.value) {
            formData.append(element.name, element.value);
        }
    });

    try {
        const resp = await fetch('/imagen', { method: 'POST', body: formData });
        if (!resp.ok) {
            let msg = 'HTTP ' + resp.status;
            try {
                const j = await resp.json();
                if (j && j.error) msg = j.error;
            } catch {}
            throw new Error(msg);
        }
        
        const data = await resp.json();
        if (data.success && data.images && data.images.length > 0) {
            if (data.images.length === 1) {
                // 単一画像の場合
                const img = data.images[0];
                
                const imgContainer = document.createElement('div');
                imgContainer.className = 'relative w-full h-full flex items-center justify-center';
                
                const imgElement = document.createElement('img');
                imgElement.src = 'data:' + img.type + ';base64,' + img.data;
                imgElement.alt = 'Generated Image';
                imgElement.className = 'max-w-full max-h-full object-contain rounded-lg shadow-md';
                
                const saveBtn = document.createElement('button');
                saveBtn.textContent = '保存';
                saveBtn.className = 'absolute top-2 right-2 bg-blue-500 hover:bg-blue-600 text-white px-3 py-1 rounded text-sm transition-colors';
                saveBtn.onclick = (event) => {
                    event.preventDefault();
                    event.stopPropagation();
                    
                    const originalText = saveBtn.textContent;
                    const originalClass = saveBtn.className;
                    saveBtn.textContent = '保存中...';
                    saveBtn.className = 'absolute top-2 right-2 bg-gray-400 text-white px-3 py-1 rounded text-sm cursor-not-allowed';
                    saveBtn.disabled = true;
                    
                    setTimeout(() => {
                        try {
                            const link = document.createElement('a');
                            link.href = imgElement.src;
                            const timestamp = new Date().toISOString().replace(/[:.]/g, '-').slice(0, 19);
                            const extension = (img.type === 'image/jpeg') ? 'jpg' : 'png';
                            link.download = 'imagen-result-' + timestamp + '.' + extension;
                            document.body.appendChild(link);
                            link.click();
                            document.body.removeChild(link);
                        } catch (error) {
                            console.error('Download failed:', error);
                        } finally {
                            setTimeout(() => {
                                saveBtn.textContent = originalText;
                                saveBtn.className = originalClass;
                                saveBtn.disabled = false;
                            }, 500);
                        }
                    }, 100);
                };
                
                imgContainer.appendChild(imgElement);
                imgContainer.appendChild(saveBtn);
                resultDisplay.innerHTML = '';
                resultDisplay.style.display = 'flex';
                resultDisplay.appendChild(imgContainer);
                multipleResults.style.display = 'none';
            } else {
                // 複数画像の場合
                resultDisplay.style.display = 'none';
                multipleResults.style.display = 'grid';
                multipleResults.innerHTML = '';
                
                data.images.forEach((img, index) => {
                    const imgContainer = document.createElement('div');
                    imgContainer.className = 'relative';
                    
                    const imgElement = document.createElement('img');
                    imgElement.src = 'data:' + img.type + ';base64,' + img.data;
                    imgElement.alt = 'Generated Image ' + (index + 1);
                    imgElement.className = 'w-full h-auto rounded-lg shadow-md';
                    
                    const label = document.createElement('div');
                    label.textContent = '画像 ' + (index + 1);
                    label.className = 'absolute top-2 left-2 bg-black bg-opacity-50 text-white px-2 py-1 rounded text-sm';
                    
                    const saveBtn = document.createElement('button');
                    saveBtn.textContent = '保存';
                    saveBtn.className = 'absolute top-2 right-2 bg-blue-500 hover:bg-blue-600 text-white px-2 py-1 rounded text-sm transition-colors';
                    saveBtn.onclick = (event) => {
                        event.preventDefault();
                        event.stopPropagation();
                        
                        const originalText = saveBtn.textContent;
                        const originalClass = saveBtn.className;
                        saveBtn.textContent = '保存中...';
                        saveBtn.className = 'absolute top-2 right-2 bg-gray-400 text-white px-2 py-1 rounded text-sm cursor-not-allowed';
                        saveBtn.disabled = true;
                        
                        setTimeout(() => {
                            try {
                                const link = document.createElement('a');
                                link.href = imgElement.src;
                                const timestamp = new Date().toISOString().replace(/[:.]/g, '-').slice(0, 19);
                                const extension = (img.type === 'image/jpeg') ? 'jpg' : 'png';
                                link.download = 'imagen-result-' + timestamp + '-' + (index + 1) + '.' + extension;
                                document.body.appendChild(link);
                                link.click();
                                document.body.removeChild(link);
                            } catch (error) {
                                console.error('Download failed:', error);
                            } finally {
                                setTimeout(() => {
                                    saveBtn.textContent = originalText;
                                    saveBtn.className = originalClass;
                                    saveBtn.disabled = false;
                                }, 500);
                            }
                        }, 100);
                    };
                    
                    imgContainer.appendChild(imgElement);
                    imgContainer.appendChild(label);
                    imgContainer.appendChild(saveBtn);
                    multipleResults.appendChild(imgContainer);
                });
            }
        } else {
            throw new Error('画像の生成に失敗しました');
        }
    } catch (err) {
        console.error(err);
        resultDisplay.innerHTML = '<span class="text-red-500 flex items-center justify-center">生成に失敗しました</span>';
        resultDisplay.style.display = 'flex';
        multipleResults.style.display = 'none';
        errorMessage.textContent = 'エラー: ' + err.message;
        errorMessage.classList.remove('hidden');
    } finally {
        submitBtn.disabled = false;
        submitBtn.textContent = '画像を生成';
    }
});
</script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, max-age=0")
	w.Write([]byte(html))
}
