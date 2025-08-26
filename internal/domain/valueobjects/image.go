package valueobjects

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/webp"
)

type ImageFormat string

const (
	JPEG ImageFormat = "jpeg"
	PNG  ImageFormat = "png"
	GIF  ImageFormat = "gif"
	WEBP ImageFormat = "webp"
)

type ImageData struct {
	data   []byte
	format ImageFormat
}

func NewImageData(data []byte) (*ImageData, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("image data cannot be empty")
	}

	format, err := detectFormat(data)
	if err != nil {
		return nil, fmt.Errorf("unsupported image format: %w", err)
	}

	return &ImageData{
		data:   data,
		format: format,
	}, nil
}

func (i *ImageData) Data() []byte {
	return i.data
}

func (i *ImageData) Format() ImageFormat {
	return i.format
}

func (i *ImageData) IsJPEG() bool {
	return i.format == JPEG
}

func (i *ImageData) ToJPEG() (*ImageData, error) {
	if i.IsJPEG() {
		return i, nil
	}

	reader := bytes.NewReader(i.data)
	img, _, err := image.Decode(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	var buf bytes.Buffer
	opts := &jpeg.Options{Quality: 90}
	if err := jpeg.Encode(&buf, img, opts); err != nil {
		return nil, fmt.Errorf("failed to encode to JPEG: %w", err)
	}

	return &ImageData{
		data:   buf.Bytes(),
		format: JPEG,
	}, nil
}

func (i *ImageData) ToBase64() string {
	return base64.StdEncoding.EncodeToString(i.data)
}

func detectFormat(data []byte) (ImageFormat, error) {
	reader := bytes.NewReader(data)
	_, format, err := image.DecodeConfig(reader)
	if err != nil {
		return "", err
	}

	switch format {
	case "jpeg":
		return JPEG, nil
	case "png":
		return PNG, nil
	case "gif":
		return GIF, nil
	case "webp":
		return WEBP, nil
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}