package main

import (
	"encoding/json"
	"testing"
	"tryon-demo/test/http"
)

func TestMainParseFunction(t *testing.T) {
	// Get the response JSON from the test data
	responseJSON := http.VirtualTryOnResponse()

	// Parse using the main.go parse function
	response, err := parse([]byte(responseJSON))
	if err != nil {
		t.Fatalf("Failed to parse JSON with main.parse function: %v", err)
	}

	// Verify the response structure
	if len(response.Predictions) == 0 {
		t.Error("Expected at least one prediction in response")
		return
	}

	// Test the first prediction
	firstPrediction := response.Predictions[0]

	// Check that MimeType is set
	if firstPrediction.MimeType == "" {
		t.Error("Expected MimeType to be set")
	}

	// Verify it's an image type
	expectedMimeType := "image/png"
	if firstPrediction.MimeType != expectedMimeType {
		t.Errorf("Expected MimeType to be %s, got %s", expectedMimeType, firstPrediction.MimeType)
	}

	// Check that BytesBase64Encoded is set
	if firstPrediction.BytesBase64Encoded == "" {
		t.Error("Expected BytesBase64Encoded to be set")
	}

	t.Logf("Successfully parsed response with main.parse function: %d predictions", len(response.Predictions))
	t.Logf("First prediction: MimeType=%s, Base64Length=%d", 
		firstPrediction.MimeType, len(firstPrediction.BytesBase64Encoded))
}

func TestJSONResponseStructure(t *testing.T) {
	// Simulate the JSON response structure that will be sent to the frontend
	images := []map[string]string{
		{
			"id":   "image_0",
			"data": "dGVzdCBkYXRh", // base64 for "test data"
			"type": "image/png",
		},
		{
			"id":   "image_1", 
			"data": "dGVzdCBkYXRhMg==", // base64 for "test data2"
			"type": "image/png",
		},
	}

	response := map[string]any{
		"success": true,
		"images":  images,
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	// Unmarshal back to verify structure
	var parsedResponse map[string]any
	err = json.Unmarshal(jsonData, &parsedResponse)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify the structure
	if success, ok := parsedResponse["success"].(bool); !ok || !success {
		t.Error("Expected success to be true")
	}

	imagesInterface, ok := parsedResponse["images"]
	if !ok {
		t.Fatal("Expected images field in response")
	}

	imagesList, ok := imagesInterface.([]any)
	if !ok {
		t.Fatal("Expected images to be an array")
	}

	if len(imagesList) != 2 {
		t.Errorf("Expected 2 images, got %d", len(imagesList))
	}

	// Check first image structure
	firstImage, ok := imagesList[0].(map[string]any)
	if !ok {
		t.Fatal("Expected first image to be an object")
	}

	expectedFields := []string{"id", "data", "type"}
	for _, field := range expectedFields {
		if _, ok := firstImage[field]; !ok {
			t.Errorf("Expected field '%s' in image object", field)
		}
	}

	t.Log("JSON response structure test passed")
}