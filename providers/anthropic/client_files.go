package anthropic

import (
	"net/http"
)

// filesPath is the API endpoint for files.
const filesPath = "/v1/files"

// buildFilesHeaders constructs headers for Files API requests.
func (p *Anthropic) buildFilesHeaders() http.Header {
	headers := p.buildHeaders()

	beta := p.config.FilesAPIBeta
	if beta == "" {
		beta = DefaultFilesAPIBeta
	}
	headers.Set("anthropic-beta", beta)

	return headers
}
