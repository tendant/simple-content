# Testing Strategy for Simple Content Management System

This directory contains tests for the Simple Content Management System. The testing strategy includes both unit tests and integration tests to ensure the system works correctly.

## Test Structure

The tests are organized as follows:

- **Unit Tests**: Located alongside the code they test in the `internal/` directory.
  - Domain model tests: `internal/domain/*_test.go`
  - Repository tests: `internal/repository/memory/*_test.go`
  - Service tests: `internal/service/*_test.go`

- **Integration Tests**: Located in the `tests/integration/` directory.
  - API workflow tests: `tests/integration/*_test.go`

- **Test Utilities**: Located in the `tests/testutil/` directory.
  - Server setup: `tests/testutil/server.go`
  - Helper functions: `tests/testutil/helpers.go`

## Running Tests

To run all tests:

```bash
go test ./...
```

To run tests with verbose output:

```bash
go test ./... -v
```

To run specific test packages:

```bash
# Run only domain tests
go test ./internal/domain -v

# Run only repository tests
go test ./internal/repository/memory -v

# Run only service tests
go test ./internal/service -v

# Run only integration tests
go test ./tests/integration -v
```

To run tests with coverage:

```bash
go test -cover ./...
```

To generate a coverage report:

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Test Types

### Unit Tests

Unit tests focus on testing individual components in isolation:

- **Domain Tests**: Verify that the domain models behave correctly.
- **Repository Tests**: Ensure that the repository layer correctly stores and retrieves data.
- **Service Tests**: Validate the business logic in the service layer.

### Integration Tests

Integration tests verify that the components work together correctly:

- **API Tests**: Test the API endpoints and workflows.
- **Derived Content Tests**: Verify the derived content functionality.

## Key Test Cases

### Derived Content

The derived content tests verify:

1. Creating derived content from original content
2. Setting different metadata for derived content
3. Creating multiple levels of derived content
4. Enforcing the maximum derivation depth (5 levels)
5. Retrieving direct derived content
6. Retrieving the entire derivation tree
7. Verifying metadata independence between original and derived content

## Test Utilities

The test utilities provide:

1. A test server setup with in-memory repositories and storage
2. Helper functions for common API operations
3. Response models for parsing API responses
