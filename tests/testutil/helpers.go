package testutil

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// ContentResponse represents the response from content-related API endpoints
type ContentResponse struct {
	ID              string `json:"id"`
	ParentID        string `json:"parent_id,omitempty"`
	OwnerID         string `json:"owner_id"`
	TenantID        string `json:"tenant_id"`
	Status          string `json:"status"`
	DerivationType  string `json:"derivation_type"`
	DerivationLevel int    `json:"derivation_level"`
}

// ContentMetadataResponse represents the response from metadata-related API endpoints
type ContentMetadataResponse struct {
	ContentID   string                 `json:"content_id"`
	ContentType string                 `json:"content_type"`
	Title       string                 `json:"title,omitempty"`
	Description string                 `json:"description,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	FileSize    int64                  `json:"file_size,omitempty"`
	CreatedBy   string                 `json:"created_by,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// CreateContent creates a new content via the API
func CreateContent(t *testing.T, serverURL string) ContentResponse {
	reqBody := map[string]string{
		"owner_id":  "00000000-0000-0000-0000-000000000001",
		"tenant_id": "00000000-0000-0000-0000-000000000001",
	}
	reqJSON, err := json.Marshal(reqBody)
	require.NoError(t, err)

	resp, err := http.Post(serverURL+"/content", "application/json", bytes.NewBuffer(reqJSON))
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var content ContentResponse
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	defer resp.Body.Close()

	err = json.Unmarshal(body, &content)
	require.NoError(t, err)

	return content
}

// CreateDerivedContent creates a derived content via the API
func CreateDerivedContent(t *testing.T, serverURL, parentID string) ContentResponse {
	reqBody := map[string]string{
		"owner_id":  "00000000-0000-0000-0000-000000000001",
		"tenant_id": "00000000-0000-0000-0000-000000000001",
	}
	reqJSON, err := json.Marshal(reqBody)
	require.NoError(t, err)

	resp, err := http.Post(serverURL+"/content/"+parentID+"/derive", "application/json", bytes.NewBuffer(reqJSON))
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var content ContentResponse
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	defer resp.Body.Close()

	err = json.Unmarshal(body, &content)
	require.NoError(t, err)

	return content
}

// SetContentMetadata sets metadata for a content via the API
func SetContentMetadata(t *testing.T, serverURL, contentID string, metadata map[string]interface{}) {
	reqJSON, err := json.Marshal(metadata)
	require.NoError(t, err)

	req, err := http.NewRequest("PUT", serverURL+"/content/"+contentID+"/metadata", bytes.NewBuffer(reqJSON))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	defer resp.Body.Close()
}

// GetContentMetadata gets metadata for a content via the API
func GetContentMetadata(t *testing.T, serverURL, contentID string) ContentMetadataResponse {
	resp, err := http.Get(serverURL + "/content/" + contentID + "/metadata")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var metadata ContentMetadataResponse
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	defer resp.Body.Close()

	err = json.Unmarshal(body, &metadata)
	require.NoError(t, err)

	return metadata
}

// GetDerivedContent gets derived content via the API
func GetDerivedContent(t *testing.T, serverURL, parentID string) []ContentResponse {
	resp, err := http.Get(serverURL + "/content/" + parentID + "/derived")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var contents []ContentResponse
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	defer resp.Body.Close()

	err = json.Unmarshal(body, &contents)
	require.NoError(t, err)

	return contents
}

// GetDerivedContentTree gets the derived content tree via the API
func GetDerivedContentTree(t *testing.T, serverURL, rootID string) []ContentResponse {
	resp, err := http.Get(serverURL + "/content/" + rootID + "/derived-tree")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var contents []ContentResponse
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	defer resp.Body.Close()

	err = json.Unmarshal(body, &contents)
	require.NoError(t, err)

	return contents
}

// AttemptCreateDerivedContent attempts to create a derived content and returns the response
func AttemptCreateDerivedContent(t *testing.T, serverURL, parentID string) *http.Response {
	reqBody := map[string]string{
		"owner_id":  "00000000-0000-0000-0000-000000000001",
		"tenant_id": "00000000-0000-0000-0000-000000000001",
	}
	reqJSON, err := json.Marshal(reqBody)
	require.NoError(t, err)

	resp, err := http.Post(serverURL+"/content/"+parentID+"/derive", "application/json", bytes.NewBuffer(reqJSON))
	require.NoError(t, err)

	return resp
}
