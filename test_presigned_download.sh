#!/bin/bash

# Test script for presigned download and preview URLs
# This script tests the filesystem storage presigned URL feature for downloads and previews

set -e

API_BASE_URL="${API_BASE_URL:-http://localhost:8080/api/v1}"
SECRET_KEY="${FS_SIGNATURE_SECRET_KEY:-my-secret-key-12345}"

echo "================================"
echo "Presigned Download/Preview URL Test"
echo "================================"
echo ""
echo "API Base URL: $API_BASE_URL"
echo "Secret Key: $SECRET_KEY"
echo ""

# Step 1: Create a content
echo "Step 1: Creating content..."
OWNER_ID=$(uuidgen)
TENANT_ID=$(uuidgen)

CONTENT_RESPONSE=$(curl -s -X POST "$API_BASE_URL/contents" \
  -H "Content-Type: application/json" \
  -d "{
    \"owner_id\": \"$OWNER_ID\",
    \"tenant_id\": \"$TENANT_ID\",
    \"name\": \"Test Document\",
    \"document_type\": \"text/plain\"
  }")

echo "Content created: $CONTENT_RESPONSE"
CONTENT_ID=$(echo "$CONTENT_RESPONSE" | jq -r '.id')
echo "Content ID: $CONTENT_ID"
echo ""

# Step 2: Upload data to the content
echo "Step 2: Uploading data to content..."
TEST_CONTENT="Hello, this is a test document for presigned download URLs!"
echo "$TEST_CONTENT" > /tmp/test_presigned_download.txt

UPLOAD_RESPONSE=$(curl -s -X POST "$API_BASE_URL/contents/$CONTENT_ID/upload" \
  -H "Content-Type: text/plain" \
  --data-binary @/tmp/test_presigned_download.txt)

echo "Upload response: $UPLOAD_RESPONSE"
echo ""

# Step 3: Get content details to retrieve presigned download URL
echo "Step 3: Getting content details with download URL..."
DETAILS_RESPONSE=$(curl -s "$API_BASE_URL/contents/$CONTENT_ID/details")
echo "Details response: $DETAILS_RESPONSE"
echo ""

DOWNLOAD_URL=$(echo "$DETAILS_RESPONSE" | jq -r '.download')
PREVIEW_URL=$(echo "$DETAILS_RESPONSE" | jq -r '.preview')

echo "Download URL: $DOWNLOAD_URL"
echo "Preview URL: $PREVIEW_URL"
echo ""

# Step 4: Test presigned download URL
echo "Step 4: Testing presigned download URL..."
if echo "$DOWNLOAD_URL" | grep -q "signature="; then
  echo "✓ Download URL contains signature (presigned URL is enabled)"

  # Download using presigned URL
  DOWNLOADED_CONTENT=$(curl -s "$DOWNLOAD_URL")

  if [ "$DOWNLOADED_CONTENT" == "$TEST_CONTENT" ]; then
    echo "✓ Downloaded content matches uploaded content"
  else
    echo "✗ Downloaded content does not match!"
    echo "Expected: $TEST_CONTENT"
    echo "Got: $DOWNLOADED_CONTENT"
    exit 1
  fi
else
  echo "⚠ Download URL does not contain signature (presigned URLs not configured)"
  echo "To enable presigned URLs, set FS_SIGNATURE_SECRET_KEY environment variable"
fi
echo ""

# Step 5: Test presigned preview URL
echo "Step 5: Testing presigned preview URL..."
if echo "$PREVIEW_URL" | grep -q "signature="; then
  echo "✓ Preview URL contains signature (presigned URL is enabled)"

  # Preview using presigned URL
  PREVIEW_CONTENT=$(curl -s "$PREVIEW_URL")

  if [ "$PREVIEW_CONTENT" == "$TEST_CONTENT" ]; then
    echo "✓ Preview content matches uploaded content"
  else
    echo "✗ Preview content does not match!"
    echo "Expected: $TEST_CONTENT"
    echo "Got: $PREVIEW_CONTENT"
    exit 1
  fi
else
  echo "⚠ Preview URL does not contain signature (presigned URLs not configured)"
  echo "To enable presigned URLs, set FS_SIGNATURE_SECRET_KEY environment variable"
fi
echo ""

# Step 6: Test expired URL (if presigned URLs are enabled)
if echo "$DOWNLOAD_URL" | grep -q "signature="; then
  echo "Step 6: Testing expired URL protection..."

  # Extract signature and create an expired URL
  OBJECT_KEY=$(echo "$DOWNLOAD_URL" | sed -E 's/.*\/download\/([^?]+).*/\1/')
  EXPIRED_TIME=$(($(date +%s) - 3600)) # 1 hour ago

  # Try to access with expired timestamp (signature will be invalid)
  EXPIRED_URL=$(echo "$DOWNLOAD_URL" | sed -E "s/expires=[0-9]+/expires=$EXPIRED_TIME/")

  HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$EXPIRED_URL")

  if [ "$HTTP_CODE" == "403" ]; then
    echo "✓ Expired URL correctly rejected (403 Forbidden)"
  else
    echo "✗ Expired URL not rejected! HTTP code: $HTTP_CODE"
    exit 1
  fi
  echo ""

  # Step 7: Test invalid signature
  echo "Step 7: Testing invalid signature protection..."

  INVALID_URL=$(echo "$DOWNLOAD_URL" | sed -E 's/signature=[^&]+/signature=invalid123/')

  HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$INVALID_URL")

  if [ "$HTTP_CODE" == "403" ]; then
    echo "✓ Invalid signature correctly rejected (403 Forbidden)"
  else
    echo "✗ Invalid signature not rejected! HTTP code: $HTTP_CODE"
    exit 1
  fi
  echo ""
fi

# Cleanup
rm -f /tmp/test_presigned_download.txt

echo "================================"
echo "All tests passed! ✓"
echo "================================"
