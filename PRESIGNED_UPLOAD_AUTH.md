# Presigned Upload Authentication Guide

This document explains how to secure presigned uploads to filesystem storage with HMAC signature-based authentication.

## Overview

Presigned URLs for filesystem storage can be secured using HMAC-SHA256 signatures, similar to how AWS S3 presigned URLs work. This prevents unauthorized uploads while maintaining the convenience of direct client-to-storage uploads.

## How It Works

### 1. **URL Signing (Server-Side)**

When generating a presigned upload URL, the server:
1. Creates a signature payload: `METHOD|PATH|EXPIRES`
2. Generates HMAC-SHA256 signature using a secret key
3. Appends signature and expiration to the URL

```
PUT|/upload/{objectKey}|{timestamp} → HMAC-SHA256 → signature
```

### 2. **URL Validation (Server-Side)**

When receiving an upload request, the server:
1. Extracts signature and expiration from query parameters
2. Checks if the URL has expired
3. Recreates the signature payload
4. Compares signatures using constant-time comparison
5. Rejects if invalid or expired

## Configuration

### Environment Variables

```bash
# Required for presigned uploads
export FS_BASE_DIR=./data/storage
export FS_URL_PREFIX=http://localhost:8080/api/v1

# Optional: Enable authentication (recommended for production)
export FS_SIGNATURE_SECRET_KEY=your-secret-key-here-min-32-chars

# Optional: Set expiration time (default: 3600 seconds / 1 hour)
export FS_PRESIGN_EXPIRES_SECONDS=1800  # 30 minutes
```

### Security Recommendations

1. **Secret Key Requirements:**
   - Minimum 32 characters
   - Use cryptographically secure random string
   - Store in environment variable or secrets manager
   - Rotate periodically

2. **Generate Secure Secret:**
   ```bash
   # Generate a secure random secret key
   openssl rand -base64 32
   ```

3. **Never Commit Secret Keys:**
   - Add `.env` to `.gitignore`
   - Use different keys per environment
   - Use secrets management in production (AWS Secrets Manager, HashiCorp Vault, etc.)

## Authentication Modes

### Mode 1: Unauthenticated (Not Recommended for Production)

If `FS_SIGNATURE_SECRET_KEY` is not set, URLs are not signed:

```
http://localhost:8080/api/v1/upload/originals/objects/ab/cd1234_file.pdf
```

**Security:** ⚠️ Anyone with the URL can upload
**Use Case:** Local development/testing only

### Mode 2: Authenticated (Recommended)

If `FS_SIGNATURE_SECRET_KEY` is set, URLs include signature and expiration:

```
http://localhost:8080/api/v1/upload/originals/objects/ab/cd1234_file.pdf?signature=a1b2c3d4...&expires=1696789012
```

**Security:** ✅ Only signed URLs accepted, time-limited
**Use Case:** Production deployments

## API Usage

### Step 1: Create Content and Object

```bash
# Create content
CONTENT_RESPONSE=$(curl -X POST http://localhost:8080/api/v1/contents \
  -H "Content-Type: application/json" \
  -d '{
    "owner_id": "550e8400-e29b-41d4-a716-446655440000",
    "tenant_id": "550e8400-e29b-41d4-a716-446655440001",
    "name": "My Document",
    "document_type": "application/pdf"
  }')

CONTENT_ID=$(echo $CONTENT_RESPONSE | jq -r '.id')

# Create object
OBJECT_RESPONSE=$(curl -X POST http://localhost:8080/api/v1/contents/$CONTENT_ID/objects \
  -H "Content-Type: application/json" \
  -d '{
    "storage_backend_name": "fs",
    "version": 1
  }')

OBJECT_ID=$(echo $OBJECT_RESPONSE | jq -r '.id')
```

### Step 2: Get Presigned Upload URL

```bash
# Get presigned URL (automatically signed if FS_SIGNATURE_SECRET_KEY is set)
URL_RESPONSE=$(curl -X GET http://localhost:8080/api/v1/objects/$OBJECT_ID/upload-url)
UPLOAD_URL=$(echo $URL_RESPONSE | jq -r '.url')

echo "Upload URL: $UPLOAD_URL"
```

**Example URLs:**

Unauthenticated:
```
http://localhost:8080/api/v1/upload/originals/objects/ab/cd1234_file.pdf
```

Authenticated:
```
http://localhost:8080/api/v1/upload/originals/objects/ab/cd1234_file.pdf?signature=e8f7d6c5b4a3...&expires=1696789012
```

### Step 3: Upload File

```bash
# Upload file using the presigned URL
curl -X PUT "$UPLOAD_URL" \
  -H "Content-Type: application/pdf" \
  --data-binary "@document.pdf"
```

**Response:**
- `200 OK` - Upload successful
- `401 Unauthorized` - Missing signature/expiration (when auth enabled)
- `403 Forbidden` - Invalid signature or expired URL
- `500 Internal Server Error` - Upload failed

## Error Responses

### Missing Signature (Auth Enabled)

```bash
# Try upload without signature
curl -X PUT http://localhost:8080/api/v1/upload/originals/objects/ab/cd1234_file.pdf \
  --data-binary "@file.pdf"
```

Response:
```json
{
  "error": {
    "code": "missing_signature",
    "message": "signature parameter is required"
  }
}
```

### Invalid Signature

```bash
# Try upload with wrong signature
curl -X PUT "http://localhost:8080/api/v1/upload/originals/objects/ab/cd1234_file.pdf?signature=invalid&expires=9999999999" \
  --data-binary "@file.pdf"
```

Response:
```json
{
  "error": {
    "code": "invalid_signature",
    "message": "invalid signature"
  }
}
```

### Expired URL

```bash
# Try upload with expired URL
curl -X PUT "http://localhost:8080/api/v1/upload/originals/objects/ab/cd1234_file.pdf?signature=...&expires=1" \
  --data-binary "@file.pdf"
```

Response:
```json
{
  "error": {
    "code": "invalid_signature",
    "message": "presigned URL has expired"
  }
}
```

## Client Implementation Examples

### JavaScript/TypeScript

```typescript
class AuthenticatedUploadClient {
  async uploadFile(file: File, contentId: string) {
    // Step 1: Create object
    const objectResponse = await fetch(
      `/api/v1/contents/${contentId}/objects`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          storage_backend_name: 'fs',
          version: 1
        })
      }
    );
    const object = await objectResponse.json();

    // Step 2: Get presigned URL (automatically signed by server)
    const urlResponse = await fetch(
      `/api/v1/objects/${object.id}/upload-url`
    );
    const { url: uploadUrl } = await urlResponse.json();

    // Step 3: Upload directly to presigned URL
    const uploadResponse = await fetch(uploadUrl, {
      method: 'PUT',
      headers: { 'Content-Type': file.type },
      body: file
    });

    if (!uploadResponse.ok) {
      throw new Error(`Upload failed: ${uploadResponse.statusText}`);
    }

    return object;
  }
}
```

### Python

```python
import requests

class AuthenticatedUploadClient:
    def __init__(self, base_url):
        self.base_url = base_url

    def upload_file(self, file_path, content_id):
        # Step 1: Create object
        object_response = requests.post(
            f"{self.base_url}/contents/{content_id}/objects",
            json={"storage_backend_name": "fs", "version": 1}
        )
        object_data = object_response.json()

        # Step 2: Get presigned URL
        url_response = requests.get(
            f"{self.base_url}/objects/{object_data['id']}/upload-url"
        )
        upload_url = url_response.json()["url"]

        # Step 3: Upload file
        with open(file_path, 'rb') as f:
            upload_response = requests.put(
                upload_url,
                data=f,
                headers={'Content-Type': 'application/octet-stream'}
            )

        upload_response.raise_for_status()
        return object_data
```

### Go

```go
func UploadFile(ctx context.Context, apiBaseURL, contentID, filePath string) error {
    // Step 1: Create object
    objectReq := map[string]interface{}{
        "storage_backend_name": "fs",
        "version": 1,
    }
    objectResp, err := createObject(ctx, apiBaseURL, contentID, objectReq)
    if err != nil {
        return fmt.Errorf("create object: %w", err)
    }

    // Step 2: Get presigned URL
    uploadURL, err := getPresignedURL(ctx, apiBaseURL, objectResp.ID)
    if err != nil {
        return fmt.Errorf("get presigned URL: %w", err)
    }

    // Step 3: Upload file
    file, err := os.Open(filePath)
    if err != nil {
        return fmt.Errorf("open file: %w", err)
    }
    defer file.Close()

    req, err := http.NewRequestWithContext(ctx, "PUT", uploadURL, file)
    if err != nil {
        return fmt.Errorf("create request: %w", err)
    }

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return fmt.Errorf("upload: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("upload failed: %s", resp.Status)
    }

    return nil
}
```

## Security Best Practices

### 1. **Use HTTPS in Production**

Always use HTTPS in production to prevent signature interception:

```bash
# Production configuration
export FS_URL_PREFIX=https://api.example.com/api/v1
```

### 2. **Set Appropriate Expiration Times**

Balance security with user experience:

```bash
# Mobile apps (longer expiration for flaky networks)
export FS_PRESIGN_EXPIRES_SECONDS=3600  # 1 hour

# Web apps (shorter expiration for security)
export FS_PRESIGN_EXPIRES_SECONDS=900   # 15 minutes
```

### 3. **Rotate Secret Keys Regularly**

Implement key rotation:

```bash
# Use versioned keys
export FS_SIGNATURE_SECRET_KEY_V1=old-key
export FS_SIGNATURE_SECRET_KEY_V2=new-key  # Current

# Gradually migrate to new key
```

### 4. **Monitor for Abuse**

Log and monitor upload patterns:
- Failed signature validations
- Expired URL attempts
- Upload rate limits per client

### 5. **Additional Security Layers**

Consider adding:
- Rate limiting per IP/client
- Content-Type validation
- File size limits
- Virus scanning
- Content validation

## Comparison with S3 Presigned URLs

| Feature | S3 Presigned URLs | FS Presigned URLs |
|---------|------------------|-------------------|
| **Signing Algorithm** | AWS Signature V4 | HMAC-SHA256 |
| **Expiration** | ✅ Yes | ✅ Yes |
| **Signature Location** | Query parameters | Query parameters |
| **Secret Management** | AWS credentials | Environment variable |
| **Validation** | AWS infrastructure | Application server |
| **Use Case** | Production (AWS) | Local dev / On-prem |

## Troubleshooting

### Problem: "missing_signature" error

**Cause:** `FS_SIGNATURE_SECRET_KEY` is configured but client is using unsigned URL

**Solution:** Ensure you're getting the URL from `/objects/{id}/upload-url` endpoint, not constructing it manually

### Problem: "invalid_signature" error

**Cause:** Secret key mismatch between URL generation and validation

**Solution:**
1. Verify `FS_SIGNATURE_SECRET_KEY` is set correctly
2. Restart server after changing secret key
3. Get a fresh presigned URL

### Problem: "presigned URL has expired"

**Cause:** URL expired based on `expires` timestamp

**Solution:**
1. Get a new presigned URL
2. Increase `FS_PRESIGN_EXPIRES_SECONDS` if needed
3. Ensure client system clock is accurate

## Migration Guide

### From Unauthenticated to Authenticated

1. **Set Secret Key:**
   ```bash
   export FS_SIGNATURE_SECRET_KEY=$(openssl rand -base64 32)
   ```

2. **Restart Server:**
   ```bash
   ./server-configured
   ```

3. **Update Clients:**
   - No code changes needed!
   - Clients automatically receive signed URLs from the API
   - Only direct URL construction (not recommended) needs updates

### Backward Compatibility

If you need to support both authenticated and unauthenticated uploads temporarily:

1. Run two server instances:
   - One with `FS_SIGNATURE_SECRET_KEY` (authenticated)
   - One without (legacy, unauthenticated)

2. Gradually migrate clients to authenticated endpoint

3. Decommission unauthenticated endpoint

## Conclusion

Presigned URL authentication provides:
- ✅ **Security:** Only authorized uploads accepted
- ✅ **Simplicity:** Clients don't need to implement signing logic
- ✅ **Performance:** Direct uploads bypass application server
- ✅ **Compatibility:** Works with any HTTP client
- ✅ **S3 Parity:** Similar security model to AWS S3

Perfect for local development testing of presigned upload workflows before deploying to production with S3!
