package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"tryon-demo/internal/application/services"
	"tryon-demo/internal/application/usecases"
)

const maxFileSize = 25 * 1024 * 1024 // 25MB

type TryOnHandler struct {
	tryOnUseCase     *usecases.TryOnUseCase
	parameterService *services.ParameterService
	isSkipGenerate   bool
	location         string // Vertex AIのリージョン情報
}

func NewTryOnHandler(
	tryOnUseCase *usecases.TryOnUseCase,
	parameterService *services.ParameterService,
	isSkipGenerate bool,
	location string,
) *TryOnHandler {
	return &TryOnHandler{
		tryOnUseCase:     tryOnUseCase,
		parameterService: parameterService,
		isSkipGenerate:   isSkipGenerate,
		location:         location,
	}
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
	if h.isSkipGenerate {
		log.Printf("[DEBUG] Skip generate mode enabled")

		parameters := h.parameterService.ParseFromRequest(r)
		sampleCount := parameters.SampleCount
		log.Printf("[DEBUG] Requested sample count: %d", sampleCount)

		images, err := h.getSampleImages(sampleCount)
		if err != nil {
			log.Printf("[ERROR] Failed to get sample images: %v", err)
			h.sendError(w, "サンプル画像の取得に失敗しました", http.StatusInternalServerError)
			return
		}

		log.Printf("[DEBUG] Got %d images from getSampleImages", len(images))

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store, max-age=0")

		response := h.createResponse(images)
		log.Printf("[DEBUG] Created response, encoding JSON...")

		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("[ERROR] Failed to encode JSON response: %v", err)
			h.sendError(w, "レスポンスの生成に失敗しました", http.StatusInternalServerError)
			return
		}

		log.Printf("[DEBUG] Successfully sent JSON response")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxFileSize)
	if err := r.ParseMultipartForm(maxFileSize); err != nil {
		h.sendError(w, "画像が大きすぎます（25MBまで対応）", http.StatusRequestEntityTooLarge)
		return
	}

	personFile, _, err := r.FormFile("person_image")
	if err != nil {
		h.sendError(w, "人物画像を選んでください", http.StatusBadRequest)
		return
	}
	defer personFile.Close()

	garmentFile, _, err := r.FormFile("garment_image")
	if err != nil {
		h.sendError(w, "衣服画像を選んでください", http.StatusBadRequest)
		return
	}
	defer garmentFile.Close()

	personData, err := io.ReadAll(personFile)
	if err != nil {
		h.sendError(w, "人物画像の読み込みに失敗しました", http.StatusInternalServerError)
		return
	}

	garmentData, err := io.ReadAll(garmentFile)
	if err != nil {
		h.sendError(w, "衣服画像の読み込みに失敗しました", http.StatusInternalServerError)
		return
	}

	parameters := h.parameterService.ParseFromRequest(r)

	input := usecases.TryOnInput{
		PersonImageData:  personData,
		GarmentImageData: garmentData,
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

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store, max-age=0")

	response := h.createResponseWithStorageInfo(output.Images, parameters.StorageURI)

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

func (h *TryOnHandler) createResponseWithStorageInfo(imagesOutput []usecases.ImageOutput, storageURI string) map[string]any {
	log.Printf("[DEBUG] createResponseWithStorageInfo called with %d images, storageURI: %s", len(imagesOutput), storageURI)

	// Storage URI指定時の処理
	if storageURI != "" {
		log.Printf("[INFO] Storage URI specified, creating success response: %s", storageURI)
		response := map[string]any{
			"success":    true,
			"images":     []map[string]string{}, // 空の画像配列
			"storageUri": storageURI,
		}
		return response
	}

	// Storage URI未指定時は通常の処理
	return h.createResponse(imagesOutput)
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
				URL:         "/static/sample_images/person/sample_men.png",
				Category:    "person",
			},
			{
				ID:          "person_men_50",
				Name:        "男性 (50代)",
				Description: "フォーマルな服装の中年男性",
				URL:         "/static/sample_images/person/sample_men_50.png",
				Category:    "person",
			},
			{
				ID:          "person_women_20",
				Name:        "女性 (20代)",
				Description: "カジュアルな服装の若い女性",
				URL:         "/static/sample_images/person/sample_women_20.png",
				Category:    "person",
			},
			{
				ID:          "person_women_70",
				Name:        "女性 (70代)",
				Description: "エレガントな服装のシニア女性",
				URL:         "/static/sample_images/person/sample_women_70.png",
				Category:    "person",
			},
		}
	} else {
		samples = []SampleImage{
			{
				ID:          "garment_tops",
				Name:        "トップス (ベーシック)",
				Description: "シンプルなデザインのトップス",
				URL:         "/static/sample_images/garment/sample_tops.png",
				Category:    "garment",
			},
			{
				ID:          "garment_tops_hade",
				Name:        "トップス (派手)",
				Description: "カラフルで目立つデザインのトップス",
				URL:         "/static/sample_images/garment/sample_tops_hade.png",
				Category:    "garment",
			},
			{
				ID:          "garment_pants",
				Name:        "パンツ",
				Description: "カジュアルなパンツ",
				URL:         "/static/sample_images/garment/sample_pants.png",
				Category:    "garment",
			},
			{
				ID:          "garment_shoes",
				Name:        "シューズ",
				Description: "スタイリッシュなシューズ",
				URL:         "/static/sample_images/garment/sample_shoes.png",
				Category:    "garment",
			},
			{
				ID:          "garment_shoes_double",
				Name:        "シューズ（両足）",
				Description: "スタイリッシュなシューズ（両足）",
				URL:         "/static/sample_images/garment/sample_shoes_double.png",
				Category:    "garment",
			},
			{
				ID:          "garment_neckless",
				Name:        "ネックレス",
				Description: "エレガントなネックレス",
				URL:         "/static/sample_images/garment/sample_neckless.png",
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

func (h *TryOnHandler) HandleIndex(w http.ResponseWriter, r *http.Request) {
	// 現在のVertex AIリージョン情報をツールチップに含める
	locationInfo := fmt.Sprintf(" 現在のVertex AIリージョン: %s", h.location)

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
</style>
</head>
<body class="bg-gray-50 text-gray-800">
<div class="container mx-auto p-4 md:p-8 max-w-5xl">
<header class="text-center mb-8">
<h1 class="text-3xl md:text-4xl font-bold text-gray-900">Vertex AI Virtual Try-On</h1>
<p class="text-gray-600 mt-2">人物と衣服の画像をアップロードして、着せ替えを試そう。</p>
</header>
<main class="bg-white p-6 md:p-8 rounded-2xl shadow-lg">
<form id="tryon-form">
<div class="grid grid-cols-1 md:grid-cols-2 gap-6 mb-6">
<div>
<label class="block text-lg font-semibold mb-2 text-gray-700">1. 人物画像</label>
<div id="person-preview" class="preview-box rounded-lg mb-3"><span class="text-gray-500">プレビュー</span></div>
<div class="flex flex-wrap gap-2 mb-2">
<label class="inline-flex items-center px-4 py-2 rounded-full bg-gradient-to-r from-indigo-500 to-blue-500 text-white shadow hover:shadow-lg cursor-pointer">
<svg class="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6H17a3 3 0 010 6h-1m-4 5V10m0 0l-2 2m2-2l2 2"/></svg>
<span>ファイルを選択</span>
<input type="file" id="person-image" name="person_image" accept="image/*" class="hidden">
</label>
<button type="button" id="person-sample-btn" class="inline-flex items-center px-4 py-2 rounded-full bg-gradient-to-r from-green-500 to-teal-500 text-white shadow hover:shadow-lg">
<svg class="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"/></svg>
<span>サンプルから選択</span>
</button>
</div>
<span id="person-name" class="ml-2 text-sm text-gray-500"></span>
</div>
<div>
<label class="block text-lg font-semibold mb-2 text-gray-700">2. 衣服画像</label>
<div id="garment-preview" class="preview-box rounded-lg mb-3"><span class="text-gray-500">プレビュー</span></div>
<div class="flex flex-wrap gap-2 mb-2">
<label class="inline-flex items-center px-4 py-2 rounded-full bg-gradient-to-r from-indigo-500 to-blue-500 text-white shadow hover:shadow-lg cursor-pointer">
<svg class="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6H17a3 3 0 010 6h-1m-4 5V10m0 0l-2 2m2-2l2 2"/></svg>
<span>ファイルを選択</span>
<input type="file" id="garment-image" name="garment_image" accept="image/*" class="hidden">
</label>
<button type="button" id="garment-sample-btn" class="inline-flex items-center px-4 py-2 rounded-full bg-gradient-to-r from-green-500 to-teal-500 text-white shadow hover:shadow-lg">
<svg class="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"/></svg>
<span>サンプルから選択</span>
</button>
</div>
<span id="garment-name" class="ml-2 text-sm text-gray-500"></span>
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
<div class="md:col-span-2">
<label class="block text-sm font-medium mb-1 text-gray-600">
Storage URI (オプション)
<div class="tooltip">
<span class="info-icon">?</span>
` + fmt.Sprintf(`<span class="tooltiptext">生成画像をGoogle Cloud Storageに保存する場合のバケットパスです。空白の場合は直接レスポンスで返されます。形式例: gs://your-bucket/path/ ※重要：バケットとVertex AIのリージョンを一致させてください（不一致の場合はエラーになります）%s</span>`, locationInfo) + `
</div>
</label>
<input type="text" name="storage_uri" placeholder="gs://your-bucket/path/" class="w-full px-3 py-2 border border-gray-300 rounded-md">
</div>
</div>
</div>
<div class="text-center space-y-3">
<button type="button" id="toggle-advanced" class="text-sm text-indigo-600 hover:text-indigo-800 mb-2">
詳細設定を表示
</button>
<div>
<div>
<button type="submit" id="submit-btn"
class="bg-gradient-to-r from-indigo-500 to-blue-600 text-white font-bold py-3 px-8 rounded-full hover:shadow-xl transform hover:-translate-y-0.5 transition-all">
着せ替えを実行
</button>
</div>
<div>
<button type="button" id="clear-btn"
class="px-4 py-2 text-sm rounded-full border border-gray-300 text-gray-700 hover:bg-gray-50">
クリア
</button>
</div>
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
</div>
<div class="flex-1 overflow-y-auto p-6">
<div id="sample-grid" class="grid grid-cols-2 md:grid-cols-3 gap-6">
<!-- サンプル画像がここに動的に追加される -->
</div>
</div>
<div class="p-6 border-t border-gray-200 text-center">
<button id="cancel-sample" class="px-6 py-2 bg-gray-300 text-gray-700 rounded-lg hover:bg-gray-400">キャンセル</button>
</div>
</div>
</div>
</div>

<footer class="text-center mt-8 text-gray-500 text-sm">
</footer>
</div>
<script>
const form = document.getElementById('tryon-form');
const personInput = document.getElementById('person-image');
const garmentInput = document.getElementById('garment-image');
const personPreview = document.getElementById('person-preview');
const garmentPreview = document.getElementById('garment-preview');
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

// 現在選択中のサンプル画像を追跡するための変数
let currentPersonSample = null;
let currentGarmentSample = null;
let currentModalCategory = null;

function setupPreview(input, previewElement, nameLabel) {
    input.addEventListener('change', (event) => {
        const file = event.target.files[0];
        nameLabel.textContent = file ? file.name : '';
        if (file) {
            const reader = new FileReader();
            reader.onload = (e) => {
                previewElement.innerHTML = '<img src="' + e.target.result + '" alt="Preview">';
            };
            reader.readAsDataURL(file);
        } else {
            previewElement.innerHTML = '<span class="text-gray-500">プレビュー</span>';
        }
    });
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
    sampleModal.classList.remove('hidden');
    
    // サンプル画像を読み込んで表示
    loadSampleImages(category).then(samples => {
        sampleGrid.innerHTML = '';
        samples.forEach(sample => {
            const sampleItem = document.createElement('div');
            sampleItem.className = 'border rounded-lg p-4 cursor-pointer hover:border-blue-500 hover:shadow-md transition-all';
            sampleItem.innerHTML = 
                '<div class="w-full h-48 bg-gray-100 rounded mb-3 flex items-center justify-center overflow-hidden">' +
                '<img src="' + sample.url + '" alt="' + sample.name + '" class="max-w-full max-h-full object-contain">' +
                '</div>' +
                '<h3 class="font-semibold text-sm text-gray-800 mb-1">' + sample.name + '</h3>' +
                '<p class="text-xs text-gray-600 leading-relaxed">' + sample.description + '</p>';
            
            sampleItem.addEventListener('click', () => {
                selectSampleImage(sample, category);
            });
            
            sampleGrid.appendChild(sampleItem);
        });
    });
}

function selectSampleImage(sample, category) {
    if (category === 'person') {
        currentPersonSample = sample;
        personPreview.innerHTML = '<img src="' + sample.url + '" alt="' + sample.name + '">';
        personName.textContent = sample.name + ' (サンプル)';
        // ファイル入力をクリアしてサンプル使用を示す
        personInput.value = '';
        personInput.removeAttribute('required');
    } else {
        currentGarmentSample = sample;
        garmentPreview.innerHTML = '<img src="' + sample.url + '" alt="' + sample.name + '">';
        garmentName.textContent = sample.name + ' (サンプル)';
        // ファイル入力をクリアしてサンプル使用を示す
        garmentInput.value = '';
        garmentInput.removeAttribute('required');
    }
    
    // モーダルを閉じる
    sampleModal.classList.add('hidden');
}

function closeSampleModal() {
    sampleModal.classList.add('hidden');
    currentModalCategory = null;
}

// サンプル画像ボタンのイベントリスナー
personSampleBtn.addEventListener('click', () => {
    showSampleModal('person');
});

garmentSampleBtn.addEventListener('click', () => {
    showSampleModal('garment');
});

// モーダル関連のイベントリスナー
closeModal.addEventListener('click', closeSampleModal);
cancelSample.addEventListener('click', closeSampleModal);

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

setupPreview(personInput, personPreview, personName);
setupPreview(garmentInput, garmentPreview, garmentName);

clearBtn.addEventListener('click', () => {
    personInput.value = '';
    garmentInput.value = '';
    personName.textContent = '';
    garmentName.textContent = '';
    personPreview.innerHTML = '<span class="text-gray-500">プレビュー</span>';
    garmentPreview.innerHTML = '<span class="text-gray-500">プレビュー</span>';
    resultDisplay.innerHTML = '';
    multipleResults.innerHTML = '';
    multipleResults.style.display = 'none';
    resultDisplay.style.display = 'flex';
    resultSection.classList.add('hidden');
    errorMessage.classList.add('hidden');
    
    // サンプル画像の選択もリセット
    currentPersonSample = null;
    currentGarmentSample = null;
    
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
    document.querySelector('input[name="storage_uri"]').value = '';
    
    // UI状態も更新
    updateSeedAvailability();
    updateCompressionQualityAvailability();
});

form.addEventListener('submit', async (event) => {
    event.preventDefault();
    
    // サンプル画像とファイルアップロードのどちらを使用するかをチェック
    const p = personInput.files[0];
    const g = garmentInput.files[0];
    
    const hasPersonImage = p || currentPersonSample;
    const hasGarmentImage = g || currentGarmentSample;
    
    if (!hasPersonImage || !hasGarmentImage) {
        errorMessage.textContent = '人物画像と衣服画像の両方を選択してください（ファイルアップロードまたはサンプルから）';
        errorMessage.classList.remove('hidden');
        return;
    }

    const MAX = 25 * 1024 * 1024;
    // ファイルアップロード使用時のみサイズチェック
    if ((p && p.size > MAX) || (g && g.size > MAX)) {
        errorMessage.textContent = '画像が大きすぎます（25MBまで対応）';
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
    
    // StorageURIを事前に初期化してフォーム要素から取得
    let storageUri = '';
    const formElements = document.querySelectorAll('#advanced-settings input, #advanced-settings select');
    formElements.forEach(element => {
        if (element.name === 'storage_uri' && element.value) {
            storageUri = element.value;
        }
    });
    
    // デバッグ用ログ
    if (storageUri) {
        console.log('[DEBUG] StorageURI specified:', storageUri);
    } else {
        console.log('[DEBUG] StorageURI not specified, images will be returned in response');
    }
    
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
    
    if (currentGarmentSample && !g) {
        // サンプル画像のURLから画像データを取得してFormDataに追加
        try {
            const response = await fetch(currentGarmentSample.url);
            const blob = await response.blob();
            formData.append('garment_image', blob, 'sample_garment.png');
        } catch (error) {
            console.error('Failed to load garment sample image:', error);
            errorMessage.textContent = '衣服サンプル画像の読み込みに失敗しました';
            errorMessage.classList.remove('hidden');
            return;
        }
    } else {
        formData.append('garment_image', g);
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
                // Storage URIが指定されている場合の処理
                if (data.storageUri) {
                    // Storage URI指定時は保存成功メッセージを表示
                    resultDisplay.style.display = 'flex';
                    multipleResults.style.display = 'none';
                    resultDisplay.innerHTML = '<div class="text-center p-8">' +
                        '<div class="text-green-600 text-6xl mb-4">✓</div>' +
                        '<h3 class="text-xl font-semibold text-gray-800 mb-2">画像生成が完了しました</h3>' +
                        '<p class="text-gray-600 mb-4">生成された画像は以下の場所に保存されました:</p>' +
                        '<p class="text-blue-600 font-mono bg-blue-50 px-4 py-2 rounded-lg">' + data.storageUri + '</p>' +
                        '<p class="text-sm text-gray-500 mt-4">Google Cloud Storageで画像を確認してください</p>' +
                        '</div>';
                } else if (data.images && data.images.length > 0) {
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
                    // Storage URIも画像データもない場合
                    throw new Error('画像の生成または保存に失敗しました');
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
