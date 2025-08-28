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

	html := `<!DOCTYPE html>
<html lang="ja">
<head>
<meta charset="UTF-8"/>
<meta name="viewport" content="width=device-width, initial-scale=1.0"/>
<title>Nanobanana - 画像編集</title>
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
.image-upload-area {
    border: 2px dashed #d1d5db;
    background: #f9fafb;
    border-radius: 8px;
    transition: all 0.3s ease;
    cursor: pointer;
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
<div class="container mx-auto p-4 md:p-8 max-w-4xl">
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
<button onclick="location.href='/nanobanana/image-editing'" class="px-4 py-2 bg-orange-700 text-white rounded-lg shadow-md font-medium ring-2 ring-orange-300">
<svg class="w-4 h-4 inline mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/></svg>
Nanobanana画像編集
</button>
</div>
</nav>

<header class="text-center mb-8">
<h1 class="text-3xl md:text-4xl font-bold text-gray-900">Nanobanana 画像編集</h1>
<p class="text-gray-600 mt-2">画像とプロンプトを使用して画像を編集します</p>
</header>

<main class="bg-white p-6 md:p-8 rounded-2xl shadow-lg">
<form id="nanobanana-form" enctype="multipart/form-data">
<div class="space-y-6 mb-6">
<!-- 画像アップロード -->
<div>
<label class="block text-lg font-semibold mb-2 text-gray-700">
編集する画像をアップロード
<div class="tooltip">
<span class="info-icon">?</span>
<span class="tooltiptext">編集したい画像をアップロードしてください。対応形式: JPG, PNG` + locationInfo + `</span>
</div>
</label>
<div class="image-upload-area p-8 text-center" id="image-upload-area">
<input type="file" id="image-input" name="image" accept="image/*" class="hidden" required>
<div id="upload-content">
<svg class="mx-auto h-12 w-12 text-gray-400 mb-4" stroke="currentColor" fill="none" viewBox="0 0 48 48">
<path d="M28 8H12a4 4 0 00-4 4v20m32-12v8m0 0v8a4 4 0 01-4 4H12a4 4 0 01-4-4v-4m32-4l-3.172-3.172a4 4 0 00-5.656 0L28 28M8 32l9.172-9.172a4 4 0 015.656 0L28 28m0 0l4 4m4-24h8m-4-4v8m-12 4h.02" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
</svg>
<p class="text-lg text-gray-600 mb-2">画像をドラッグ&ドロップするか、クリックして選択</p>
<p class="text-sm text-gray-500">JPG, PNG形式をサポート</p>
</div>
<div id="image-preview" class="hidden">
<img id="preview-image" class="max-w-full max-h-64 mx-auto rounded-lg">
<p class="text-sm text-gray-600 mt-2">別の画像に変更するにはクリック</p>
</div>
</div>
</div>

<!-- プロンプト -->
<div>
<label class="block text-lg font-semibold mb-2 text-gray-700">
編集プロンプト
<div class="tooltip">
<span class="info-icon">?</span>
<span class="tooltiptext">画像にどのような編集を加えたいかを具体的に記述してください。例：「背景を青空に変更」「犬を猫に変更」など</span>
</div>
</label>
<textarea id="prompt" name="prompt" required rows="4" 
class="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-orange-500 focus:border-orange-500" 
placeholder="例: 背景を美しい夕日の海に変更してください"></textarea>
</div>
</div>

<!-- 実行ボタン（メイン） -->
<div class="text-center mb-8">
<button type="submit" id="submit-btn"
class="bg-gradient-to-r from-orange-500 to-red-600 text-white font-bold py-4 px-12 rounded-full hover:shadow-xl transform hover:-translate-y-0.5 transition-all text-lg">
画像を編集
</button>
</div>

<!-- クリアボタン（サブ） -->
<div class="text-center">
<button type="button" id="clear-btn"
class="px-6 py-2 text-sm rounded-lg border border-gray-300 text-gray-600 hover:bg-gray-50 hover:border-gray-400 transition-all">
<svg class="w-4 h-4 inline mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/></svg>
フォームクリア
</button>
</div>
</form>

<div id="result-section" class="mt-10 hidden">
<h2 class="text-2xl font-bold text-center mb-4 text-gray-800">編集結果</h2>
<div id="result-display" class="result-preview"></div>
<div id="response-text" class="mt-4 p-4 bg-gray-50 rounded-lg hidden">
<h3 class="font-semibold text-gray-700 mb-2">レスポンス:</h3>
<p id="response-content" class="text-gray-600"></p>
</div>
</div>

<div id="error-message" class="mt-6 hidden bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded-lg"></div>
</main>

</div>
<script>
const form = document.getElementById('nanobanana-form');
const imageInput = document.getElementById('image-input');
const imageUploadArea = document.getElementById('image-upload-area');
const uploadContent = document.getElementById('upload-content');
const imagePreview = document.getElementById('image-preview');
const previewImage = document.getElementById('preview-image');
const promptInput = document.getElementById('prompt');
const resultSection = document.getElementById('result-section');
const resultDisplay = document.getElementById('result-display');
const responseText = document.getElementById('response-text');
const responseContent = document.getElementById('response-content');
const errorMessage = document.getElementById('error-message');
const submitBtn = document.getElementById('submit-btn');
const clearBtn = document.getElementById('clear-btn');

// ファイルアップロード関連の処理
imageUploadArea.addEventListener('click', () => {
    imageInput.click();
});

imageUploadArea.addEventListener('dragover', (e) => {
    e.preventDefault();
    imageUploadArea.classList.add('drag-over');
});

imageUploadArea.addEventListener('dragleave', (e) => {
    e.preventDefault();
    imageUploadArea.classList.remove('drag-over');
});

imageUploadArea.addEventListener('drop', (e) => {
    e.preventDefault();
    imageUploadArea.classList.remove('drag-over');
    
    const files = e.dataTransfer.files;
    if (files.length > 0) {
        handleFileSelect(files[0]);
    }
});

imageInput.addEventListener('change', (e) => {
    if (e.target.files.length > 0) {
        handleFileSelect(e.target.files[0]);
    }
});

function handleFileSelect(file) {
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
        imagePreview.classList.remove('hidden');
    };
    reader.readAsDataURL(file);
}

// クリアボタン
clearBtn.addEventListener('click', () => {
    if (confirm('フォームをクリアしますか？')) {
        imageInput.value = '';
        promptInput.value = '';
        uploadContent.classList.remove('hidden');
        imagePreview.classList.add('hidden');
        resultDisplay.innerHTML = '';
        resultSection.classList.add('hidden');
        responseText.classList.add('hidden');
        errorMessage.classList.add('hidden');
    }
});

form.addEventListener('submit', async (event) => {
    event.preventDefault();
    
    const prompt = promptInput.value.trim();
    const imageFile = imageInput.files[0];
    
    if (!prompt) {
        errorMessage.textContent = 'プロンプトを入力してください';
        errorMessage.classList.remove('hidden');
        return;
    }
    
    if (!imageFile) {
        errorMessage.textContent = '画像を選択してください';
        errorMessage.classList.remove('hidden');
        return;
    }

    submitBtn.disabled = true;
    submitBtn.textContent = '編集中...';
    resultSection.classList.remove('hidden');
    
    // 前の結果をクリアしてローディングアニメーションを表示
    resultDisplay.innerHTML = '<div class="loader"></div>';
    resultDisplay.style.display = 'flex';
    responseText.classList.add('hidden');
    errorMessage.classList.add('hidden');

    const formData = new FormData();
    formData.append('prompt', prompt);
    formData.append('image', imageFile);

    try {
        const resp = await fetch('/nanobanana/image-editing', { method: 'POST', body: formData });
        if (!resp.ok) {
            let msg = 'HTTP ' + resp.status;
            try {
                const j = await resp.json();
                if (j && j.error) msg = j.error;
            } catch {}
            throw new Error(msg);
        }
        
        const data = await resp.json();
        if (data.success && data.image) {
            // 結果の画像を表示
            const imgContainer = document.createElement('div');
            imgContainer.className = 'relative w-full h-full flex items-center justify-center';
            
            const imgElement = document.createElement('img');
            imgElement.src = 'data:' + data.image.type + ';base64,' + data.image.data;
            imgElement.alt = 'Edited Image';
            imgElement.className = 'max-w-full max-h-full object-contain rounded-lg shadow-md';
            
            const saveBtn = document.createElement('button');
            saveBtn.textContent = '保存';
            saveBtn.className = 'absolute top-2 right-2 bg-orange-500 hover:bg-orange-600 text-white px-3 py-1 rounded text-sm transition-colors';
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
                        const extension = (data.image.type === 'image/jpeg') ? 'jpg' : 'png';
                        link.download = 'nanobanana-edited-' + timestamp + '.' + extension;
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
            
            // レスポンステキストを表示
            if (data.response) {
                responseContent.textContent = data.response;
                responseText.classList.remove('hidden');
            }
        } else {
            throw new Error('画像の編集に失敗しました');
        }
    } catch (err) {
        console.error(err);
        resultDisplay.innerHTML = '<span class="text-red-500 flex items-center justify-center">編集に失敗しました</span>';
        resultDisplay.style.display = 'flex';
        responseText.classList.add('hidden');
        errorMessage.textContent = 'エラー: ' + err.message;
        errorMessage.classList.remove('hidden');
    } finally {
        submitBtn.disabled = false;
        submitBtn.textContent = '画像を編集';
    }
});
</script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, max-age=0")
	w.Write([]byte(html))
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

	// 画像ファイルの取得
	file, _, err := r.FormFile("image")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "画像ファイルが必要です",
		})
		return
	}
	defer file.Close()

	// 画像データの読み込み
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

	input := usecases.NanobananaInput{
		Model:     h.getDefaultNanobananaModel(),
		Prompt:    prompt,
		ImageData: imageDataObj,
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
		imageBase64 := base64.StdEncoding.EncodeToString(output.Image.Data())
		response["image"] = map[string]string{
			"data": imageBase64,
			"type": output.Image.MimeType(),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
