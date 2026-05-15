package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"tryon-demo/internal/application/usecases"
)

// HandleVeo - 動画生成API
func (h *VeoHandler) HandleVeo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.sendError(w, "POST method required", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxFileSize)
	if err := r.ParseMultipartForm(maxFileSize); err != nil {
		h.sendError(w, "画像が大きすぎます（10MBまで対応）", http.StatusRequestEntityTooLarge)
		return
	}

	// 画像プロンプト（オプション）
	imagenPrompt := r.FormValue("imagenPrompt")

	// 動画プロンプト（必須）
	videoPrompt := r.FormValue("videoPrompt")
	if videoPrompt == "" {
		h.sendError(w, "動画プロンプトを入力してください", http.StatusBadRequest)
		return
	}

	veoModel := r.FormValue("veoModel")
	if veoModel == "" {
		h.sendError(w, "Veoモデルを選択してください", http.StatusBadRequest)
		return
	}

	isValidVeoModel := h.isValidVeoModel(veoModel)
	if !isValidVeoModel {
		h.sendError(w, "無効なモデルです", http.StatusBadRequest)
		return
	}

	// 画像ファイルまたはプロンプトのいずれかは必須
	imageFile, imageFileHeader, err := r.FormFile("image")
	hasImageFile := err == nil

	if !hasImageFile && imagenPrompt == "" {
		h.sendError(w, "画像ファイルまたは画像生成プロンプトのいずれかを指定してください", http.StatusBadRequest)
		return
	}

	var imageData []byte
	var imageMimeType string

	if hasImageFile {
		defer imageFile.Close()
		imageMimeType = imageFileHeader.Header.Get("Content-Type")

		imageData, err = io.ReadAll(imageFile)
		if err != nil {
			h.sendError(w, "画像の読み込みに失敗しました", http.StatusInternalServerError)
			return
		}
	}

	// VeoUseCaseの入力を準備
	input := usecases.VeoInput{
		ImagenPrompt:  imagenPrompt,
		ImagenModel:   h.getDefaultImagenModelForVeo(), // サーバー側で固定
		ImageData:     imageData,
		ImageMimeType: imageMimeType,
		VideoPrompt:   videoPrompt,
		VideoModel:    veoModel,
	}

	output, err := h.veoUseCase.Execute(r.Context(), input)
	if err != nil {
		log.Printf("Video generation failed: %v", err)

		if h.isQuotaError(err) {
			h.sendError(w, "現在サーバーが混雑しています。しばらく待ってから再試行してください。", http.StatusTooManyRequests)
			return
		}

		h.sendError(w, fmt.Sprintf("動画生成に失敗しました: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store, max-age=0")

	response := h.createVeoResponse(output.Videos)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode JSON response: %v", err)
		h.sendError(w, "レスポンスの生成に失敗しました", http.StatusInternalServerError)
		return
	}
}

// createVeoResponse - Veo用のレスポンスを生成
func (h *VeoHandler) createVeoResponse(videosData [][]byte) map[string]any {
	log.Printf("[DEBUG] createVeoResponse called with %d videos", len(videosData))

	if len(videosData) == 0 {
		log.Printf("[WARNING] No video data")
		return map[string]any{
			"success": false,
			"error":   "動画データがありません",
		}
	}

	videos := make([]map[string]string, 0, len(videosData))
	totalSize := 0
	for i, videoData := range videosData {
		if len(videoData) == 0 {
			log.Printf("[WARNING] Empty video data at index %d", i)
			continue
		}

		base64Data := base64.StdEncoding.EncodeToString(videoData)
		totalSize += len(videoData)
		log.Printf("[DEBUG] Video %d: size=%d bytes, base64 length=%d characters", i, len(videoData), len(base64Data))

		videos = append(videos, map[string]string{
			"data": base64Data,
			"type": "video/mp4",
		})
	}

	if len(videos) == 0 {
		log.Printf("[WARNING] All video data is empty")
		return map[string]any{
			"success": false,
			"error":   "すべての動画データが空です",
		}
	}

	log.Printf("[DEBUG] Total %d videos, total size: %d bytes", len(videos), totalSize)

	response := map[string]any{
		"success": true,
		"videos":  videos,
		"model":   h.getDefaultVeoModel(),
	}

	return response
}

// isValidVeoModel - 指定されたモデルIDが有効かどうかチェック
func (h *VeoHandler) isValidVeoModel(modelID string) bool {
	for _, model := range supportedVeoModels {
		if model.ID == modelID {
			return true
		}
	}
	return false
}

// getDefaultVeoModel - デフォルトのVeoモデルIDを取得
func (h *VeoHandler) getDefaultVeoModel() string {
	return "veo-3.0-generate-preview" // 固定モデル
}

// サポートされるVeoモデル一覧（サーバー側で固定）
var supportedVeoModels = []VeoModel{
	{
		ID:          "veo-3.0-generate-preview",
		Name:        "Veo 3.0 Preview",
		Description: "最新動画生成モデル（プレビュー版）",
	},
	{
		ID:          "veo-3.0-fast-generate-preview",
		Name:        "Veo 3.0 Fast",
		Description: "最新動画生成モデル（高速版）",
	},
	{
		ID:          "veo-2.0-generate-001",
		Name:        "Veo 2.0",
		Description: "動画生成モデル（旧バージョン）",
	},
}

// isQuotaError - クォータエラーかどうかを判定
func (h *VeoHandler) isQuotaError(err error) bool {
	return fmt.Sprintf("%v", err) != "" &&
		(fmt.Sprintf("%v", err) == "service temporarily unavailable due to high demand")
}

// sendError - エラーレスポンスを送信
func (h *VeoHandler) sendError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// getDefaultImagenModelForVeo - Veo用のデフォルトImagenモデルIDを取得
func (h *VeoHandler) getDefaultImagenModelForVeo() string {
	return "imagen-3.0-generate-002" // 安定版を推奨
}

// HandleVeoIndex - Veo動画生成画面を表示
func (h *VeoHandler) HandleVeoIndex(w http.ResponseWriter, r *http.Request) {
	// 現在のVertex AIリージョン情報をツールチップに含める
	locationInfo := fmt.Sprintf(" 現在のVertex AIリージョン: %s", h.location)

	// モデル選択肢を動的に生成
	var modelOptions strings.Builder
	for i, model := range supportedVeoModels {
		selected := ""
		if model.ID == h.getDefaultVeoModel() {
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
	renderPage(w, "veo_index.html", pageData{LocationInfo: locationInfo, ModelOptions: modelOptions.String()})
}
