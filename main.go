package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"cloud.google.com/go/vertexai/genai"
	"github.com/gorilla/mux"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"

	// WebP形式のサポートを追加
	_ "golang.org/x/image/webp"
)

const (
	maxFileSize = 25 * 1024 * 1024 // 25MB
)

type Server struct {
	projectID string
	location  string
	vtoModel  string
	client    *genai.Client
	useREST   bool // REST API使用フラグ
}

// REST API用の構造体定義
type ImageData struct {
	BytesBase64Encoded string `json:"bytesBase64Encoded,omitempty"`
	GcsUri             string `json:"gcsUri,omitempty"`
}

type PersonImage struct {
	Image ImageData `json:"image"`
}

type ProductImage struct {
	Image ImageData `json:"image"`
}

type Instance struct {
	PersonImage   PersonImage    `json:"personImage"`
	ProductImages []ProductImage `json:"productImages"`
}

type OutputOptions struct {
	MimeType           string `json:"mimeType,omitempty"`
	CompressionQuality int    `json:"compressionQuality,omitempty"`
}

type Parameters struct {
	AddWatermark     bool          `json:"addWatermark,omitempty"`
	BaseSteps        int           `json:"baseSteps,omitempty"`
	PersonGeneration string        `json:"personGeneration,omitempty"`
	SafetySetting    string        `json:"safetySetting,omitempty"`
	SampleCount      int           `json:"sampleCount,omitempty"`
	Seed             int           `json:"seed,omitempty"`
	StorageUri       string        `json:"storageUri,omitempty"`
	OutputOptions    OutputOptions `json:"outputOptions,omitempty"`
}

type PredictRequest struct {
	Instances  []Instance `json:"instances"`
	Parameters Parameters `json:"parameters,omitempty"`
}

type PredictResponse struct {
	Predictions []interface{} `json:"predictions"`
	Metadata    interface{}   `json:"metadata,omitempty"`
}

// NewServer はServerを初期化する
func NewServer() (*Server, error) {
	projectID := os.Getenv("PROJECT_ID")
	if projectID == "" {
		return nil, fmt.Errorf("環境変数 PROJECT_ID が未設定です")
	}

	location := os.Getenv("LOCATION")
	if location == "" {
		location = "us-central1"
	}

	// Python版と同じデフォルト値を使用
	vtoModel := os.Getenv("VTO_MODEL")
	if vtoModel == "" {
		vtoModel = "virtual-try-on-preview-08-04"
	}

	log.Printf("[boot] Using VTO_MODEL=%s", vtoModel)

	// REST API使用フラグを確認
	useREST := os.Getenv("USE_REST_API") == "true"
	log.Printf("[boot] USE_REST_API=%v", useREST)

	var client *genai.Client
	if !useREST {
		// 従来のgenai.Clientを使用する場合
		ctx := context.Background()
		// エンドポイント
		endpoint := fmt.Sprintf("%s-aiplatform.googleapis.com:443", location)
		slog.Info("endpoint", "endpoint", endpoint)
		var err error
		client, err = genai.NewClient(ctx, projectID, location, option.WithEndpoint(endpoint))
		if err != nil {
			return nil, fmt.Errorf("failed to create genai client: %w", err)
		}
	}

	return &Server{
		projectID: projectID,
		location:  location,
		vtoModel:  vtoModel,
		client:    client,
		useREST:   useREST,
	}, nil
}

// convertToJPEG は画像をJPEG形式に変換（Python版のensure_jpeg_bytesと同等）
func convertToJPEG(data []byte) ([]byte, error) {
	reader := bytes.NewReader(data)

	// まず画像形式を検出
	_, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image config (サポートされていない画像形式の可能性があります。JPEG、PNG、GIF、WebP形式をお試しください): %w", err)
	}

	log.Printf("Detected image format: %s", format)

	// すでにJPEGの場合はそのまま返す
	if format == "jpeg" {
		return data, nil
	}

	// 画像をデコード
	reader.Seek(0, 0)
	img, _, err := image.Decode(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image (画像形式: %s): %w", format, err)
	}

	// JPEGにエンコード
	var buf bytes.Buffer
	opts := &jpeg.Options{Quality: 90}
	if err := jpeg.Encode(&buf, img, opts); err != nil {
		return nil, fmt.Errorf("failed to encode to JPEG: %w", err)
	}

	log.Printf("Successfully converted %s to JPEG", format)
	return buf.Bytes(), nil
}

// getAccessToken はGoogleクラウド認証用のアクセストークンを取得する
// gcloud auth print-access-tokenの代替機能
func getAccessToken(ctx context.Context) (string, error) {
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

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	// Python版のindex.htmlと同じHTMLを返す
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
<label class="inline-flex items-center px-4 py-2 rounded-full bg-gradient-to-r from-indigo-500 to-blue-500 text-white shadow hover:shadow-lg cursor-pointer">
<svg class="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6H17a3 3 0 010 6h-1m-4 5V10m0 0l-2 2m2-2l2 2"/></svg>
<span>ファイルを選択</span>
<input type="file" id="person-image" name="person_image" accept="image/*" class="hidden" required>
</label>
<span id="person-name" class="ml-2 text-sm text-gray-500"></span>
</div>
<div>
<label class="block text-lg font-semibold mb-2 text-gray-700">2. 衣服画像</label>
<div id="garment-preview" class="preview-box rounded-lg mb-3"><span class="text-gray-500">プレビュー</span></div>
<label class="inline-flex items-center px-4 py-2 rounded-full bg-gradient-to-r from-indigo-500 to-blue-500 text-white shadow hover:shadow-lg cursor-pointer">
<svg class="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6H17a3 3 0 010 6h-1m-4 5V10m0 0l-2 2m2-2l2 2"/></svg>
<span>ファイルを選択</span>
<input type="file" id="garment-image" name="garment_image" accept="image/*" class="hidden" required>
</label>
<span id="garment-name" class="ml-2 text-sm text-gray-500"></span>
</div>
</div>
<div class="text-center space-y-3">
<button type="submit" id="submit-btn"
class="bg-gradient-to-r from-indigo-500 to-blue-600 text-white font-bold py-3 px-8 rounded-full hover:shadow-xl transform hover:-translate-y-0.5 transition-all">
着せ替えを実行
</button>
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
</div>
<div id="error-message" class="mt-6 hidden bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded-lg"></div>
</main>
<footer class="text-center mt-8 text-gray-500 text-sm">
<p>Powered by Google Cloud Vertex AI & Cloud Run</p>
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
const errorMessage = document.getElementById('error-message');
const submitBtn = document.getElementById('submit-btn');
const clearBtn = document.getElementById('clear-btn');

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
    resultSection.classList.add('hidden');
    errorMessage.classList.add('hidden');
});

form.addEventListener('submit', async (event) => {
    event.preventDefault();
    const p = personInput.files[0];
    const g = garmentInput.files[0];
    if (!p || !g) return;

    const MAX = 25 * 1024 * 1024;
    if (p.size > MAX || g.size > MAX) {
        errorMessage.textContent = '画像が大きすぎます（25MBまで対応）';
        errorMessage.classList.remove('hidden');
        return;
    }

    submitBtn.disabled = true;
    submitBtn.textContent = '生成中...';
    resultSection.classList.remove('hidden');
    resultDisplay.innerHTML = '<div class="loader"></div>';
    errorMessage.classList.add('hidden');

    const formData = new FormData();
    formData.append('person_image', p);
    formData.append('garment_image', g);

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
        const blob = await resp.blob();
        resultDisplay.innerHTML = '<img src="' + URL.createObjectURL(blob) + '" alt="Result">';
    } catch (err) {
        console.error(err);
        resultDisplay.innerHTML = '<span class="text-red-500">生成に失敗しました</span>';
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

func (s *Server) handleTryOn(w http.ResponseWriter, r *http.Request) {
	// ファイルサイズ制限
	r.Body = http.MaxBytesReader(w, r.Body, maxFileSize)
	if err := r.ParseMultipartForm(maxFileSize); err != nil {
		sendError(w, "画像が大きすぎます（25MBまで対応）", http.StatusRequestEntityTooLarge)
		return
	}

	// 画像ファイルを取得
	personFile, _, errFormFilePerson := r.FormFile("person_image")
	if errFormFilePerson != nil {
		sendError(w, "人物画像を選んでください", http.StatusBadRequest)
		return
	}
	defer personFile.Close()

	garmentFile, _, errFormFileGarment := r.FormFile("garment_image")
	if errFormFileGarment != nil {
		sendError(w, "衣服画像を選んでください", http.StatusBadRequest)
		return
	}
	defer garmentFile.Close()

	// 画像データを読み込み
	personData, errReadAllPerson := io.ReadAll(personFile)
	if errReadAllPerson != nil {
		sendError(w, "人物画像の読み込みに失敗しました", http.StatusInternalServerError)
		return
	}

	garmentData, errReadAllGarment := io.ReadAll(garmentFile)
	if errReadAllGarment != nil {
		sendError(w, "衣服画像の読み込みに失敗しました", http.StatusInternalServerError)
		return
	}

	// JPEG形式に変換（Python版と同じ処理）
	personJPEG, errConvertToJPEGPerson := convertToJPEG(personData)
	if errConvertToJPEGPerson != nil {
		log.Printf("Failed to convert person image: %v", errConvertToJPEGPerson)
		sendError(w, "人物画像の処理に失敗しました", http.StatusInternalServerError)
		return
	}

	garmentJPEG, errConvertToJPEGGarment := convertToJPEG(garmentData)
	if errConvertToJPEGGarment != nil {
		log.Printf("Failed to convert garment image: %v", errConvertToJPEGGarment)
		sendError(w, "衣服画像の処理に失敗しました", http.StatusInternalServerError)
		return
	}

	// Virtual Try-On API を呼び出し
	ctx := context.Background()
	var result []byte
	var errCallVirtualTryOn error

	if s.useREST {
		log.Printf("Using REST API for Virtual Try-On")
		result, errCallVirtualTryOn = s.callVirtualTryOnREST(ctx, personJPEG, garmentJPEG)
	} else {
		log.Printf("Using genai.Client for Virtual Try-On")
		result, errCallVirtualTryOn = s.callVirtualTryOn(ctx, personJPEG, garmentJPEG)
	}

	if errCallVirtualTryOn != nil {
		log.Printf("Virtual Try-On failed: %v", errCallVirtualTryOn)

		// クォータエラーの場合は特別なメッセージ
		if isQuotaError(errCallVirtualTryOn) {
			sendError(w, "現在サーバーが混雑しています。しばらく待ってから再試行してください。", http.StatusTooManyRequests)
			return
		}

		hint := "ヒント: 露出や著名人・ロゴ類・過度な加工を避け、人物と衣服がはっきり写る画像で再試行してください。"
		sendError(w, fmt.Sprintf("生成に失敗しました: %v %s", errCallVirtualTryOn, hint), http.StatusInternalServerError)
		return
	}

	// 結果を返す
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "no-store, max-age=0")
	w.Write(result)
}

func (s *Server) callVirtualTryOn(ctx context.Context, personImage, garmentImage []byte) ([]byte, error) {
	// GenerativeModelを作成
	model := s.client.GenerativeModel(s.vtoModel)

	// Python版のRecontextImageSourceと同等の構造を作成
	personPart := genai.ImageData("image/jpeg", personImage)
	garmentPart := genai.ImageData("image/jpeg", garmentImage)

	// プロンプトを構築（Virtual Try-On用の特殊な形式）
	prompt := []genai.Part{
		genai.Text("person:"),
		personPart,
		genai.Text("garment:"),
		garmentPart,
	}

	// 生成設定
	model.SetTemperature(0.4)
	model.SetTopK(32)
	model.SetTopP(1)
	model.SetMaxOutputTokens(2048)
	model.ResponseMIMEType = "image/jpeg"

	// 生成を実行
	resp, err := model.GenerateContent(ctx, prompt...)
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	// レスポンスから画像を取得
	if len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates in response")
	}

	candidate := resp.Candidates[0]
	if candidate.Content == nil || len(candidate.Content.Parts) == 0 {
		return nil, fmt.Errorf("no content in response")
	}

	// 画像データを抽出
	for _, part := range candidate.Content.Parts {
		if blob, ok := part.(genai.Blob); ok {
			if blob.MIMEType == "image/jpeg" || blob.MIMEType == "image/png" {
				return blob.Data, nil
			}
		}
	}

	return nil, fmt.Errorf("no image found in response")
}

// callVirtualTryOnREST はREST APIを直接呼び出してVirtual Try-Onを実行する
// curlコマンドの模倣実装
func (s *Server) callVirtualTryOnREST(ctx context.Context, personImage, garmentImage []byte) ([]byte, error) {
	// アクセストークンを取得
	accessToken, err := getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	// 画像をBase64エンコード
	personB64 := base64.StdEncoding.EncodeToString(personImage)
	garmentB64 := base64.StdEncoding.EncodeToString(garmentImage)

	// リクエストペイロードを構築
	request := PredictRequest{
		Instances: []Instance{
			{
				PersonImage: PersonImage{
					Image: ImageData{
						BytesBase64Encoded: personB64,
					},
				},
				ProductImages: []ProductImage{
					{
						Image: ImageData{
							BytesBase64Encoded: garmentB64,
						},
					},
				},
			},
		},
		Parameters: Parameters{
			AddWatermark:     false,
			BaseSteps:        4,
			PersonGeneration: "base-person-from-provided-image",
			SafetySetting:    "block_some",
			SampleCount:      1,
			Seed:             0,
			OutputOptions: OutputOptions{
				MimeType:           "image/jpeg",
				CompressionQuality: 90,
			},
		},
	}

	// JSONにシリアライズ
	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// API エンドポイントURL
	url := fmt.Sprintf("https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/google/models/%s:predict",
		s.location, s.projectID, s.location, s.vtoModel)

	// HTTPリクエストを作成
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// ヘッダーを設定
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	// HTTPクライアントでリクエストを送信
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// レスポンスを読み取り
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	// レスポンスをパース
	var predResp PredictResponse
	if err := json.Unmarshal(respBody, &predResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// 画像データを抽出
	if len(predResp.Predictions) == 0 {
		return nil, fmt.Errorf("no predictions in response")
	}

	// 予測結果から画像データを取得
	// レスポンス構造は実際のAPIレスポンスに依存するため、適切に調整が必要
	prediction, ok := predResp.Predictions[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid prediction format")
	}

	// 画像データが格納されているフィールドを探す
	// 実際のAPIレスポンス構造に合わせて調整が必要
	var imageB64 string
	if bytesB64, exists := prediction["bytesBase64Encoded"]; exists {
		if s, ok := bytesB64.(string); ok {
			imageB64 = s
		}
	} else if images, exists := prediction["images"]; exists {
		if imageArray, ok := images.([]interface{}); ok && len(imageArray) > 0 {
			if imageObj, ok := imageArray[0].(map[string]interface{}); ok {
				if bytesB64, exists := imageObj["bytesBase64Encoded"]; exists {
					if s, ok := bytesB64.(string); ok {
						imageB64 = s
					}
				}
			}
		}
	}

	if imageB64 == "" {
		return nil, fmt.Errorf("no image data found in response")
	}

	// Base64デコード
	imageData, err := base64.StdEncoding.DecodeString(imageB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 image: %w", err)
	}

	return imageData, nil
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// isQuotaError checks if the error is a quota exceeded error
func isQuotaError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "quota exceeded") ||
		strings.Contains(errStr, "resourceexhausted")
}

func sendError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// main
func main() {
	server, err := NewServer()
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// genai.Clientが存在する場合のみクローズ
	if server.client != nil {
		defer server.client.Close()
	}

	r := mux.NewRouter()
	r.HandleFunc("/", server.handleIndex).Methods("GET")
	r.HandleFunc("/tryon", server.handleTryOn).Methods("POST")
	r.HandleFunc("/healthz", server.handleHealth).Methods("GET")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting server on port %s", port)
	log.Printf("Project: %s, Location: %s, Model: %s", server.projectID, server.location, server.vtoModel)
	log.Printf("API Mode: %s", func() string {
		if server.useREST {
			return "REST API"
		}
		return "genai.Client"
	}())

	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
