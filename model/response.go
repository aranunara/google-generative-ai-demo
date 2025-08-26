package model

// VirtualTryOnResponse represents the response structure from Google's Virtual Try-On API
type VirtualTryOnResponse struct {
	Predictions []Prediction `json:"predictions"`
}

// Prediction represents a single prediction result
type Prediction struct {
	MimeType            string `json:"mimeType"`
	BytesBase64Encoded  string `json:"bytesBase64Encoded"`
}
