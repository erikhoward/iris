package openai

import "io"

// FilePurpose represents the intended use of an uploaded file.
type FilePurpose string

const (
	FilePurposeAssistants FilePurpose = "assistants"
	FilePurposeBatch      FilePurpose = "batch"
	FilePurposeFineTune   FilePurpose = "fine-tune"
	FilePurposeVision     FilePurpose = "vision"
	FilePurposeUserData   FilePurpose = "user_data"
	FilePurposeEvals      FilePurpose = "evals"
)

// File represents an uploaded file in OpenAI.
type File struct {
	ID        string      `json:"id"`
	Object    string      `json:"object"`
	Bytes     int64       `json:"bytes"`
	CreatedAt int64       `json:"created_at"`
	ExpiresAt *int64      `json:"expires_at,omitempty"`
	Filename  string      `json:"filename"`
	Purpose   FilePurpose `json:"purpose"`
}

// ExpiresAfter defines file expiration policy.
type ExpiresAfter struct {
	Anchor  string `json:"anchor"`
	Seconds int    `json:"seconds"`
}

// FileUploadRequest contains parameters for uploading a file.
type FileUploadRequest struct {
	File         io.Reader
	Filename     string
	Purpose      FilePurpose
	ExpiresAfter *ExpiresAfter
}

// FileListRequest contains parameters for listing files.
type FileListRequest struct {
	Purpose *FilePurpose
	Limit   *int
	After   *string
	Order   *string
}

// FileListResponse contains paginated file results.
type FileListResponse struct {
	Object  string `json:"object"`
	Data    []File `json:"data"`
	HasMore bool   `json:"has_more"`
	FirstID string `json:"first_id,omitempty"`
	LastID  string `json:"last_id,omitempty"`
}

// FileDeleteResponse contains the result of a file deletion.
type FileDeleteResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Deleted bool   `json:"deleted"`
}
