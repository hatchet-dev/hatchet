# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is the Ruby SDK for Hatchet, currently in early development (version 0.0.0). The project follows standard Ruby gem conventions with a simple structure containing a main `Hatchet` module with a `Client` class for API key authentication.

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

## Architecture

**Core Structure:**
- `lib/hatchet.rb` - Main entry point, defines `Hatchet` module with `Error` and `Client` classes
- `lib/hatchet/version.rb` - Version constant
- `spec/` - RSpec tests with monkey patching disabled
- `sig/hatchet.rbs` - Ruby type signatures

**Key Classes:**
- `Hatchet::Client` - Main client class that accepts an API key for authentication
- `Hatchet::Error` - Base error class for gem-specific exceptions

The codebase uses frozen string literals and follows Ruby 3.1+ requirements.

Keep the CLAUDE.md instructions up to date as the project continues to develop.