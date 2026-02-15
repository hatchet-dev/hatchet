# frozen_string_literal: true
# typed: strict

# Integration file for generated Hatchet REST API client
# This file loads the generated REST client and makes it available under the Hatchet::Clients::Rest namespace

begin
  # Add the generated client's lib/ directory to $LOAD_PATH so that
  # `require 'hatchet-sdk-rest/...'` calls in the entry point resolve correctly.
  rest_lib_path = File.expand_path("rest/lib", __dir__)
  $LOAD_PATH.unshift(rest_lib_path) unless $LOAD_PATH.include?(rest_lib_path)

  # Load the generated REST client
  require_relative "rest/lib/hatchet-sdk-rest"

  # The generated client creates classes under HatchetSdkRest module
  # We need to alias them to our desired namespace structure
  module Hatchet
    module Clients
      module Rest
        # Re-export the main classes from the generated client
        ApiClient = ::HatchetSdkRest::ApiClient
        ApiError = ::HatchetSdkRest::ApiError

        # Enhanced Configuration class with Hatchet integration
        class Configuration < ::HatchetSdkRest::Configuration
          # Create a Configuration instance from a Hatchet::Config
          #
          # @param hatchet_config [Hatchet::Config] The main Hatchet configuration
          # @return [Configuration] Configured REST client configuration
          def self.from_hatchet_config(hatchet_config)
            config = new
            config.access_token = hatchet_config.token

            # Extract host from server_url
            if hatchet_config.server_url && !hatchet_config.server_url.empty?
              config.host = hatchet_config.server_url.gsub(%r{^https?://}, "").split("/").first
              config.scheme = hatchet_config.server_url.start_with?("https") ? "https" : "http"
            end

            # Set timeout if available
            if hatchet_config.listener_v2_timeout
              config.timeout = hatchet_config.listener_v2_timeout / 1000.0 # Convert ms to seconds
            end

            # Use FlatParamsEncoder so array query parameters are serialized as
            # key=val1&key=val2 instead of key[]=val1&key[]=val2, matching what
            # the Hatchet API server expects.
            config.params_encoder = Faraday::FlatParamsEncoder

            config
          end
        end

        # Re-export API classes
        WorkflowApi = ::HatchetSdkRest::WorkflowApi
        EventApi = ::HatchetSdkRest::EventApi
        StepRunApi = ::HatchetSdkRest::StepRunApi
        WorkflowRunApi = ::HatchetSdkRest::WorkflowRunApi
        WorkflowRunsApi = ::HatchetSdkRest::WorkflowRunsApi
        TenantApi = ::HatchetSdkRest::TenantApi
        UserApi = ::HatchetSdkRest::UserApi
        WorkerApi = ::HatchetSdkRest::WorkerApi

        # Re-export commonly used model classes
        CreateEventRequest = ::HatchetSdkRest::CreateEventRequest
        Event = ::HatchetSdkRest::Event

        # Add more API classes and models as needed - you can extend this list
        # with any other generated API classes or models you want to expose
      end
    end
  end
rescue LoadError => e
  # If the generated client files are not available, define an empty module
  # This allows the main SDK to load without errors even before generation
  warn "REST client not fully loaded: #{e.message}" if ENV["DEBUG"]

  module Hatchet
    module Clients
      module Rest
        # Placeholder classes that will raise helpful errors
        class ApiClient
          def initialize(*)
            raise LoadError, "REST client not generated. Run `rake api:generate` to generate it."
          end
        end

        class Configuration
          def self.from_hatchet_config(*)
            raise LoadError, "REST client not generated. Run `rake api:generate` to generate it."
          end
        end

        # Placeholder model classes
        class CreateEventRequest
          def initialize(*)
            raise LoadError, "REST client not generated. Run `rake api:generate` to generate it."
          end
        end

        class Event
          def initialize(*)
            raise LoadError, "REST client not generated. Run `rake api:generate` to generate it."
          end
        end
      end
    end
  end
end
