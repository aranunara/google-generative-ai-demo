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
	"tryon-demo/internal/domain/valueobjects"
)

type NanobananaHandler struct {
	nanobananaUseCase *usecases.NanobananaUseCase
	location          string // Vertex AIのリージョン情報
}

func NewNanobananaHandler(nanobananaUseCase *usecases.NanobananaUseCase, location string) *NanobananaHandler {
	return &NanobananaHandler{
		nanobananaUseCase: nanobananaUseCase,
		location:          location,
	}
}

func (h *NanobananaHandler) HandleNanobananaIndex(w http.ResponseWriter, r *http.Request) {
	// 現在のVertex AIリージョン情報をツールチップに含める
	locationInfo := fmt.Sprintf(" 現在のVertex AIリージョン: %s", h.location)
	renderPage(w, "nanobanana_index.html", pageData{LocationInfo: locationInfo})
}

func (h *NanobananaHandler) getDefaultNanobananaModel() string {
	return "gemini-2.5-flash-image-preview"
}

func (h *NanobananaHandler) HandleNanobanana(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// フォームデータの解析
	err := r.ParseMultipartForm(32 << 20) // 32MB
	if err != nil {
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}

	prompt := r.FormValue("prompt")
	if prompt == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "プロンプトが必要です",
		})
		return
	}

	// 複数画像ファイルの取得
	form := r.MultipartForm
	if form == nil || form.File == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "画像ファイルが必要です",
		})
		return
	}

	imageFiles, exists := form.File["images"]
	if !exists || len(imageFiles) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "画像ファイルが必要です",
		})
		return
	}

	// 最大3枚まで制限
	if len(imageFiles) > 3 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "画像は最大3枚までアップロードできます",
		})
		return
	}

	// 画像データの読み込み
	var imageDatas []*valueobjects.ImageData
	for _, fileHeader := range imageFiles {
		file, err := fileHeader.Open()
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "画像ファイルの読み込みに失敗しました",
			})
			return
		}
		defer file.Close()

		imageData, err := io.ReadAll(file)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "画像ファイルの読み込みに失敗しました",
			})
			return
		}

		// MIMEタイプの検証
		contentType := http.DetectContentType(imageData)
		if !strings.HasPrefix(contentType, "image/") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "有効な画像ファイルを選択してください",
			})
			return
		}

		// UseCase実行用の入力データを準備
		imageDataObj, err := valueobjects.NewImageData(imageData, contentType)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   fmt.Sprintf("画像データの作成に失敗しました: %v", err),
			})
			return
		}

		imageDatas = append(imageDatas, imageDataObj)
	}

	input := usecases.NanobananaInput{
		Model:      h.getDefaultNanobananaModel(),
		Prompt:     prompt,
		ImageDatas: imageDatas,
	}

	// UseCase実行
	ctx := r.Context()
	output, err := h.nanobananaUseCase.ModifyImage(ctx, input)
	if err != nil {
		log.Printf("Error executing Nanobanana use case: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("画像編集に失敗しました: %v", err),
		})
		return
	}

	// レスポンスの構築
	response := map[string]interface{}{
		"success": true,
		// "response": output.Response,
	}

	// 画像データがある場合は追加
	if output.Image != nil {
		log.Printf("Successfully received image data, size: %d bytes", len(output.Image.Data()))
		imageBase64 := base64.StdEncoding.EncodeToString(output.Image.Data())
		response["image"] = map[string]string{
			"data": imageBase64,
			"type": output.Image.MimeType(),
		}
	} else {
		log.Printf("No image data in output, response text: %s", output.Response)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "画像データが返されませんでした",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
