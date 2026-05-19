# frozen_string_literal: true

RSpec.describe Hatchet do
  let(:valid_token) { "eyJhbGciOiJIUzI1NiJ9.test_token" }

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

  it "has a version number" do
    expect(Hatchet::VERSION).not_to be nil
  end

  describe "Error class" do
    it "inherits from StandardError" do
      expect(Hatchet::Error.new).to be_a(StandardError)
    end

    it "can be raised with a message" do
      expect { raise Hatchet::Error, "test error" }.to raise_error(Hatchet::Error, "test error")
    end
  end

  describe "Client" do
    describe "#initialize" do
      it "creates a client with valid token" do
        client = Hatchet::Client.new(token: valid_token)
        expect(client).not_to be nil
        expect(client.config).to be_a(Hatchet::Config)
        expect(client.config.token).to eq(valid_token)
      end

      it "creates a client with custom configuration" do
        client = Hatchet::Client.new(
          token: valid_token,
          host_port: "custom.example.com:8080",
          namespace: "custom_namespace",
        )
        expect(client.config.host_port).to eq("custom.example.com:8080")
        expect(client.config.namespace).to eq("custom_namespace_")
      end

      it "raises error for invalid token" do
        expect { Hatchet::Client.new(token: "invalid") }.to raise_error(Hatchet::Error, /Token must be a valid JWT/)
      end

      it "raises error when no token provided" do
        expect do
          Hatchet::Client.new
        end.to raise_error(Hatchet::Error, "Hatchet Token is required. Please set HATCHET_CLIENT_TOKEN in your environment.")
      end
    end

    describe "#config" do
      it "exposes the configuration object" do
        client = Hatchet::Client.new(token: valid_token)
        expect(client.config).to be_a(Hatchet::Config)
        expect(client.config.token).to eq(valid_token)
      end

      it "allows access to all configuration properties" do
        client = Hatchet::Client.new(
          token: valid_token,
          host_port: "test.example.com:9090",
          server_url: "https://test.example.com",
          namespace: "test_ns",
        )

        expect(client.config.host_port).to eq("test.example.com:9090")
        expect(client.config.server_url).to eq("https://test.example.com")
        expect(client.config.namespace).to eq("test_ns_")
      end
    end

    context "with environment variables" do
      before do
        ENV["HATCHET_CLIENT_TOKEN"] = valid_token
        ENV["HATCHET_CLIENT_HOST_PORT"] = "env.example.com:7777"
      end

      after do
        ENV.delete("HATCHET_CLIENT_TOKEN")
        ENV.delete("HATCHET_CLIENT_HOST_PORT")
      end

      it "creates client from environment variables" do
        client = Hatchet::Client.new
        expect(client.config.token).to eq(valid_token)
        expect(client.config.host_port).to eq("env.example.com:7777")
      end

      it "allows explicit options to override environment variables" do
        client = Hatchet::Client.new(host_port: "explicit.example.com:8888")
        expect(client.config.token).to eq(valid_token)
        expect(client.config.host_port).to eq("explicit.example.com:8888")
      end
    end
  end
end
