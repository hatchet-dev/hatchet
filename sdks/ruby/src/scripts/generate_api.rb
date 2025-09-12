#!/usr/bin/env ruby
# frozen_string_literal: true

# Script to generate Ruby REST API client from OpenAPI specification
# Similar to the Python SDK's generate.sh script

require "fileutils"
require "json"
require "open3"

class ApiGenerator
  OPENAPI_SPEC_PATH = "../../../../bin/oas/openapi.yaml"
  GENERATOR_CONFIG_PATH = "../config/openapi_generator_config.json"
  OUTPUT_DIR = "../lib/hatchet/clients/rest"
  TEMP_DIR = "../tmp/openapi_generation"

  def initialize
    @root_dir = __dir__
    @patches_applied = 0
  end

  def generate
    puts "🚀 Starting Ruby REST API client generation..."

    setup_directories
    install_openapi_generator if needed
    generate_client_code
    apply_custom_patches
    cleanup_temp_files

    puts "✅ Ruby REST API client generation completed successfully!"
  end

  private

  def setup_directories
    puts "📁 Setting up directories..."
    FileUtils.mkdir_p(OUTPUT_DIR)
    FileUtils.mkdir_p(TEMP_DIR)
  end

  def install_openapi_generator
    puts "🔧 Checking OpenAPI Generator installation..."

    # Check if openapi-generator-cli is available
    stdout, stderr, status = Open3.capture3("which openapi-generator-cli")

    if status.success?
      puts "✅ OpenAPI Generator CLI found"
    else
      puts "📦 Installing OpenAPI Generator CLI via npm..."
      system("npm install -g @openapitools/openapi-generator-cli@7.13.0") ||
        raise("Failed to install OpenAPI Generator CLI")
    end
  end

  def generate_client_code
    puts "🏗️  Generating Ruby client code from OpenAPI spec..."

    openapi_spec = File.expand_path(OPENAPI_SPEC_PATH, @root_dir)
    config_file = File.expand_path(GENERATOR_CONFIG_PATH, @root_dir)
    output_path = File.expand_path(OUTPUT_DIR, @root_dir)

    unless File.exist?(openapi_spec)
      raise "OpenAPI spec not found at #{openapi_spec}"
    end

    unless File.exist?(config_file)
      puts "⚠️  Config file not found at #{config_file}, using default configuration"
      config_file = nil
    else
      puts "✅ Using config file: #{config_file}"
    end

    # Generate Ruby client using OpenAPI Generator
    additional_props = build_additional_properties

    cmd = [
      "openapi-generator-cli", "generate",
      "-i", openapi_spec,
      "-g", "ruby",
      "-o", output_path,
      "--skip-validate-spec",
      "--global-property", "apiTests=false,modelTests=false,apiDocs=true,modelDocs=true",
      "--additional-properties", additional_props
    ]

    cmd += ["-c", config_file] if config_file

    puts "Running: #{cmd.join(' ')}"
    puts "Additional properties: #{additional_props}"
    system(*cmd) || raise("Failed to generate Ruby client code")
  end

  def build_additional_properties
    [
      "gemName=hatchet-sdk-rest",
      "moduleName=HatchetSdkRest",
      "gemVersion=0.0.1",
      "gemDescription=HatchetRubySDKRestClient",
      "gemAuthor=HatchetTeam",
      "gemHomepage=https://github.com/hatchet-dev/hatchet",
      "gemLicense=MIT",
      "library=faraday"
    ].join(",")
  end

  def apply_custom_patches
    puts "🔧 Applying custom patches..."

    # Apply Ruby-specific patches here
    patch_require_statements
    patch_cookie_auth
  end

  def patch_require_statements
    puts "  📝 Patching require statements..."

    # Find all generated Ruby files and fix require statements
    Dir.glob("#{OUTPUT_DIR}/**/*.rb").each do |file|
      content = File.read(file)

      # Update require paths to be relative to the gem structure
      content.gsub!(/require ['"]hatchet-sdk-rest\//, "require 'hatchet/clients/rest/")

      # Ensure consistent module naming
      content.gsub!(/module HatchetSdkRest/, "module Hatchet\n  module Clients\n    module Rest")
      content.gsub!(/^end$/, "    end\n  end\nend") if content.include?("module Hatchet")

      File.write(file, content)
    end
  end

  def patch_cookie_auth
    puts "  🍪 Enhancing cookie auth..."
    # find the auth_settings hash in the configuration.rb file
    output_path = File.expand_path(OUTPUT_DIR, @root_dir)
    config_file = File.join(output_path, "lib/hatchet-sdk-rest/configuration.rb")
    if not File.exist?(config_file)
      puts "config_file does not exist"
      return false
    end
    content = File.read(config_file)

    # Apply the fix - replace 'in: ,' with 'in: 'header','
    if content.gsub!(/in:\s*,/, "in: 'header',")
      puts "    ✅ Successfully applied cookie auth patch"
      File.write(config_file, content)
      @patches_applied += 1
    else
      puts "    ❌ Failed to apply cookie auth patch - no matches found"
    end
  end

  def cleanup_temp_files
    puts "🧹 Cleaning up temporary files..."
    FileUtils.rm_rf(TEMP_DIR) if Dir.exist?(TEMP_DIR)
  end

  def needed
    !system("which openapi-generator-cli > /dev/null 2>&1")
  end
end

# Run the generator if this script is executed directly
if __FILE__ == $0
  begin
    ApiGenerator.new.generate
  rescue => e
    puts "❌ Generation failed: #{e.message}"
    puts e.backtrace.join("\n") if ENV["DEBUG"]
    exit 1
  end
end
