# frozen_string_literal: true

require "socket"

module Hatchet
  # Worker that processes tasks from the Hatchet engine.
  #
  # The worker connects to the Hatchet server via gRPC, registers its workflows,
  # listens for task assignments, and executes them in a thread pool.
  #
  # @example Start a worker
  #   worker = hatchet.worker("my-worker", workflows: [my_workflow])
  #   worker.start
  class Worker
    # @return [String] Worker name
    attr_reader :name

    # @return [Array<Workflow, Task>] Registered workflows
    attr_reader :workflows

    # @return [Integer] Number of concurrent task execution slots
    attr_reader :slots

    # @return [Hash] Worker labels for scheduling
    attr_reader :labels

    # @return [Hatchet::Client] The Hatchet client
    attr_reader :client

    # @return [String, nil] Worker ID assigned by the server
    attr_accessor :worker_id

    # @param name [String] Worker name
    # @param client [Hatchet::Client] The Hatchet client
    # @param workflows [Array<Workflow, Task>] Workflows to register
    # @param slots [Integer] Number of concurrent task slots (default: 10)
    # @param labels [Hash] Worker labels (default: {})
    def initialize(name:, client:, workflows: [], slots: 10, labels: {})
      @name = name
      @client = client
      @workflows = workflows
      @slots = slots
      @labels = client.config.worker_preset_labels.merge(labels)
      @worker_id = nil
      @shutdown = false
    end

    # Start the worker. This blocks until shutdown is requested.
    #
    # 1. Registers workflows with the Hatchet server
    # 2. Starts the health check server
    # 3. Starts the action listener
    # 4. Processes actions in the thread pool
    # 5. Handles graceful shutdown on SIGINT/SIGTERM
    def start
      setup_signal_handlers

      @client.config.logger.info("Starting worker '#{@name}' with #{@slots} slots")
      @client.config.logger.info("Registering #{@workflows.length} workflow(s)")

      # Register workflows with the server
      register_workflows

      # Start the health check server
      start_health_check

      # Start the action listener loop
      run_action_listener
    rescue Interrupt
      @shutdown = true
      @client.config.logger.info("Worker '#{@name}' interrupted, shutting down...")
    rescue StandardError => e
      @client.config.logger.error("Worker error: #{e.message}")
      raise
    end

    # Request graceful shutdown
    def stop
      @shutdown = true
    end

    # Access worker context for label management
    #
    # @return [WorkerContext]
    def context
      @context ||= WorkerContext.new(worker: self)
    end

    private

    def setup_signal_handlers
      @main_thread = Thread.current

      Signal.trap("INT") do
        @shutdown = true
        @main_thread&.raise(Interrupt)
      end

      Signal.trap("TERM") do
        @shutdown = true
        @main_thread&.raise(Interrupt)
      end
    end

    def register_workflows
      # Build protos for all workflows
      registrations = @workflows.filter_map do |wf|
        if wf.is_a?(Workflow) || (wf.is_a?(Task) && wf.workflow)
          workflow = wf.is_a?(Workflow) ? wf : wf.workflow
          { workflow: workflow, proto: workflow.to_proto(@client.config) }
        end
      end

      # Register all workflows concurrently via gRPC admin (v1 PutWorkflow)
      threads = registrations.map do |reg|
        Thread.new do
          @client.admin_grpc.put_workflow(reg[:proto])
          @client.config.logger.info("  - Registered workflow: #{reg[:workflow].name}")
        end
      end

      threads.each(&:join)

      # Collect all action IDs (service_name:task_name)
      action_ids = collect_action_ids

      # Register the worker with the dispatcher
      response = @client.dispatcher_grpc.register(
        name: @name,
        actions: action_ids,
        slots: @slots,
        labels: @labels,
      )

      @worker_id = response.worker_id
      @client.config.logger.info("Worker registered with id=#{@worker_id}, #{action_ids.length} action(s)")
    end

    def start_health_check
      port = @client.config.healthcheck.port

      @health_thread = Thread.new do
        server = TCPServer.new("0.0.0.0", port)
        @client.config.logger.info("Health check server listening on port #{port}")

        loop do
          break if @shutdown

          begin
            client_socket = server.accept_nonblock
            response = "HTTP/1.1 200 OK\r\nContent-Length: 2\r\nConnection: close\r\n\r\nOK"
            client_socket.puts response
            client_socket.close
          rescue IO::WaitReadable
            server.wait_readable(1)
            retry unless @shutdown
          rescue StandardError => e
            @client.config.logger.debug("Health check error: #{e.message}")
          end
        end

        begin
          server.close
        rescue StandardError
          nil
        end
      end
    end

    def collect_action_ids
      ids = []
      @workflows.each do |wf|
        workflow = wf.is_a?(Workflow) ? wf : wf.workflow
        next unless workflow

        service_name = @client.config.apply_namespace(workflow.name.downcase)

        workflow.tasks.each_key do |name|
          ids << "#{service_name}:#{name}"
        end

        ids << "#{service_name}:on_failure" if workflow.on_failure
        ids << "#{service_name}:on_success" if workflow.on_success
      end
      ids
    end

    def run_action_listener
      @client.config.logger.info("Worker '#{@name}' is running. Press Ctrl+C to stop.")

      # Create the runner for executing tasks
      runner = WorkerRuntime::Runner.new(
        workflows: @workflows,
        slots: @slots,
        dispatcher_client: @client.dispatcher_grpc,
        event_client: @client.event_grpc,
        logger: @client.config.logger,
        client: @client,
      )

      # Create the action listener with retry/reconnect logic
      listener = WorkerRuntime::ActionListener.new(
        dispatcher_client: @client.dispatcher_grpc,
        worker_id: @worker_id,
        logger: @client.config.logger,
      )

      listener.start do |action|
        break if @shutdown

        runner.execute(action)
      end

      @client.config.logger.info("Worker '#{@name}' shutting down...")
      runner.shutdown
    end
  end

  # Provides worker-level operations available from within task contexts
  class WorkerContext
    # @param worker [Worker] The worker instance
    def initialize(worker:)
      @worker = worker
    end

    # Get the worker ID
    # @return [String, nil]
    def id
      @worker.worker_id
    end

    # Get current worker labels
    # @return [Hash]
    def labels
      @worker.labels
    end

    # Upsert worker labels on the server and update local state.
    #
    # @param new_labels [Hash] Labels to add or update
    def upsert_labels(new_labels)
      @worker.labels.merge!(new_labels)

      return unless @worker.worker_id && @worker.client

      @worker.client.dispatcher_grpc.upsert_worker_labels(
        worker_id: @worker.worker_id,
        labels: new_labels,
      )
    end
  end
end
