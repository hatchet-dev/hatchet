# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is the Ruby SDK for Hatchet (gem name: hatchet-sdk), currently in early development (version 0.0.0). The project follows standard Ruby gem conventions with a simple structure containing a main `Hatchet` module with a `Client` class for JWT token authentication.

The SDK includes comprehensive documentation and type signatures for excellent IDE support, including parameter hints, auto-completion, and type checking.

## Common Commands

**Testing:**
- `rake spec` or `bundle exec rspec` - Run all tests
- `rspec spec/hatchet_spec.rb` - Run specific test file

**Linting:**
- `rubocop` - Run Ruby linter
- `rubocop --safe-auto-correct` - Auto-fix safe lint issues

**Development:**
- `bin/setup` - Install dependencies after checkout
- `bin/console` - Start interactive console with gem loaded
- `bundle exec rake install` - Install gem locally
- `rake` - Run default task (spec + rubocop)

**Gem Management:**
- `bundle exec rake release` - Release new version (updates version, creates git tag, pushes to rubygems)

**Documentation:**
- `yard doc` - Generate YARD documentation (if yard gem is installed)
- `rbs validate` - Validate RBS type signatures for syntax errors

**REST API Generation:**
- `rake api:generate` - Generate REST API client from OpenAPI specification
- `rake api:clean` - Remove generated REST API client files
- `rake api:regenerate` - Clean and regenerate REST API client
- `rake api:validate` - Validate OpenAPI specification
- `rake api:info` - Show OpenAPI specification information
- `./scripts/generate.sh` - Alternative shell script for generation

## Architecture

**Core Structure:**
- `lib/hatchet-sdk.rb` - Main entry point, defines `Hatchet` module with `Error` and `Client` classes
- `lib/hatchet/version.rb` - Version constant
- `lib/hatchet/config.rb` - Configuration classes with comprehensive JWT token support
- `lib/hatchet/clients.rb` - Client factory and REST client integration
- `lib/hatchet/clients/rest/` - Generated REST API client (created via `rake api:generate`)
- `spec/` - RSpec tests with monkey patching disabled (36+ test cases)
- `sig/hatchet-sdk.rbs` - Ruby type signatures for IDE integration
- `scripts/` - Code generation and maintenance scripts

**Key Classes:**
- `Hatchet::Client` - Main client class that accepts JWT token for authentication
- `Hatchet::Config` - Configuration class supporting multiple sources (params, env vars, JWT payload)
- `Hatchet::Clients` - Factory for creating REST and other protocol clients
- `Hatchet::Clients::Rest::*` - Generated REST API clients (WorkflowApi, EventApi, etc.)
- `Hatchet::TLSConfig` - TLS configuration for secure connections
- `Hatchet::HealthcheckConfig` - Worker health monitoring configuration
- `Hatchet::Error` - Base error class for gem-specific exceptions

**Configuration Sources (priority order):**
1. Explicit constructor parameters (highest priority)
2. Environment variables (`HATCHET_CLIENT_*`)
3. JWT token payload (tenant_id extracted from 'sub' field)
4. Default values (lowest priority)

**Documentation & IDE Support:**
- **YARD documentation** - Comprehensive JSDoc-style comments with examples
- **RBS type signatures** - Complete type definitions for IDE parameter hints
- **Sorbet compatibility** - Tagged with `# typed: strict` for type checking
- IDEs with Ruby LSP/RubyMine will show parameter hints, auto-completion, and types

The codebase uses frozen string literals and follows Ruby 3.1+ requirements.

## Development Notes

**When adding new configuration options:**
1. Add the parameter to `Config#initialize` method
2. Update the `@option` YARD documentation in both `Client` and `Config` classes
3. Add the parameter to RBS type signatures in `sig/hatchet-sdk.rbs`
4. Add comprehensive test coverage in `spec/hatchet/config_spec.rb`
5. Update this CLAUDE.md file with any architectural changes

**REST API Client Generation:**
- The SDK can generate a complete REST API client from the OpenAPI spec at `../../../../bin/oas/openapi.yaml`
- Use `rake api:generate` to create Ruby bindings for all Hatchet REST endpoints
- Generated client integrates with `Hatchet::Config` for authentication and configuration
- Supports both sync and async operations with comprehensive error handling
- See `REST_API_GENERATION.md` for detailed instructions

**Testing JWT token functionality:**
- Use `Base64.encode64('{"sub":"tenant-id"}').gsub(/\n/, "").gsub(/=+$/, "")` to create test JWT payloads
- The config extracts tenant_id from the 'sub' field in JWT tokens
- Test both explicit tenant_id override and JWT extraction scenarios

Keep the CLAUDE.md instructions up to date as the project continues to develop.
