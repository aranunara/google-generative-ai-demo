package entities

import (
	"fmt"
	"time"

	"tryon-demo/internal/domain/valueobjects"
)

type TryOnRequestID string

type TryOnRequest struct {
	id           TryOnRequestID
	personImage  *valueobjects.ImageData
	garmentImage *valueobjects.ImageData
	parameters   *valueobjects.TryOnParameters
	createdAt    time.Time
}

func NewTryOnRequest(
	personImage *valueobjects.ImageData,
	garmentImage *valueobjects.ImageData,
	parameters *valueobjects.TryOnParameters,
) (*TryOnRequest, error) {
	if personImage == nil {
		return nil, fmt.Errorf("person image is required")
	}

	if garmentImage == nil {
		return nil, fmt.Errorf("garment image is required")
	}

	if parameters == nil {
		parameters = valueobjects.DefaultTryOnParameters()
	}

	id := TryOnRequestID(fmt.Sprintf("req_%d", time.Now().UnixNano()))

	return &TryOnRequest{
		id:           id,
		personImage:  personImage,
		garmentImage: garmentImage,
		parameters:   parameters,
		createdAt:    time.Now(),
	}, nil
}

func (r *TryOnRequest) ID() TryOnRequestID {
	return r.id
}

func (r *TryOnRequest) PersonImage() *valueobjects.ImageData {
	return r.personImage
}

func (r *TryOnRequest) GarmentImage() *valueobjects.ImageData {
	return r.garmentImage
}

func (r *TryOnRequest) Parameters() *valueobjects.TryOnParameters {
	return r.parameters
}

func (r *TryOnRequest) CreatedAt() time.Time {
	return r.createdAt
}

func (r *TryOnRequest) PrepareImages() error {
	var err error
	
	r.personImage, err = r.personImage.ToJPEG()
	if err != nil {
		return fmt.Errorf("failed to convert person image to JPEG: %w", err)
	}

	r.garmentImage, err = r.garmentImage.ToJPEG()
	if err != nil {
		return fmt.Errorf("failed to convert garment image to JPEG: %w", err)
	}

	return nil
}