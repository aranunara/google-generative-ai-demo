package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"tryon-demo/internal/application/services"
	"tryon-demo/internal/application/usecases"
)

const maxFileSize = 25 * 1024 * 1024 // 25MB

type TryOnHandler struct {
	tryOnUseCase     *usecases.TryOnUseCase
	parameterService *services.ParameterService
}

func NewTryOnHandler(
	tryOnUseCase *usecases.TryOnUseCase,
	parameterService *services.ParameterService,
) *TryOnHandler {
	return &TryOnHandler{
		tryOnUseCase:     tryOnUseCase,
		parameterService: parameterService,
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
<span class="tooltiptext">生成画像をGoogle Cloud Storageに保存する場合のバケットパスです。空白の場合は直接レスポンスで返されます。形式例: gs://your-bucket/path/</span>
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
    
    // 前の結果をクリアしてローディングアニメーションを表示
    resultDisplay.innerHTML = '<div class="loader"></div>';
    resultDisplay.style.display = 'flex';
    multipleResults.innerHTML = '';
    multipleResults.style.display = 'none';
    errorMessage.classList.add('hidden');

    const formData = new FormData();
    formData.append('person_image', p);
    formData.append('garment_image', g);
    
    // フォームの全てのパラメータを追加
    const formElements = document.querySelectorAll('#advanced-settings input, #advanced-settings select');
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
            if (data.success && data.images && data.images.length > 0) {
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
                throw new Error('Invalid response format');
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

func (h *TryOnHandler) HandleTryOn(w http.ResponseWriter, r *http.Request) {
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

	var images []map[string]string
	for i, img := range output.Images {
		images = append(images, map[string]string{
			"id":   fmt.Sprintf("image_%d", i),
			"data": base64.StdEncoding.EncodeToString(img.Data),
			"type": img.Type,
		})
	}

	response := map[string]any{
		"success": true,
		"images":  images,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode JSON response: %v", err)
		h.sendError(w, "レスポンスの生成に失敗しました", http.StatusInternalServerError)
		return
	}
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
