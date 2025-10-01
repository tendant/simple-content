#!/bin/bash
# Demo script showing admin CLI capabilities

set -e

echo "=== Simple Content Admin CLI Demo ==="
echo ""

# Use the example to populate data
echo "Step 1: Creating sample data using the admin operations example..."
go run ../../examples/admin-operations/main.go > /dev/null 2>&1 &
EXAMPLE_PID=$!
sleep 2

# Note: The example uses in-memory storage, so we can't actually query it
# This is just for demonstration purposes
echo "Note: This demo uses memory database (empty). "
echo "In production, you would connect to PostgreSQL with real data."
echo ""

# Build the admin tool
echo "Step 2: Building admin CLI..."
go build -o admin .
echo "✓ Built successfully"
echo ""

# Show help
echo "Step 3: Display help..."
DATABASE_TYPE=memory ./admin help | head -20
echo ""

# Test commands with memory database (empty)
echo "Step 4: Test commands (with empty memory database)..."
echo ""

echo "Command: ./admin list"
DATABASE_TYPE=memory ./admin list
echo ""

echo "Command: ./admin count"
DATABASE_TYPE=memory ./admin count
echo ""

echo "Command: ./admin stats"
DATABASE_TYPE=memory ./admin stats
echo ""

echo "Command: ./admin count --json"
DATABASE_TYPE=memory ./admin count --json
echo ""

# Example with PostgreSQL (commented out)
cat << 'EOF'
Step 5: Example with PostgreSQL (requires running database):

# Export database connection
export DATABASE_TYPE=postgres
export DATABASE_URL="postgres://user:pass@localhost/dbname"
export DB_SCHEMA=content

# List all contents
./admin list

# List with filtering
./admin list --tenant-id=550e8400-e29b-41d4-a716-446655440000

# Count by status
./admin count --status=uploaded

# Get statistics
./admin stats

# Export as JSON for scripting
./admin list --limit=1000 --json > export.json
./admin stats --json | jq '.statistics.by_status'

EOF

echo ""
echo "=== Demo Complete ==="
echo ""
echo "The admin CLI is ready to use!"
echo "Try: ./admin help"
echo ""
echo "For PostgreSQL:"
echo "  export DATABASE_URL='postgres://user:pass@localhost/dbname'"
echo "  export DATABASE_TYPE=postgres"
echo "  ./admin list"
