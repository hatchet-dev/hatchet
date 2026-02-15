# frozen_string_literal: true

require "base64"

RSpec.describe Hatchet::Config do
  let(:valid_token) { "eyJhbGciOiJIUzI1NiJ9.valid_token" }
  let(:invalid_token) { "invalid_token" }
  let(:token_with_tenant_id) do
    # JWT with payload: {"sub": "jwt-tenant-123"}
    header = "eyJhbGciOiJIUzI1NiJ9" # {"alg":"HS256"}
    payload = Base64.encode64('{"sub":"jwt-tenant-123"}').gsub("\n", "").gsub(/=+$/, "")
    signature = "fake-signature"
    "#{header}.#{payload}.#{signature}"
  end

  around do |example|
    # Store original env vars
    original_env = ENV.select { |k, _| k.start_with?("HATCHET_CLIENT_") }

    # Clear all HATCHET_CLIENT_ env vars before each test
    ENV.keys.select { |k| k.start_with?("HATCHET_CLIENT_") }.each { |k| ENV.delete(k) }

    example.run

    # Restore original env vars
    ENV.keys.select { |k| k.start_with?("HATCHET_CLIENT_") }.each { |k| ENV.delete(k) }
    original_env.each { |k, v| ENV[k] = v }
  end

  describe "#initialize" do
    context "with valid token" do
      it "creates config with token" do
        config = described_class.new(token: valid_token)
        expect(config.token).to eq(valid_token)
      end

      it "sets default values" do
        config = described_class.new(token: valid_token)
        expect(config.host_port).to eq("localhost:7070")
        expect(config.server_url).to eq("https://app.dev.hatchet-tools.com")
        expect(config.namespace).to eq("")
        expect(config.tenant_id).to eq("")
        expect(config.grpc_max_recv_message_length).to eq(4 * 1024 * 1024)
        expect(config.grpc_max_send_message_length).to eq(4 * 1024 * 1024)
        expect(config.worker_preset_labels).to eq({})
        expect(config.listener_v2_timeout).to be_nil
      end

      it "accepts custom options" do
        config = described_class.new(
          token: valid_token,
          host_port: "custom.example.com:8080",
          server_url: "https://custom.example.com",
          namespace: "test_namespace",
          grpc_max_recv_message_length: 8 * 1024 * 1024,
          worker_preset_labels: { "env" => "test" },
          listener_v2_timeout: 5000,
        )

        expect(config.host_port).to eq("custom.example.com:8080")
        expect(config.server_url).to eq("https://custom.example.com")
        expect(config.namespace).to eq("test_namespace_")
        expect(config.grpc_max_recv_message_length).to eq(8 * 1024 * 1024)
        expect(config.worker_preset_labels).to eq({ "env" => "test" })
        expect(config.listener_v2_timeout).to eq(5000)
      end

      it "extracts tenant_id from JWT token payload when not explicitly set" do
        config = described_class.new(token: token_with_tenant_id)
        expect(config.tenant_id).to eq("jwt-tenant-123")
      end
    end

    context "with invalid token" do
      it "raises error for empty token" do
        expect do
          described_class.new(token: "")
        end.to raise_error(Hatchet::Error,
                           "Hatchet Token is required. Please set HATCHET_CLIENT_TOKEN in your environment.",)
      end

      it "raises error for invalid JWT token" do
        expect { described_class.new(token: invalid_token) }.to raise_error(
          Hatchet::Error, /Hatchet Token must be a valid JWT/,
        )
      end
    end

    context "with no token" do
      it "raises error when no token provided" do
        expect do
          described_class.new
        end.to raise_error(Hatchet::Error,
                           "Hatchet Token is required. Please set HATCHET_CLIENT_TOKEN in your environment.",)
      end
    end
  end

  describe "#apply_namespace" do
    let(:config) { described_class.new(token: valid_token, namespace: "test_ns") }

    it "applies namespace to resource name" do
      expect(config.apply_namespace("workflow")).to eq("test_ns_workflow")
    end

    it "does not duplicate namespace if already present" do
      expect(config.apply_namespace("test_ns_workflow")).to eq("test_ns_workflow")
    end

    it "returns nil for nil resource name" do
      expect(config.apply_namespace(nil)).to be_nil
    end

    it "returns resource name unchanged when namespace is empty" do
      config_no_ns = described_class.new(token: valid_token, namespace: "")
      expect(config_no_ns.apply_namespace("workflow")).to eq("workflow")
    end

    it "accepts namespace override" do
      expect(config.apply_namespace("workflow", namespace_override: "override_")).to eq("override_workflow")
    end
  end

  describe "namespace normalization" do
    it "converts namespace to lowercase" do
      config = described_class.new(token: valid_token, namespace: "TEST_NAMESPACE")
      expect(config.namespace).to eq("test_namespace_")
    end

    it "adds underscore to namespace if missing" do
      config = described_class.new(token: valid_token, namespace: "test")
      expect(config.namespace).to eq("test_")
    end

    it "does not add underscore if already present" do
      config = described_class.new(token: valid_token, namespace: "test_")
      expect(config.namespace).to eq("test_")
    end
  end

  describe "environment variable loading" do
    before do
      # Clear any existing env vars
      %w[
        HATCHET_CLIENT_TOKEN
        HATCHET_CLIENT_HOST_PORT
        HATCHET_CLIENT_SERVER_URL
        HATCHET_CLIENT_NAMESPACE
      ].each { |var| ENV.delete(var) }
    end

    after do
      # Clean up env vars
      %w[
        HATCHET_CLIENT_TOKEN
        HATCHET_CLIENT_HOST_PORT
        HATCHET_CLIENT_SERVER_URL
        HATCHET_CLIENT_NAMESPACE
      ].each { |var| ENV.delete(var) }
    end

    it "loads token from environment variable" do
      ENV["HATCHET_CLIENT_TOKEN"] = valid_token
      config = described_class.new
      expect(config.token).to eq(valid_token)
    end

    it "loads configuration from environment variables" do
      ENV["HATCHET_CLIENT_TOKEN"] = valid_token
      ENV["HATCHET_CLIENT_HOST_PORT"] = "env.example.com:9090"
      ENV["HATCHET_CLIENT_SERVER_URL"] = "https://env.example.com"
      ENV["HATCHET_CLIENT_NAMESPACE"] = "env_namespace"

      config = described_class.new
      expect(config.token).to eq(valid_token)
      expect(config.host_port).to eq("env.example.com:9090")
      expect(config.server_url).to eq("https://env.example.com")
      expect(config.namespace).to eq("env_namespace_")
      expect(config.tenant_id).to eq("") # tenant_id only comes from JWT token
    end

    it "prefers explicit options over environment variables" do
      ENV["HATCHET_CLIENT_TOKEN"] = valid_token
      ENV["HATCHET_CLIENT_HOST_PORT"] = "env.example.com:9090"

      config = described_class.new(host_port: "explicit.example.com:8080")
      expect(config.token).to eq(valid_token)
      expect(config.host_port).to eq("explicit.example.com:8080")
    end

    it "ignores environment variable tenant_id and uses JWT token payload" do
      ENV["HATCHET_CLIENT_TOKEN"] = token_with_tenant_id
      ENV["HATCHET_CLIENT_TENANT_ID"] = "env-tenant"

      config = described_class.new
      expect(config.tenant_id).to eq("jwt-tenant-123") # tenant_id only comes from JWT token
    end

    it "uses JWT token tenant_id when environment variable is not set" do
      ENV["HATCHET_CLIENT_TOKEN"] = token_with_tenant_id

      config = described_class.new
      expect(config.tenant_id).to eq("jwt-tenant-123")
    end
  end

  describe "type parsing" do
    let(:config) { described_class.new(token: valid_token) }

    describe "#parse_int" do
      it "parses valid integer strings" do
        expect(config.send(:parse_int, "123")).to eq(123)
        expect(config.send(:parse_int, "0")).to eq(0)
      end

      it "returns nil for invalid strings" do
        expect(config.send(:parse_int, "invalid")).to be_nil
        expect(config.send(:parse_int, "")).to be_nil
        expect(config.send(:parse_int, nil)).to be_nil
      end
    end


  end

  describe "logger" do
    it "has a default logger" do
      config = described_class.new(token: valid_token)
      expect(config.logger).to be_a(Logger)
    end

    it "accepts custom logger" do
      custom_logger = Logger.new(StringIO.new)
      config = described_class.new(token: valid_token, logger: custom_logger)
      expect(config.logger).to eq(custom_logger)
    end
  end

  describe ".env file loading" do
    let(:temp_env_file) { ".env.test" }

    before do
      # Stub ENV_FILE_NAMES to include our test file
      stub_const("#{described_class}::ENV_FILE_NAMES", [temp_env_file])
    end

    after do
      FileUtils.rm_f(temp_env_file)
      ENV.delete("TEST_VAR")
    end

    it "loads variables from .env file" do
      File.write(temp_env_file, "TEST_VAR=test_value\nHATCHET_CLIENT_TOKEN=#{valid_token}")

      config = described_class.new
      expect(ENV.fetch("TEST_VAR", nil)).to eq("test_value")
      expect(config.token).to eq(valid_token)
    end

    it "handles quoted values" do
      File.write(temp_env_file, "TEST_VAR=\"quoted_value\"\nHATCHET_CLIENT_TOKEN='#{valid_token}'")

      config = described_class.new
      expect(ENV.fetch("TEST_VAR", nil)).to eq("quoted_value")
      expect(config.token).to eq(valid_token)
    end

    it "ignores comments and empty lines" do
      File.write(temp_env_file, <<~ENV_FILE)
        # This is a comment

        HATCHET_CLIENT_TOKEN=#{valid_token}
        # Another comment
        TEST_VAR=value
      ENV_FILE

      config = described_class.new
      expect(config.token).to eq(valid_token)
      expect(ENV.fetch("TEST_VAR", nil)).to eq("value")
    end

    it "does not override existing environment variables" do
      ENV["TEST_VAR"] = "existing_value"
      File.write(temp_env_file, "TEST_VAR=file_value\nHATCHET_CLIENT_TOKEN=#{valid_token}")

      described_class.new
      expect(ENV.fetch("TEST_VAR", nil)).to eq("existing_value")
    end
  end
end
