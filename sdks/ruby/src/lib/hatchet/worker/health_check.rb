# frozen_string_literal: true

require "socket"

module Hatchet
  module WorkerRuntime
    # Simple HTTP health check server using raw TCPServer.
    #
    # WEBrick was removed from Ruby's default gems in Ruby 3.4+, so we use
    # a raw TCPServer with minimal HTTP response to avoid adding a dependency.
    #
    # @example
    #   health = HealthCheck.new(port: 8001, logger: logger)
    #   health.start  # runs in a background thread
    #   health.stop
    class HealthCheck
      # @param port [Integer] Port to listen on
      # @param logger [Logger] Logger instance
      def initialize(port:, logger:)
        @port = port
        @logger = logger
        @running = false
        @thread = nil
        @server = nil
      end

      # Start the health check server in a background thread
      def start
        @running = true

        @thread = Thread.new do
          @server = TCPServer.new("0.0.0.0", @port)
          @logger.info("Health check server listening on port #{@port}")

          loop do
            break unless @running

            begin
              client = @server.accept_nonblock
              request = client.gets # Read the request line

              response_body = "OK"
              response = "HTTP/1.1 200 OK\r\n" \
                         "Content-Type: text/plain\r\n" \
                         "Content-Length: #{response_body.length}\r\n" \
                         "Connection: close\r\n" \
                         "\r\n" \
                         "#{response_body}"

              client.print response
              client.close
            rescue IO::WaitReadable
              IO.select([@server], nil, nil, 1)
              retry unless !@running
            rescue => e
              @logger.debug("Health check error: #{e.message}") if @running
            end
          end
        rescue => e
          @logger.error("Health check server error: #{e.message}")
        ensure
          @server&.close rescue nil
        end
      end

      # Stop the health check server
      def stop
        @running = false
        @server&.close rescue nil
        @thread&.join(5)
      end

      # Check if the server is running
      # @return [Boolean]
      def running?
        @running && @thread&.alive?
      end
    end
  end
end
