#!/usr/bin/env ruby
# frozen_string_literal: true

# Custom patches for generated Ruby REST API client
# Similar to the Python SDK's apply_patches.py

require "fileutils"

class CodePatcher
  def initialize(output_dir)
    @output_dir = output_dir
    @patches_applied = 0
  end

  def apply_all_patches
    puts "🔧 Applying custom patches to generated Ruby code..."
    
    apply_module_structure_patches
    apply_require_statement_patches
    apply_documentation_patches
    apply_configuration_patches
    apply_error_handling_patches
    add_yard_documentation
    add_sorbet_signatures
    
    puts "✅ Applied #{@patches_applied} patches successfully"
  end

  private

  def apply_module_structure_patches
    puts "  📁 Fixing module structure..."
    
    # Find all Ruby files in the generated code
    Dir.glob("#{@output_dir}/**/*.rb").each do |file|
      content = File.read(file)
      modified = false
      
      # Fix module declarations to match our desired structure
      if content.gsub!(/^module HatchetSdkRest/, "module Hatchet\n  module Clients\n    module Rest")
        # Update corresponding end statements
        content.gsub!(/^end$/) do |match|
          if content.count("module") > content.count("end")
            "    end\n  end\nend"
          else
            match
          end
        end
        modified = true
      end
      
      # Fix class declarations within modules
      content.gsub!(/^class ([A-Z]\w+)/, "      class \\1")
      
      if modified
        File.write(file, content)
        @patches_applied += 1
      end
    end
  end

  def apply_require_statement_patches
    puts "  📝 Fixing require statements..."
    
    Dir.glob("#{@output_dir}/**/*.rb").each do |file|
      content = File.read(file)
      modified = false
      
      # Fix require statements to use proper gem structure
      if content.gsub!(/require ['"]hatchet-sdk-rest\//, "require 'hatchet/clients/rest/")
        modified = true
      end
      
      # Add proper frozen string literal comments
      unless content.start_with?("# frozen_string_literal: true")
        content = "# frozen_string_literal: true\n\n#{content}"
        modified = true
      end
      
      if modified
        File.write(file, content)
        @patches_applied += 1
      end
    end
  end

  def apply_documentation_patches
    puts "  📚 Enhancing documentation..."
    
    Dir.glob("#{@output_dir}/**/*.rb").each do |file|
      content = File.read(file)
      modified = false
      
      # Add gem description header to main files
      if file.end_with?("/api_client.rb") || file.end_with?("/configuration.rb")
        unless content.include?("# Hatchet Ruby SDK - REST API Client")
          header = <<~HEADER
            # frozen_string_literal: true
            
            # Hatchet Ruby SDK - REST API Client
            # Generated from OpenAPI specification
            # 
            # This file contains the main REST API client for the Hatchet workflow engine.
            # It provides Ruby bindings for all Hatchet REST API endpoints.
            #
            # @see https://docs.hatchet.run API Documentation
            
          HEADER
          
          content = content.sub(/^# frozen_string_literal: true\n\n/, header)
          modified = true
        end
      end
      
      if modified
        File.write(file, content)
        @patches_applied += 1
      end
    end
  end

  def apply_configuration_patches
    puts "  ⚙️  Enhancing configuration..."
    
    config_file = File.join(@output_dir, "configuration.rb")
    return unless File.exist?(config_file)
    
    content = File.read(config_file)
    
    # Add integration with our existing Config class
    integration_code = <<~INTEGRATION
      
      # Integration with Hatchet::Config
      # Allows using the main SDK configuration with the REST client
      def self.from_hatchet_config(hatchet_config)
        config = new
        config.access_token = hatchet_config.token
        config.host = hatchet_config.server_url
        config.timeout = hatchet_config.listener_v2_timeout if hatchet_config.listener_v2_timeout
        config
      end
    INTEGRATION
    
    # Add before the final 'end' of the Configuration class
    if content.gsub!(/^(\s*)end\s*$/) { |match| "#{integration_code}\n#{match}" }
      File.write(config_file, content)
      @patches_applied += 1
    end
  end

  def apply_error_handling_patches
    puts "  🚨 Enhancing error handling..."
    
    # Find and enhance exception classes
    Dir.glob("#{@output_dir}/**/api_error.rb").each do |file|
      content = File.read(file)
      
      # Add integration with our base Error class
      integration = <<~ERROR_INTEGRATION
        
        # Make API errors inherit from Hatchet::Error for consistency
        class ApiError < ::Hatchet::Error
          attr_reader :response_headers, :response_body
          
          def initialize(message = nil, response_headers: nil, response_body: nil)
            super(message)
            @response_headers = response_headers
            @response_body = response_body
          end
        end
      ERROR_INTEGRATION
      
      # Replace the default ApiError class
      if content.gsub!(/class ApiError.*?^end/m, integration.strip)
        File.write(file, content)
        @patches_applied += 1
      end
    end
  end

  def add_yard_documentation
    puts "  📖 Adding YARD documentation..."
    
    # Add YARD tags to main API classes
    Dir.glob("#{@output_dir}/api/*.rb").each do |file|
      content = File.read(file)
      class_name = File.basename(file, ".rb").split("_").map(&:capitalize).join
      
      yard_header = <<~YARD
        
        # #{class_name} API client
        #
        # Provides access to #{class_name.downcase} related endpoints in the Hatchet API.
        # All methods in this class correspond to REST API endpoints documented at https://docs.hatchet.run
        #
        # @example Initialize and use the API
        #   config = Hatchet::Clients::Rest::Configuration.from_hatchet_config(hatchet_config)
        #   api_client = Hatchet::Clients::Rest::ApiClient.new(config)
        #   #{class_name.downcase}_api = Hatchet::Clients::Rest::#{class_name}Api.new(api_client)
        #
      YARD
      
      # Add YARD documentation before class declaration
      content.gsub!(/^(\s*)(class #{class_name}Api)/, "#{yard_header}\\1\\2")
      
      File.write(file, content)
      @patches_applied += 1
    end
  end

  def add_sorbet_signatures
    puts "  🏷️  Adding Sorbet type signatures..."
    
    Dir.glob("#{@output_dir}/**/*.rb").each do |file|
      content = File.read(file)
      
      # Add Sorbet typing header
      unless content.include?("# typed:")
        content = content.sub(/^# frozen_string_literal: true/, "# frozen_string_literal: true\n# typed: strict")
        
        File.write(file, content)
        @patches_applied += 1
      end
    end
  end
end

# Run the patcher if called directly
if __FILE__ == $0
  output_dir = ARGV[0]
  
  unless output_dir && Dir.exist?(output_dir)
    puts "❌ Usage: #{$0} <output_directory>"
    puts "   Output directory must exist and contain generated code"
    exit 1
  end
  
  begin
    patcher = CodePatcher.new(output_dir)
    patcher.apply_all_patches
  rescue => e
    puts "❌ Patching failed: #{e.message}"
    puts e.backtrace.join("\n") if ENV["DEBUG"]
    exit 1
  end
end