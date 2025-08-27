package valueobjects

type VideoData struct {
	data []byte
}

func NewVideoData(data []byte) *VideoData {
	return &VideoData{data: data}
}

func (v *VideoData) Data() []byte {
	return v.data
}
