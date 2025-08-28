package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

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

	veoModel := h.getDefaultVeoModel() // サーバー側で固定
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

	response := h.createVeoResponse(output.Video)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode JSON response: %v", err)
		h.sendError(w, "レスポンスの生成に失敗しました", http.StatusInternalServerError)
		return
	}
}

// createVeoResponse - Veo用のレスポンスを生成
func (h *VeoHandler) createVeoResponse(videoData []byte) map[string]any {
	log.Printf("[DEBUG] createVeoResponse called with video size: %d bytes", len(videoData))

	if len(videoData) == 0 {
		log.Printf("[WARNING] Empty video data")
		return map[string]any{
			"success": false,
			"error":   "動画データが空です",
		}
	}

	base64Data := base64.StdEncoding.EncodeToString(videoData)
	log.Printf("[DEBUG] Base64 encoded video length: %d characters", len(base64Data))

	response := map[string]any{
		"success": true,
		"video": map[string]string{
			"data": base64Data,
			"type": "video/mp4",
		},
		"model": h.getDefaultVeoModel(),
	}

	return response
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

	html := `<!DOCTYPE html>
<html lang="ja">
<head>
<meta charset="UTF-8"/>
<meta name="viewport" content="width=device-width, initial-scale=1.0"/>
<title>Vertex AI Veo - 動画生成</title>
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
.result-preview{width:100%;height:400px;background:#f3f4f6;border:2px dashed #d1d5db;display:flex;align-items:center;justify-content:center;overflow:hidden;border-radius:8px}
.result-preview video{max-width:100%;max-height:100%;object-fit:contain}
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
<button onclick="location.href='/veo'" class="px-4 py-2 bg-purple-700 text-white rounded-lg shadow-md font-medium ring-2 ring-purple-300">
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
<h1 class="text-3xl md:text-4xl font-bold text-gray-900">Vertex AI Veo</h1>
<p class="text-gray-600 mt-2">画像から動画を生成します</p>
<p class="text-sm text-indigo-600 mt-1">使用モデル: veo-3.0-generate-preview` + locationInfo + `</p>
</header>
<main class="bg-white p-6 md:p-8 rounded-2xl shadow-lg">
<form id="veo-form">
<div class="space-y-6 mb-6">
<div>
<label class="block text-lg font-semibold mb-4 text-gray-700">
入力画像の設定方法
<div class="tooltip">
<span class="info-icon">?</span>
<span class="tooltiptext">動画生成のベースとなる画像を設定する方法を選択してください。画像生成を使用する場合はImagen AIが画像を生成してから動画化します。</span>
</div>
</label>
<div class="mb-4">
<label class="inline-flex items-center">
<input type="checkbox" id="useImageGeneration" class="rounded border-gray-300 text-indigo-600 shadow-sm focus:border-indigo-300 focus:ring focus:ring-indigo-200 focus:ring-opacity-50">
<span class="ml-2 text-md font-medium text-gray-700">画像生成を使用する（プロンプトから画像を生成）</span>
</label>
</div>

<!-- 画像アップロード用UI（デフォルト表示） -->
<div id="image-upload-section">
<label class="block text-md font-medium mb-2 text-gray-600">
画像ファイルをアップロード
<div class="tooltip">
<span class="info-icon">?</span>
<span class="tooltiptext">動画生成のベースとなる画像ファイルをアップロードしてください。</span>
</div>
</label>
<div id="image-upload-area" class="image-upload-area p-6 text-center" onclick="document.getElementById('image-input').click()">
<div id="upload-prompt" class="upload-prompt">
<svg class="w-12 h-12 text-gray-400 mx-auto mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6H17a3 3 0 010 6h-1m-4 5V10m0 0l-2 2m2-2l2 2"/>
</svg>
<p class="text-lg font-medium text-gray-700 mb-2">画像をドラッグ&ドロップ</p>
<p class="text-sm text-gray-500 mb-4">または、クリックしてファイルを選択</p>
<span class="inline-flex items-center px-4 py-2 rounded-full bg-gradient-to-r from-indigo-500 to-blue-500 text-white shadow hover:shadow-lg text-sm font-medium">
<svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6H17a3 3 0 010 6h-1m-4 5V10m0 0l-2 2m2-2l2 2"/>
</svg>
ファイルを選択
</span>
</div>
<div id="image-preview" class="hidden">
<img id="preview-img" src="" alt="Preview" class="max-w-full max-h-48 object-contain mx-auto rounded-lg shadow-md">
<p id="image-name" class="mt-2 text-sm text-gray-600 font-medium"></p>
<button type="button" id="remove-image" class="mt-2 px-3 py-1 bg-red-500 text-white text-sm rounded-lg hover:bg-red-600 transition-colors">
削除
</button>
</div>
</div>
<input type="file" id="image-input" name="image" accept="image/*" class="hidden">
</div>

<!-- 画像生成用UI（非表示がデフォルト） -->
<div id="image-generation-section" style="display: none;">
<label class="block text-md font-medium mb-2 text-gray-600">
画像生成プロンプト
<div class="tooltip">
<span class="info-icon">?</span>
<span class="tooltiptext">生成したい画像の詳細な説明を入力してください。Imagen AIが画像を生成してから動画化します。</span>
</div>
</label>
<textarea id="imagenPrompt" name="imagenPrompt" rows="3" 
class="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500" 
placeholder="例: A beautiful landscape with mountains and a lake during sunset"></textarea>
</div>
</div>
<div>
<label class="block text-lg font-semibold mb-2 text-gray-700">
動画プロンプト（必須）
<div class="tooltip">
<span class="info-icon">?</span>
<span class="tooltiptext">生成したい動画の詳細な説明を入力してください。動きの種類、シーン、雰囲気などを具体的に記述してください。</span>
</div>
</label>
<textarea id="videoPrompt" name="videoPrompt" required rows="4" 
class="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500" 
placeholder="例: A person walking through a beautiful garden with flowers blooming in slow motion"></textarea>
</div>
</div>
<!-- 実行ボタン（メイン） -->
<div class="text-center mb-8">
<button type="submit" id="submit-btn"
class="bg-gradient-to-r from-indigo-500 to-blue-600 text-white font-bold py-4 px-12 rounded-full hover:shadow-xl transform hover:-translate-y-0.5 transition-all text-lg">
動画を生成
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
<div id="result-display" class="result-preview"></div>
</div>
<div id="error-message" class="mt-6 hidden bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded-lg"></div>
</main>

</div>
<script>
const form = document.getElementById('veo-form');
const videoPromptInput = document.getElementById('videoPrompt');
const imagenPromptInput = document.getElementById('imagenPrompt');
const imageInput = document.getElementById('image-input');
const imagePreview = document.getElementById('image-preview');
const imageName = document.getElementById('image-name');
const resultSection = document.getElementById('result-section');
const resultDisplay = document.getElementById('result-display');
const errorMessage = document.getElementById('error-message');
const submitBtn = document.getElementById('submit-btn');
const clearBtn = document.getElementById('clear-btn');
const useImageGenerationCheckbox = document.getElementById('useImageGeneration');
const imageUploadSection = document.getElementById('image-upload-section');
const imageGenerationSection = document.getElementById('image-generation-section');

// 画像プレビュー設定
function setupImagePreview() {
    const imageUploadArea = document.getElementById('image-upload-area');
    const uploadPrompt = document.getElementById('upload-prompt');
    const previewImg = document.getElementById('preview-img');
    const removeBtn = document.getElementById('remove-image');

    // ドラッグ&ドロップ
    imageUploadArea.addEventListener('dragover', (e) => {
        e.preventDefault();
        imageUploadArea.classList.add('drag-over');
    });

    imageUploadArea.addEventListener('dragleave', (e) => {
        e.preventDefault();
        if (!imageUploadArea.contains(e.relatedTarget)) {
            imageUploadArea.classList.remove('drag-over');
        }
    });

    imageUploadArea.addEventListener('drop', (e) => {
        e.preventDefault();
        imageUploadArea.classList.remove('drag-over');
        const files = e.dataTransfer.files;
        if (files.length > 0) {
            const file = files[0];
            if (file.type.startsWith('image/')) {
                imageInput.files = files;
                showImagePreview(file);
            }
        }
    });

    // ファイル選択
    imageInput.addEventListener('change', (e) => {
        const file = e.target.files[0];
        if (file) {
            showImagePreview(file);
        }
    });

    // 削除ボタン
    removeBtn.addEventListener('click', (e) => {
        e.stopPropagation();
        clearImagePreview();
    });

    function showImagePreview(file) {
        const reader = new FileReader();
        reader.onload = (e) => {
            previewImg.src = e.target.result;
            imageName.textContent = file.name;
            uploadPrompt.classList.add('hidden');
            imagePreview.classList.remove('hidden');
        };
        reader.readAsDataURL(file);
    }

    function clearImagePreview() {
        imageInput.value = '';
        previewImg.src = '';
        imageName.textContent = '';
        uploadPrompt.classList.remove('hidden');
        imagePreview.classList.add('hidden');
    }
}

setupImagePreview();

// チェックボックスの状態変更を監視してUIを切り替え
function toggleImageInputMethod() {
    const useGeneration = useImageGenerationCheckbox.checked;
    const uploadPrompt = document.getElementById('upload-prompt');
    const previewImg = document.getElementById('preview-img');
    
    if (useGeneration) {
        // 画像生成を使用する場合
        imageUploadSection.style.display = 'none';
        imageGenerationSection.style.display = 'block';
        // 画像アップロード関連をクリア
        imageInput.value = '';
        imageName.textContent = '';
        previewImg.src = '';
        uploadPrompt.classList.remove('hidden');
        imagePreview.classList.add('hidden');
    } else {
        // 画像アップロードを使用する場合
        imageUploadSection.style.display = 'block';
        imageGenerationSection.style.display = 'none';
        // 画像生成プロンプトをクリア
        imagenPromptInput.value = '';
    }
}

// チェックボックスの変更イベントを監視
useImageGenerationCheckbox.addEventListener('change', toggleImageInputMethod);

// 初期状態を設定
toggleImageInputMethod();

// クリアボタン
clearBtn.addEventListener('click', () => {
    if (confirm('全ての入力内容をクリアしますか？')) {
        const uploadPrompt = document.getElementById('upload-prompt');
        const previewImg = document.getElementById('preview-img');
        
        videoPromptInput.value = '';
        imagenPromptInput.value = '';
        imageInput.value = '';
        imageName.textContent = '';
        previewImg.src = '';
        uploadPrompt.classList.remove('hidden');
        imagePreview.classList.add('hidden');
        resultDisplay.innerHTML = '';
        resultSection.classList.add('hidden');
        errorMessage.classList.add('hidden');
        
        // チェックボックスもリセット
        useImageGenerationCheckbox.checked = false;
        toggleImageInputMethod();
    }
});

// フォーム送信
form.addEventListener('submit', async (event) => {
    event.preventDefault();
    
    const videoPrompt = videoPromptInput.value.trim();
    const useGeneration = useImageGenerationCheckbox.checked;
    const imagenPrompt = imagenPromptInput.value.trim();
    const imageFile = imageInput.files[0];
    
    if (!videoPrompt) {
        errorMessage.textContent = '動画プロンプトを入力してください';
        errorMessage.classList.remove('hidden');
        return;
    }
    
    if (useGeneration) {
        // 画像生成を使用する場合
        if (!imagenPrompt) {
            errorMessage.textContent = '画像生成プロンプトを入力してください';
            errorMessage.classList.remove('hidden');
            return;
        }
    } else {
        // 画像アップロードを使用する場合
        if (!imageFile) {
            errorMessage.textContent = '画像ファイルを選択してください';
            errorMessage.classList.remove('hidden');
            return;
        }
    }

    const MAX = 10 * 1024 * 1024;
    if (imageFile && imageFile.size > MAX) {
        errorMessage.textContent = '画像が大きすぎます（10MBまで対応）';
        errorMessage.classList.remove('hidden');
        return;
    }

    submitBtn.disabled = true;
    submitBtn.textContent = '生成中...';
    resultSection.classList.remove('hidden');
    
    // 前の結果をクリアしてローディングアニメーションを表示
    resultDisplay.innerHTML = '<div class="loader"></div>';
    errorMessage.classList.add('hidden');

    const formData = new FormData();
    formData.append('videoPrompt', videoPrompt);
    
    if (useGeneration) {
        // 画像生成を使用する場合
        formData.append('imagenPrompt', imagenPrompt);
    } else {
        // 画像アップロードを使用する場合
        formData.append('image', imageFile);
    }

    try {
        const resp = await fetch('/veo', { method: 'POST', body: formData });
        if (!resp.ok) {
            let msg = 'HTTP ' + resp.status;
            try {
                const j = await resp.json();
                if (j && j.error) msg = j.error;
            } catch {}
            throw new Error(msg);
        }
        
        const data = await resp.json();
        if (data.success && data.video) {
            // 動画表示
            const videoContainer = document.createElement('div');
            videoContainer.className = 'relative w-full h-full flex items-center justify-center';
            
            const videoElement = document.createElement('video');
            videoElement.src = 'data:' + data.video.type + ';base64,' + data.video.data;
            videoElement.alt = 'Generated Video';
            videoElement.className = 'max-w-full max-h-full object-contain rounded-lg shadow-md';
            videoElement.controls = true;
            videoElement.autoplay = false;
            
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
                        link.href = videoElement.src;
                        const timestamp = new Date().toISOString().replace(/[:.]/g, '-').slice(0, 19);
                        link.download = 'veo-result-' + timestamp + '.mp4';
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
            
            videoContainer.appendChild(videoElement);
            videoContainer.appendChild(saveBtn);
            resultDisplay.innerHTML = '';
            resultDisplay.appendChild(videoContainer);
        } else {
            throw new Error('動画の生成に失敗しました');
        }
    } catch (err) {
        console.error(err);
        resultDisplay.innerHTML = '<span class="text-red-500 flex items-center justify-center">生成に失敗しました</span>';
        errorMessage.textContent = 'エラー: ' + err.message;
        errorMessage.classList.remove('hidden');
    } finally {
        submitBtn.disabled = false;
        submitBtn.textContent = '動画を生成';
    }
});
</script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, max-age=0")
	w.Write([]byte(html))
}
