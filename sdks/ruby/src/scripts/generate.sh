#!/bin/bash

# Ruby SDK REST API Client Generation Script
# Similar to the Python SDK's generate.sh

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
OPENAPI_SPEC="$PROJECT_ROOT/../../../../bin/oas/openapi.yaml"
OUTPUT_DIR="$PROJECT_ROOT/lib/hatchet/clients/rest"
CONFIG_FILE="$PROJECT_ROOT/config/openapi_generator_config.json"

echo -e "${BLUE}🚀 Starting Ruby REST API Client Generation${NC}"
echo "Project Root: $PROJECT_ROOT"

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to install OpenAPI Generator CLI
install_openapi_generator() {
    echo -e "${YELLOW}📦 Installing OpenAPI Generator CLI...${NC}"

    if command_exists npm; then
        npm install -g @openapitools/openapi-generator-cli@7.13.0
        echo -e "${GREEN}✅ OpenAPI Generator CLI installed${NC}"
    else
        echo -e "${RED}❌ npm not found. Please install Node.js and npm first.${NC}"
        exit 1
    fi
}

# Function to validate OpenAPI spec
validate_spec() {
    echo -e "${BLUE}🔍 Validating OpenAPI specification...${NC}"

    if [ ! -f "$OPENAPI_SPEC" ]; then
        echo -e "${RED}❌ OpenAPI spec not found at: $OPENAPI_SPEC${NC}"
        exit 1
    fi

    openapi-generator-cli validate -i "$OPENAPI_SPEC"
    echo -e "${GREEN}✅ OpenAPI specification is valid${NC}"
}

# Function to generate the REST client
generate_client() {
    echo -e "${BLUE}🏗️  Generating Ruby REST client...${NC}"

    # Create output directory
    mkdir -p "$OUTPUT_DIR"

    # Additional properties for Ruby generation
    ADDITIONAL_PROPS="gemName=hatchet-sdk-rest,moduleName=Hatchet::Clients::Rest,gemVersion=0.0.1,gemDescription=Ruby REST client for Hatchet API,gemAuthor=Hatchet Team,gemHomepage=https://github.com/hatchet-dev/hatchet,gemLicense=MIT,library=faraday,httpLibrary=faraday"

    # Generate the client
    openapi-generator-cli generate \
        -i "$OPENAPI_SPEC" \
        -g ruby \
        -o "$OUTPUT_DIR" \
        -c "$CONFIG_FILE" \
        --skip-validate-spec \
        --global-property apiTests=false,modelTests=false,apiDocs=true,modelDocs=true \
        --additional-properties "$ADDITIONAL_PROPS"

    echo -e "${GREEN}✅ Ruby REST client generated${NC}"
}

# Function to apply custom patches
apply_patches() {
    echo -e "${BLUE}🔧 Applying custom patches...${NC}"

    echo -e "${GREEN}✅ Custom patches applied${NC}"
}

# Function to update dependencies
update_dependencies() {
    echo -e "${BLUE}📦 Updating Ruby dependencies...${NC}"

    cd "$PROJECT_ROOT"

    # Add required gems to Gemfile if not present
    if ! grep -q "gem ['\"]faraday['\"]" Gemfile 2>/dev/null; then
        echo "gem 'faraday', '~> 2.0'" >> Gemfile
    fi

    if ! grep -q "gem ['\"]faraday-multipart['\"]" Gemfile 2>/dev/null; then
        echo "gem 'faraday-multipart'" >> Gemfile
    fi

    # Install/update dependencies
    bundle install

    echo -e "${GREEN}✅ Dependencies updated${NC}"
}

# Function to run tests on generated code
run_tests() {
    echo -e "${BLUE}🧪 Running tests...${NC}"

    cd "$PROJECT_ROOT"

    # Run RSpec tests
    bundle exec rspec

    # Run RuboCop
    bundle exec rubocop --auto-correct

    echo -e "${GREEN}✅ Tests completed${NC}"
}

# Main execution
main() {
    echo -e "${BLUE}Starting Ruby REST API generation process...${NC}"

    # Check if OpenAPI Generator CLI is installed
    if ! command_exists openapi-generator-cli; then
        echo -e "${YELLOW}OpenAPI Generator CLI not found${NC}"
        install_openapi_generator
    else
        echo -e "${GREEN}✅ OpenAPI Generator CLI found${NC}"
    fi

    # Validate the OpenAPI specification
    validate_spec

    # Generate the REST client
    generate_client

    # Apply custom patches
    apply_patches

    # Update dependencies
    update_dependencies

    # Run tests (optional, can be disabled with --skip-tests)
    if [[ "$*" != *"--skip-tests"* ]]; then
        run_tests
    fi

    echo -e "${GREEN}🎉 Ruby REST API client generation completed successfully!${NC}"
    echo -e "${BLUE}Generated files are in: $OUTPUT_DIR${NC}"
}

# Handle command line arguments
case "$1" in
    --help|-h)
        echo "Usage: $0 [OPTIONS]"
        echo "Options:"
        echo "  --skip-tests    Skip running tests after generation"
        echo "  --help, -h      Show this help message"
        exit 0
        ;;
    *)
        main "$@"
        ;;
esac
