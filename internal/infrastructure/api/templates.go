package api

import (
	"bytes"
	"embed"
	"log"
	"net/http"
	"text/template"
)

// 静的ページ（埋め込み値なし）はバイト列として埋め込み、そのまま配信する。
//
//go:embed templates/tryon_index.html
var tryOnIndexHTML []byte

// 動的ページ（locationInfo / modelOptions を埋め込む）は text/template で描画する。
// 既存の出力をそのまま維持するため、自動エスケープのない text/template を使用する。
//
//go:embed templates/imagen_index.html templates/veo_index.html templates/nanobanana_index.html
var dynamicTemplateFS embed.FS

var pageTemplates = template.Must(template.ParseFS(dynamicTemplateFS, "templates/*.html"))

// pageData は動的ページのテンプレートに渡す値。
type pageData struct {
	// LocationInfo は現在のVertex AIリージョン情報。
	LocationInfo string
	// ModelOptions は <option> 要素群（Go側で生成済みのHTML文字列）。
	ModelOptions string
}

// renderStaticPage は埋め込み済みの静的HTMLをそのまま配信する。
func renderStaticPage(w http.ResponseWriter, body []byte) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, max-age=0")
	w.Write(body)
}

// renderPage は動的ページをバッファに描画してから配信する。
// テンプレート実行に失敗した場合はヘッダを書き込む前に500を返す。
func renderPage(w http.ResponseWriter, name string, data pageData) {
	var buf bytes.Buffer
	if err := pageTemplates.ExecuteTemplate(&buf, name, data); err != nil {
		log.Printf("template execute failed (%s): %v", name, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, max-age=0")
	w.Write(buf.Bytes())
}
