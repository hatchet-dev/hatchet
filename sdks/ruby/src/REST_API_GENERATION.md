# REST API Generation for Hatchet Ruby SDK

This document explains how to generate and use the REST API client for the Hatchet Ruby SDK from the OpenAPI specification.

## Overview

The Hatchet Ruby SDK can generate a comprehensive REST API client from the OpenAPI specification located at `../../../../bin/oas/openapi.yaml`. This provides Ruby bindings for all Hatchet REST API endpoints with full type safety and documentation.

## Prerequisites

1. **Node.js and npm** - Required for OpenAPI Generator CLI
2. **Ruby 3.1+** - For running the generation scripts
3. **Bundler** - For dependency management

## Quick Start

### 1. Install OpenAPI Generator CLI

```bash
npm install -g @openapitools/openapi-generator-cli@latest
```

### 2. Generate the REST API Client

```bash
# Using Rake (recommended)
rake api:generate

# Or using the shell script directly
./scripts/generate.sh

# Or using the Ruby script
ruby scripts/generate_api.rb
```

### 3. Use the Generated Client

```ruby
require 'hatchet-sdk'

# Create main Hatchet configuration
config = Hatchet::Config.new(token: "your-jwt-token")

# Create REST API client
rest_client = Hatchet::Clients.rest_client(config)

# Use specific API endpoints
workflows_api = Hatchet::Clients::Rest::WorkflowApi.new(rest_client)
events_api = Hatchet::Clients::Rest::EventApi.new(rest_client)

# Make API calls
workflows = workflows_api.list_workflows
```

## Available Rake Tasks

- `rake api:generate` - Generate REST API client from OpenAPI spec
- `rake api:clean` - Remove generated REST API client files
- `rake api:regenerate` - Clean and regenerate (clean + generate)
- `rake api:validate` - Validate the OpenAPI specification
- `rake api:install_generator` - Install OpenAPI Generator CLI if not present
- `rake api:info` - Show information about the OpenAPI specification

## Generated Structure

After generation, you'll have:

```
lib/hatchet/clients/rest/
├── api_client.rb           # Main API client
├── configuration.rb        # Client configuration
├── api_error.rb           # Error handling
├── api/                   # API endpoint classes
│   ├── workflow_api.rb    # Workflow operations
│   ├── event_api.rb       # Event operations
│   ├── step_run_api.rb    # Step run operations
│   └── ... (other APIs)
└── models/                # Data models
    ├── workflow.rb        # Workflow model
    ├── event.rb           # Event model
    └── ... (other models)
```

## Configuration

### OpenAPI Generator Configuration

The generation is controlled by `config/openapi_generator_config.json`:

```json
{
  "gemName": "hatchet-sdk-rest",
  "moduleName": "Hatchet::Clients::Rest",
  "library": "faraday",
  "httpLibrary": "faraday",
  "useAutoload": true,
  "sortParamsByRequiredFlag": true,
  "generateExceptionClass": true
}
```

### Custom Patches

The generation process applies custom patches via `scripts/patch_generated_code.rb`:

1. **Module Structure** - Ensures proper Ruby module nesting
2. **Require Statements** - Fixes require paths for gem structure
3. **Documentation** - Adds comprehensive YARD documentation
4. **Error Handling** - Integrates with `Hatchet::Error`
5. **Configuration Integration** - Connects with `Hatchet::Config`
6. **Sorbet Types** - Adds type signatures for better IDE support

## Integration with Main SDK

The REST client integrates seamlessly with the main SDK:

```ruby
# Main configuration
hatchet_config = Hatchet::Config.new(
  token: "your-jwt-token",
  server_url: "https://api.hatchet.com"
)

# REST client automatically inherits configuration
rest_client = Hatchet::Clients.rest_client(hatchet_config)

# All configuration options are passed through:
# - Authentication token
# - Server URL
# - Timeouts
# - Custom headers
```

## Error Handling

Generated API errors inherit from `Hatchet::Error`:

```ruby
begin
  workflows = workflows_api.list_workflows
rescue Hatchet::Clients::Rest::ApiError => e
  puts "API Error: #{e.message}"
  puts "Status: #{e.code}"
  puts "Headers: #{e.response_headers}"
  puts "Body: #{e.response_body}"
end
```

## Type Safety and IDE Support

The generated client includes:

- **YARD documentation** - Comprehensive method and parameter documentation
- **RBS type signatures** - For IDE parameter hints and type checking
- **Sorbet compatibility** - Type annotations for static analysis
- **Parameter validation** - Runtime validation of required parameters

## Customization

### Adding Custom Methods

You can extend the generated API classes:

```ruby
module Hatchet
  module Clients
    module Rest
      class WorkflowApi
        # Add custom convenience methods
        def get_workflow_by_name(name)
          workflows = list_workflows
          workflows.find { |w| w.name == name }
        end
      end
    end
  end
end
```

### Custom Configuration

Create specialized configurations:

```ruby
# Custom REST configuration
rest_config = Hatchet::Clients::Rest::Configuration.new
rest_config.access_token = "custom-token"
rest_config.host = "custom-host.com"
rest_config.timeout = 30

rest_client = Hatchet::Clients::Rest::ApiClient.new(rest_config)
```

## Development and Debugging

### Debugging Generation

Set the `DEBUG` environment variable for verbose output:

```bash
DEBUG=1 rake api:generate
```

### Validation

Always validate the OpenAPI spec before generation:

```bash
rake api:validate
```

### Regeneration

When the OpenAPI spec changes, regenerate the client:

```bash
rake api:regenerate
```

This will clean existing files and generate fresh ones.

## Troubleshooting

### Common Issues

1. **OpenAPI Generator CLI not found**
   ```bash
   rake api:install_generator
   ```

2. **Module loading errors**
   - Ensure you've run `bundle install` after generation
   - Check that all required gems are installed

3. **Authentication failures**
   - Verify your JWT token is valid
   - Check server URL configuration

4. **Type errors with Sorbet**
   - Run `bundle exec srb tc` to check types
   - Update RBS signatures if needed

### Getting Help

- Check `rake api:info` for OpenAPI spec information
- Review generated documentation in the `api/` and `models/` directories
- Consult the main Hatchet API documentation at https://docs.hatchet.run

## Maintenance

The REST API generation setup requires minimal maintenance:

1. **OpenAPI Spec Updates** - Regenerate when the spec changes
2. **Dependency Updates** - Keep OpenAPI Generator CLI updated
3. **Custom Patches** - Update patches when generation patterns change

The generation process is designed to be idempotent and safe to run multiple times.
