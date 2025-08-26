package entities

import (
	"fmt"
	"time"

	"tryon-demo/internal/domain/valueobjects"
)

type TryOnResultID string

type TryOnResult struct {
	id        TryOnResultID
	requestID TryOnRequestID
	images    []*valueobjects.ImageData
	createdAt time.Time
}

func NewTryOnResult(requestID TryOnRequestID, images []*valueobjects.ImageData) *TryOnResult {
	id := TryOnResultID(fmt.Sprintf("result_%d", time.Now().UnixNano()))
	
	return &TryOnResult{
		id:        id,
		requestID: requestID,
		images:    images,
		createdAt: time.Now(),
	}
}

func (r *TryOnResult) ID() TryOnResultID {
	return r.id
}

func (r *TryOnResult) RequestID() TryOnRequestID {
	return r.requestID
}

func (r *TryOnResult) Images() []*valueobjects.ImageData {
	return r.images
}

func (r *TryOnResult) CreatedAt() time.Time {
	return r.createdAt
}

func (r *TryOnResult) HasImages() bool {
	return len(r.images) > 0
}