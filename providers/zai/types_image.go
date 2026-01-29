package zai

// zaiImageRequest represents a request to the Z.ai image generations endpoint.
type zaiImageRequest struct {
	Model   string `json:"model"`
	Prompt  string `json:"prompt"`
	Quality string `json:"quality,omitempty"`
	Size    string `json:"size,omitempty"`
	UserID  string `json:"user_id,omitempty"`
}

// zaiImageResponse represents a response from the Z.ai image API.
type zaiImageResponse struct {
	Created       int64                   `json:"created"`
	Data          []zaiImageData          `json:"data"`
	ContentFilter []zaiImageContentFilter `json:"content_filter,omitempty"`
}

// zaiImageData represents a single image in the response.
type zaiImageData struct {
	URL string `json:"url"`
}

// zaiImageContentFilter contains content safety information.
type zaiImageContentFilter struct {
	Role  string `json:"role"`
	Level int    `json:"level"`
}

// zaiAsyncImageResponse is returned when submitting an async image request.
type zaiAsyncImageResponse struct {
	Model      string `json:"model"`
	ID         string `json:"id"`
	RequestID  string `json:"request_id"`
	TaskStatus string `json:"task_status"`
}

// zaiAsyncResultResponse contains the result of an async image request.
type zaiAsyncResultResponse struct {
	Model       string           `json:"model"`
	TaskStatus  string           `json:"task_status"`
	ImageResult []zaiImageResult `json:"image_result,omitempty"`
	RequestID   string           `json:"request_id"`
}

// zaiImageResult contains the result image data.
type zaiImageResult struct {
	URL string `json:"url"`
}
