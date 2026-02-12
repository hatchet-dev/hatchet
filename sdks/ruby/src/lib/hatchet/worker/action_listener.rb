# frozen_string_literal: true

module Hatchet
  module WorkerRuntime
    # Listens for action assignments from the Hatchet dispatcher via gRPC streaming.
    #
    # The action listener establishes a server-streaming gRPC connection (ListenV2),
    # receives action assignments (task executions), and forwards them to the runner
    # for execution.
    #
    # Includes retry/reconnect logic with exponential backoff and a heartbeat thread.
    #
    # @example
    #   listener = ActionListener.new(dispatcher_client: client, worker_id: id, logger: logger)
    #   listener.start do |action|
    #     runner.execute(action)
    #   end
    class ActionListener
      MAX_RETRIES = 15
      BASE_BACKOFF_SECONDS = 1
      MAX_BACKOFF_SECONDS = 30
      HEALTHY_CONNECTION_THRESHOLD = 30 # seconds
      HEARTBEAT_INTERVAL = 4 # seconds
      MAX_MISSED_HEARTBEATS = 3

      # @param dispatcher_client [Hatchet::Clients::Grpc::Dispatcher] The gRPC dispatcher client
      # @param worker_id [String] The registered worker ID
      # @param logger [Logger] Logger instance
      def initialize(dispatcher_client:, worker_id:, logger:)
        @dispatcher_client = dispatcher_client
        @worker_id = worker_id
        @logger = logger
        @running = false
        @heartbeat_thread = nil
        @missed_heartbeats = 0
      end

      # Start listening for actions. Blocks until stopped.
      #
      # Implements retry logic with exponential backoff:
      # - Max 15 retries
      # - Reset retry counter if connection was alive > 30 seconds
      # - Handles GRPC::Unavailable, GRPC::DeadlineExceeded, and EOF
      #
      # @yield [action] Called for each action received from the dispatcher
      def start(&block)
        @running = true
        retries = 0

        start_heartbeat_thread

        while @running && retries < MAX_RETRIES
          connection_start = Time.now

          begin
            @logger.info("Action listener connecting for worker #{@worker_id} (attempt #{retries + 1})")

            # Get the server-streaming response (ListenV2)
            stream = @dispatcher_client.listen(worker_id: @worker_id)

            # Iterate over the stream, yielding each AssignedAction
            stream.each do |action|
              break unless @running

              # Reset retries on successful message receipt
              if Time.now - connection_start > HEALTHY_CONNECTION_THRESHOLD
                retries = 0
                connection_start = Time.now
              end

              block.call(action)
            end

            # Stream ended normally (server closed)
            @logger.info("Action listener stream ended normally")
          rescue ::GRPC::Unavailable => e
            @logger.warn("gRPC unavailable: #{e.message}")
          rescue ::GRPC::DeadlineExceeded => e
            @logger.warn("gRPC deadline exceeded: #{e.message}")
          rescue ::GRPC::Cancelled => e
            @logger.info("gRPC stream cancelled: #{e.message}")
            break unless @running
          rescue ::GRPC::Unknown => e
            @logger.warn("gRPC unknown error: #{e.message}")
          rescue StopIteration
            @logger.info("Action listener stream ended (StopIteration)")
          rescue Interrupt
            @logger.info("Action listener interrupted")
            @running = false
            break
          rescue => e
            @logger.warn("Action listener error: #{e.class}: #{e.message}")
          end

          break unless @running

          # Check if connection was healthy long enough to reset retries
          connection_duration = Time.now - connection_start
          if connection_duration > HEALTHY_CONNECTION_THRESHOLD
            retries = 0
          else
            retries += 1
          end

          if retries >= MAX_RETRIES
            @logger.error("Action listener exhausted #{MAX_RETRIES} retries. Giving up.")
            break
          end

          # Exponential backoff
          backoff = [BASE_BACKOFF_SECONDS * (2**(retries - 1)), MAX_BACKOFF_SECONDS].min
          @logger.info("Reconnecting in #{backoff}s (retry #{retries}/#{MAX_RETRIES})")
          sleep(backoff)
        end

        stop_heartbeat_thread
        @logger.info("Action listener stopped for worker #{@worker_id}")
      end

      # Stop listening for actions.
      def stop
        @running = false
        stop_heartbeat_thread
        @logger.info("Action listener stopping for worker #{@worker_id}")
      end

      private

      # Start a background thread that sends heartbeats to the dispatcher.
      def start_heartbeat_thread
        @missed_heartbeats = 0

        @heartbeat_thread = Thread.new do
          while @running
            begin
              @dispatcher_client.heartbeat(worker_id: @worker_id)
              @missed_heartbeats = 0
            rescue => e
              @missed_heartbeats += 1
              @logger.warn("Heartbeat failed (#{@missed_heartbeats}/#{MAX_MISSED_HEARTBEATS}): #{e.message}")

              if @missed_heartbeats >= MAX_MISSED_HEARTBEATS
                @logger.error("Too many missed heartbeats. Interrupting listener.")
                @running = false
                break
              end
            end

            sleep(HEARTBEAT_INTERVAL)
          end
        end

        @heartbeat_thread.abort_on_exception = false
      end

      # Stop the heartbeat thread.
      def stop_heartbeat_thread
        return unless @heartbeat_thread

        @heartbeat_thread.kill rescue nil
        @heartbeat_thread = nil
      end
    end
  end
end
