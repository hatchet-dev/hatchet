# Integration Testing Guide

This guide explains how to run integration tests for the Hatchet Ruby SDK that make actual API calls to test the real functionality.

## Overview

The Ruby SDK includes two types of tests:

1. **Unit Tests** (`spec/hatchet/`) - Fast tests that use mocks and don't require API credentials
2. **Integration Tests** (`spec/integration/`) - Tests that make real API calls and require valid credentials

## Running Unit Tests

Unit tests run by default and don't require any setup:

```bash
bundle exec rspec
```

## Running Integration Tests

Integration tests require a valid Hatchet API token and will be skipped automatically if no credentials are found.

### Prerequisites

1. Valid Hatchet API token (JWT)
2. Access to a Hatchet tenant with some workflow/task run data

### Setup

Set your API credentials as environment variables:

```bash
export HATCHET_CLIENT_TOKEN="your-jwt-token-here"
```

Optional additional configuration:

```bash
export HATCHET_CLIENT_TENANT_ID="your-tenant-id"  # Will be extracted from JWT if not provided
```

### Running Integration Tests

#### Run only integration tests:
```bash
bundle exec rspec spec/integration/ --tag integration
```

#### Run all tests including integration:
```bash
bundle exec rspec
```

#### Force integration tests even without token (will skip individual tests):
```bash
RUN_INTEGRATION_TESTS=true bundle exec rspec spec/integration/
```

### What Integration Tests Cover

The integration tests for the Runs feature test:

#### ✅ Basic API Connectivity
- List workflow runs
- List with pagination
- Filter operations
- Empty result handling

#### ✅ Workflow Run Operations
- Get workflow run details
- Get workflow run status  
- Get workflow run results
- Data structure validation

#### ✅ Task Run Operations
- Get task run details
- Data structure validation

#### ✅ Workflow Creation (Conditional)
- Attempt to create workflow runs
- Gracefully handles missing test workflows

#### ✅ Error Handling
- Invalid IDs
- Invalid date ranges
- Network errors

#### ✅ Data Structure Validation
- Response object structure
- Required fields
- Data types

#### ⚠️ Bulk Operations (Structure Only)
- Tests helper classes
- Does NOT test actual bulk cancel/replay (too dangerous for integration tests)

## Sample Output

### With Valid Credentials
```
🔗 Running integration tests with real API credentials
   Tenant ID: abc123-def456
   Server URL: https://app.dev.hatchet-tools.com

Hatchet::Features::Runs Integration
  API connectivity and basic operations
    ✓ can list workflow runs without error
    ✓ returns a TaskSummaryList when listing runs
    ✓ can list runs with pagination without error
    ✓ returns an array when using list_with_pagination
    ✓ can filter runs by various parameters
    ✓ handles empty results gracefully
  workflow run operations
    ✓ can retrieve workflow run details
    ✓ returns WorkflowRunDetails when getting a workflow run
    ✓ can get workflow run status
    ✓ returns a TaskStatus when getting status
    ✓ can get workflow run result
  ...
```

### Without Credentials
```
⚠️  Integration tests skipped (no HATCHET_CLIENT_TOKEN found)
   Set HATCHET_CLIENT_TOKEN or RUN_INTEGRATION_TESTS=true to run integration tests

Finished in 0.00123 seconds
0 examples, 0 failures (all integration tests skipped)
```

## Safety Considerations

The integration tests are designed to be safe:

- ✅ **Read Operations**: Tests safely read existing data
- ✅ **Safe Creation**: May attempt to create test workflows but handles failures gracefully
- ❌ **No Destructive Operations**: Does not actually cancel or replay runs in integration tests
- ❌ **No Data Modification**: Does not modify or delete existing workflow runs

## Troubleshooting

### "No recent workflow runs found for testing"
- Your tenant needs some historical workflow runs to test against
- Try running some workflows first, or use a tenant with existing data

### "Integration tests require HATCHET_CLIENT_TOKEN to be set"
- Make sure your JWT token is properly set in the environment
- Verify the token is valid and not expired

### API Connection Errors
- Check your network connection
- Verify `HATCHET_CLIENT_SERVER_URL` is correct
- Ensure your token has the necessary permissions

### Workflow Creation Failures
- This is expected if you don't have a workflow named "test-workflow"
- The test will show a warning but won't fail
- Create a simple test workflow if you want to test the full creation flow

## Contributing

When adding new integration tests:

1. Always tag them with `:integration`
2. Use the `IntegrationHelper` methods
3. Handle missing test data gracefully with `skip` or `safely_attempt_operation`
4. Avoid destructive operations
5. Validate response data structures
6. Add appropriate documentation

## CI/CD Integration

For automated testing pipelines:

```bash
# Run only unit tests (fast, no credentials needed)
bundle exec rspec --tag ~integration

# Run integration tests with credentials (slower, requires setup)
HATCHET_CLIENT_TOKEN=$SECRET_TOKEN bundle exec rspec --tag integration
```