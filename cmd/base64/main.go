package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/jpeg"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func main() {
	// このファイルがある同じ階層の「images」ディレクトリの中身を取得する
	files, err := os.ReadDir("images")
	if err != nil {
		log.Fatal(err)
	}

	validExtensions := []string{".jpg", ".png", ".jpeg"}

	for _, file := range files {
		// 拡張子が.jpg, .png, .jpegのファイルを取得
		if slices.Contains(validExtensions, strings.ToLower(filepath.Ext(file.Name()))) {
			encoded := encode(filepath.Join("images", file.Name()))
			// ファイル名の拡張子を除いたものをファイル名として保存
			save(strings.TrimSuffix(file.Name(), filepath.Ext(file.Name())), encoded)
		}
	}

}

func encode(file string) *string {
	// 画像ファイルを開く
	f, err := os.Open(file)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	// 画像デコード
	img, _, err := image.Decode(f)
	if err != nil {
		panic(err)
	}

	// 画像をPNGにエンコード
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}

	// Base64エンコード
	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())

	return &encoded
}

func save(file string, encoded *string) {
	// 同じ階層の「encoded」ディレクトリに.txt形式でファイルを保存する
	os.WriteFile(fmt.Sprintf("encoded/%s.txt", file), []byte(*encoded), 0644)
}
