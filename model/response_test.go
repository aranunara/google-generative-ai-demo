package model

import (
	"encoding/json"
	"testing"

	"tryon-demo/test/http"
)

func TestVirtualTryOnResponseParsing(t *testing.T) {
	// Get the response JSON from the test data
	responseJSON := http.VirtualTryOnResponse()

	// Parse the JSON into our struct
	var response VirtualTryOnResponse
	err := json.Unmarshal([]byte(responseJSON), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
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

	// Verify the base64 data starts with a valid PNG header (when decoded)
	if len(firstPrediction.BytesBase64Encoded) < 10 {
		t.Error("Expected BytesBase64Encoded to contain substantial data")
	}

	t.Logf("Successfully parsed response with %d predictions", len(response.Predictions))
	t.Logf("First prediction: MimeType=%s, Base64Length=%d", 
		firstPrediction.MimeType, len(firstPrediction.BytesBase64Encoded))
}

func TestVirtualTryOnResponseJSONMarshaling(t *testing.T) {
	// Create a test response
	testResponse := VirtualTryOnResponse{
		Predictions: []Prediction{
			{
				MimeType:           "image/png",
				BytesBase64Encoded: "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChAI9jz22jQAAAABJRU5ErkJggg==",
			},
		},
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(testResponse)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	// Unmarshal back
	var parsedResponse VirtualTryOnResponse
	err = json.Unmarshal(jsonData, &parsedResponse)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify data integrity
	if len(parsedResponse.Predictions) != 1 {
		t.Errorf("Expected 1 prediction, got %d", len(parsedResponse.Predictions))
	}

	if parsedResponse.Predictions[0].MimeType != testResponse.Predictions[0].MimeType {
		t.Error("MimeType not preserved during JSON marshaling/unmarshaling")
	}

	if parsedResponse.Predictions[0].BytesBase64Encoded != testResponse.Predictions[0].BytesBase64Encoded {
		t.Error("BytesBase64Encoded not preserved during JSON marshaling/unmarshaling")
	}

	t.Log("JSON marshaling and unmarshaling test passed")
}