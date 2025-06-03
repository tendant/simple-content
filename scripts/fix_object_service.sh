#!/bin/bash

# Update all NewObjectService calls to include contentMetadataRepo
find /Users/bd/Workspace/Torpago/simple-content -type f -name "*.go" -exec grep -l "objectService := service.NewObjectService" {} \; | while read file; do
  echo "Updating $file"
  # Check if the file already has the contentMetadataRepo parameter
  if grep -q "objectService := service.NewObjectService(.*contentMetadataRepo" "$file"; then
    echo "  Already updated"
  else
    # Replace the pattern
    sed -i '' 's/objectService := service.NewObjectService(\([^)]*\))/objectService := service.NewObjectService(\1, contentMetadataRepo)/g' "$file"
    echo "  Updated"
  fi
done

# Special handling for object_service_test.go
if grep -q "service := service.NewObjectService(.*contentMetadataRepo" "/Users/bd/Workspace/Torpago/simple-content/pkg/service/object_service_test.go"; then
  echo "object_service_test.go already updated"
else
  sed -i '' 's/service := service.NewObjectService(\([^)]*\))/service := service.NewObjectService(\1, contentMetadataRepo)/g' "/Users/bd/Workspace/Torpago/simple-content/pkg/service/object_service_test.go"
  echo "Updated object_service_test.go"
fi

# Special handling for server.go
if grep -q "objectService := service.NewObjectService(.*contentMetadataRepo" "/Users/bd/Workspace/Torpago/simple-content/tests/testutil/server.go"; then
  echo "server.go already updated"
else
  sed -i '' 's/objectService := service.NewObjectService(\([^)]*\))/objectService := service.NewObjectService(\1, contentMetadataRepo)/g' "/Users/bd/Workspace/Torpago/simple-content/tests/testutil/server.go"
  echo "Updated server.go"
fi

# Special handling for cmd/server/main.go
if grep -q "objectService := service.NewObjectService(.*contentMetadataRepo" "/Users/bd/Workspace/Torpago/simple-content/cmd/server/main.go"; then
  echo "cmd/server/main.go already updated"
else
  sed -i '' 's/objectService := service.NewObjectService(\([^)]*\))/objectService := service.NewObjectService(\1, contentMetadataRepo)/g' "/Users/bd/Workspace/Torpago/simple-content/cmd/server/main.go"
  echo "Updated cmd/server/main.go"
fi
