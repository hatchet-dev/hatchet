# frozen_string_literal: true
# typed: strict

require_relative "hatchet/version"
require_relative "hatchet/config"

# Define base error class before loading submodules that depend on it
module Hatchet
  # Base error class for all Hatchet-related errors
  class Error < StandardError; end
end

require_relative "hatchet/clients"
require_relative "hatchet/features/events"
require_relative "hatchet/features/runs"
require_relative "hatchet/features/tenant"
require_relative "hatchet/features/logs"
require_relative "hatchet/features/workers"
require_relative "hatchet/features/cel"
require_relative "hatchet/features/workflows"
require_relative "hatchet/features/filters"
require_relative "hatchet/features/metrics"
require_relative "hatchet/features/rate_limits"
require_relative "hatchet/features/cron"
require_relative "hatchet/features/scheduled"

# Core classes
require_relative "hatchet/exceptions"
require_relative "hatchet/concurrency"
require_relative "hatchet/conditions"
require_relative "hatchet/rate_limit"
require_relative "hatchet/labels"
require_relative "hatchet/trigger_options"
require_relative "hatchet/default_filter"
require_relative "hatchet/workflow_run"
require_relative "hatchet/context"
require_relative "hatchet/durable_context"
require_relative "hatchet/task"
require_relative "hatchet/workflow"
require_relative "hatchet/context_vars"
require_relative "hatchet/worker_obj"

# gRPC connection and client infrastructure
require_relative "hatchet/connection"

# Generated protobuf contracts (add contracts directory to load path for internal requires)
$LOAD_PATH.unshift(File.join(__dir__, "hatchet", "contracts")) unless $LOAD_PATH.include?(File.join(__dir__, "hatchet", "contracts"))
require_relative "hatchet/contracts/dispatcher/dispatcher_pb"
require_relative "hatchet/contracts/dispatcher/dispatcher_services_pb"
require_relative "hatchet/contracts/events/events_pb"
require_relative "hatchet/contracts/events/events_services_pb"
require_relative "hatchet/contracts/workflows/workflows_pb"
require_relative "hatchet/contracts/workflows/workflows_services_pb"
require_relative "hatchet/contracts/v1/shared/condition_pb"
require_relative "hatchet/contracts/v1/dispatcher_pb"
require_relative "hatchet/contracts/v1/dispatcher_services_pb"
require_relative "hatchet/contracts/v1/workflows_pb"
require_relative "hatchet/contracts/v1/workflows_services_pb"

# gRPC client wrappers
require_relative "hatchet/clients/grpc/dispatcher"
require_relative "hatchet/clients/grpc/admin"
require_relative "hatchet/clients/grpc/event_client"

# Worker runtime
require_relative "hatchet/worker/action_listener"
require_relative "hatchet/worker/workflow_run_listener"
require_relative "hatchet/worker/runner"

# Ruby SDK for Hatchet workflow engine
#
# @see https://docs.hatchet.run for Hatchet documentation
module Hatchet
  # The main client for interacting with Hatchet services.
  #
  # @example Basic usage with API token
  #   hatchet = Hatchet::Client.new()
  #
  # @example With custom configuration
  #   hatchet = Hatchet::Client.new(
  #     token: "your-jwt-token",
  #     namespace: "production"
  #   )
  #
  # @example Define a workflow
  #   wf = hatchet.workflow(name: "MyWorkflow")
  #   step1 = wf.task(:step1) { |input, ctx| { "result" => 42 } }
  #
  # @example Define a standalone task
  #   my_task = hatchet.task(name: "my_task") { |input, ctx| { "result" => "done" } }
  class Client
    # @return [Config] The configuration object used by this client
    attr_reader :config

    # Initialize a new Hatchet client with the given configuration options.
    #
    # @param options [Hash] Configuration options for the client
    # @option options [Boolean] :debug Enable debug logging (default: false)
    # @option options [String] :token The JWT token for authentication (required)
    # @option options [String] :tenant_id Override tenant ID (extracted from JWT token 'sub' field if not provided)
    # @option options [String] :host_port gRPC server host and port (default: "localhost:7070")
    # @option options [String] :server_url Server URL for HTTP requests
    # @option options [String] :namespace Namespace prefix for resource names (default: "")
    # @option options [Logger] :logger Custom logger instance
    # @option options [Hash] :worker_preset_labels Hash of preset labels for workers
    #
    # @raise [Error] if token or configuration is missing or invalid
    def initialize(**options)
      @debug = options.delete(:debug) || false
      @config = Config.new(**options)
    end

    def rest_client
      @rest_client ||= Hatchet::Clients.rest_client(@config)
    end

    # Feature Client for interacting with Hatchet events
    # @return [Hatchet::Features::Events]
    def events
      @events ||= Hatchet::Features::Events.new(rest_client, event_grpc, @config)
    end

    # Feature Client for interacting with Hatchet workflow runs
    # @return [Hatchet::Features::Runs]
    def runs
      @runs ||= Hatchet::Features::Runs.new(rest_client, @config, client: self)
    end

    # Feature Client for interacting with the current tenant
    # @return [Hatchet::Features::Tenant]
    def tenant
      @tenant ||= Hatchet::Features::Tenant.new(rest_client, @config)
    end

    # Feature Client for interacting with Hatchet logs
    # @return [Hatchet::Features::Logs]
    def logs
      @logs ||= Hatchet::Features::Logs.new(rest_client, @config)
    end

    # Feature Client for managing workers
    # @return [Hatchet::Features::Workers]
    def workers
      @workers ||= Hatchet::Features::Workers.new(rest_client, @config)
    end

    # Feature Client for debugging CEL expressions
    # @return [Hatchet::Features::CEL]
    def cel
      @cel ||= Hatchet::Features::CEL.new(rest_client, @config)
    end

    # Feature Client for managing workflow definitions
    # @return [Hatchet::Features::Workflows]
    def workflows
      @workflows ||= Hatchet::Features::Workflows.new(rest_client, @config)
    end

    # Feature Client for managing filters
    # @return [Hatchet::Features::Filters]
    def filters
      @filters ||= Hatchet::Features::Filters.new(rest_client, @config)
    end

    # Feature Client for reading metrics
    # @return [Hatchet::Features::Metrics]
    def metrics
      @metrics ||= Hatchet::Features::Metrics.new(rest_client, @config)
    end

    # Feature Client for managing rate limits
    # @return [Hatchet::Features::RateLimits]
    def rate_limits
      @rate_limits ||= Hatchet::Features::RateLimits.new(admin_grpc, @config)
    end

    # Feature Client for managing cron workflows
    # @return [Hatchet::Features::Cron]
    def cron
      @cron ||= Hatchet::Features::Cron.new(rest_client, @config)
    end

    # Feature Client for managing scheduled workflows
    # @return [Hatchet::Features::Scheduled]
    def scheduled
      @scheduled ||= Hatchet::Features::Scheduled.new(rest_client, @config)
    end

    # Create a new workflow definition
    #
    # @param name [String] Workflow name
    # @param opts [Hash] Workflow options
    # @return [Hatchet::Workflow]
    #
    # @example
    #   wf = hatchet.workflow(name: "MyWorkflow")
    #   wf.task(:step1) { |input, ctx| { "value" => 42 } }
    def workflow(name:, **opts)
      Workflow.new(name: name, client: self, **opts)
    end

    # Create a standalone task (auto-wraps in a single-task workflow)
    #
    # @param name [String] Task name
    # @param opts [Hash] Task options
    # @yield [input, ctx] The task execution block
    # @return [Hatchet::Task]
    #
    # @example
    #   my_task = hatchet.task(name: "my_task") { |input, ctx| { "result" => "done" } }
    def task(name:, **opts, &block)
      # Create a workflow wrapper for standalone tasks
      wf = Workflow.new(name: name, client: self,
                        on_events: opts.delete(:on_events) || [],
                        default_filters: opts.delete(:default_filters) || [],)
      wf.task(name, **opts, &block)
    end

    # Create a standalone durable task
    #
    # @param name [String] Task name
    # @param opts [Hash] Task options
    # @yield [input, ctx] The task execution block
    # @return [Hatchet::Task]
    def durable_task(name:, **opts, &block)
      wf = Workflow.new(name: name, client: self,
                        on_events: opts.delete(:on_events) || [],
                        default_filters: opts.delete(:default_filters) || [],)
      wf.durable_task(name, **opts, &block)
    end

    # Create a new worker
    #
    # @param name [String] Worker name
    # @param opts [Hash] Worker options (workflows:, slots:, labels:, lifespan:)
    # @return [Hatchet::Worker]
    #
    # @example
    #   worker = hatchet.worker("my-worker", workflows: [wf], slots: 10)
    #   worker.start
    def worker(name, **opts)
      Worker.new(name: name, client: self, **opts)
    end

    # Convenience accessor for the logger
    # @return [Logger]
    def logger
      @config.logger
    end

    # @return [String] The tenant ID
    def tenant_id
      @config.tenant_id
    end

    # Shared gRPC channel (lazy-initialized).
    # A single channel is shared across all gRPC stubs for connection reuse.
    #
    # @return [GRPC::Core::Channel]
    def channel
      @channel ||= Connection.new_channel(@config)
    end

    # gRPC Dispatcher client (lazy-initialized).
    #
    # @return [Hatchet::Clients::Grpc::Dispatcher]
    def dispatcher_grpc
      @dispatcher_grpc ||= Clients::Grpc::Dispatcher.new(config: @config, channel: channel)
    end

    # gRPC Admin client (lazy-initialized).
    # Uses both v0 WorkflowService and v1 AdminService stubs.
    #
    # @return [Hatchet::Clients::Grpc::Admin]
    def admin_grpc
      @admin_grpc ||= Clients::Grpc::Admin.new(config: @config, channel: channel)
    end

    # gRPC Event client (lazy-initialized).
    #
    # @return [Hatchet::Clients::Grpc::EventClient]
    def event_grpc
      @event_grpc ||= Clients::Grpc::EventClient.new(config: @config, channel: channel)
    end

    # Pooled gRPC listener for workflow run completion events (lazy-initialized).
    #
    # Maintains a single bidi stream to `Dispatcher.SubscribeToWorkflowRuns`
    # shared by all callers of `WorkflowRunRef#result`.
    #
    # @return [Hatchet::WorkflowRunListener]
    def workflow_run_listener
      @workflow_run_listener ||= WorkflowRunListener.new(config: @config, channel: channel)
    end

    # High-level admin client for workflow triggering.
    # Delegates to the gRPC admin client with context variable propagation.
    #
    # @return [AdminClient]
    def admin
      @admin ||= AdminClient.new(client: self)
    end
  end

  # Admin client for triggering and scheduling workflows.
  #
  # Delegates to the gRPC admin client for actual RPC calls, while handling
  # context variable propagation for parent-child workflow linking.
  class AdminClient
    def initialize(client:)
      @client = client
      @spawn_indices = ContextVars::SpawnIndexTracker.new
    end

    # Trigger a workflow run and wait for result.
    #
    # @param workflow_or_task [Workflow, Task, String] The workflow or task to trigger
    # @param input [Hash] Workflow input
    # @param options [TriggerWorkflowOptions, nil] Trigger options
    # @return [Hash] The workflow run result
    def trigger_workflow(workflow_or_task, input, options: nil)
      ref = trigger_workflow_no_wait(workflow_or_task, input, options: options)
      ref.result
    end

    # Trigger a workflow run without waiting for the result.
    #
    # @param workflow_or_task [Workflow, Task, String] The workflow or task to trigger
    # @param input [Hash] Workflow input
    # @param options [TriggerWorkflowOptions, nil] Trigger options
    # @return [WorkflowRunRef] A reference to the running workflow
    def trigger_workflow_no_wait(workflow_or_task, input, options: nil)
      name = workflow_or_task.respond_to?(:name) ? workflow_or_task.name : workflow_or_task.to_s

      # Merge user options with context vars for parent-child linking
      opts = build_trigger_options(options)

      run_id = @client.admin_grpc.trigger_workflow(name, input: input, options: opts)
      WorkflowRunRef.new(
        workflow_run_id: run_id,
        client: @client,
        listener: @client.workflow_run_listener,
      )
    end

    # Trigger many workflow runs and wait for all results.
    #
    # @param workflow_or_task [Workflow, Task, String] The workflow or task to trigger
    # @param items [Array<Hash>] Array of { input:, options: } items
    # @param return_exceptions [Boolean] Return exceptions instead of raising
    # @return [Array] Results or exceptions
    def trigger_workflow_many(workflow_or_task, items, return_exceptions: false)
      refs = trigger_workflow_many_no_wait(workflow_or_task, items)

      # Collect results concurrently using threads so that all subscriptions
      # are sent at once rather than serially waiting for each one.
      threads = refs.map do |ref|
        Thread.new do
          if return_exceptions
            begin
              ref.result
            rescue StandardError => e
              e
            end
          else
            ref.result
          end
        end
      end

      threads.map(&:value)
    end

    # Trigger many workflow runs without waiting.
    #
    # Uses bulk gRPC triggering for efficiency (batched by 1000).
    #
    # @param workflow_or_task [Workflow, Task, String] The workflow or task to trigger
    # @param items [Array<Hash>] Array of { input:, options: } items
    # @return [Array<WorkflowRunRef>] References to the running workflows
    def trigger_workflow_many_no_wait(workflow_or_task, items)
      name = workflow_or_task.respond_to?(:name) ? workflow_or_task.name : workflow_or_task.to_s

      # Build trigger items with context vars for parent-child linking
      trigger_items = items.map do |item|
        input = item[:input] || {}
        opts = build_trigger_options(item[:options])
        { input: input, options: opts }
      end

      run_ids = @client.admin_grpc.bulk_trigger_workflow(name, trigger_items)
      run_ids.map do |run_id|
        WorkflowRunRef.new(
          workflow_run_id: run_id,
          client: @client,
          listener: @client.workflow_run_listener,
        )
      end
    end

    # Schedule a workflow for future execution.
    #
    # @param workflow [Workflow, Task, String] The workflow to schedule
    # @param time [Time] When to execute
    # @param input [Hash] Workflow input
    # @param options [TriggerWorkflowOptions, nil] Schedule options
    # @return [Object] Schedule response
    def schedule_workflow(workflow, time, input: {}, options: nil)
      name = workflow.respond_to?(:name) ? workflow.name : workflow.to_s
      opts = build_trigger_options(options)
      @client.admin_grpc.schedule_workflow(name, run_at: time, input: input, options: opts)
    end

    private

    def build_trigger_options(user_options)
      # Merge user options with context vars for parent-child linking
      parent_workflow_run_id = ContextVars.workflow_run_id
      parent_step_run_id = ContextVars.step_run_id
      action_key = ContextVars.action_key

      opts = user_options.to_h

      if parent_workflow_run_id
        opts[:parent_id] ||= parent_workflow_run_id
        opts[:parent_task_run_external_id] ||= parent_step_run_id

        opts[:child_index] ||= @spawn_indices.next_index(action_key) if action_key

        parent_meta = ContextVars.additional_metadata
        opts[:additional_metadata] = parent_meta.merge(opts[:additional_metadata] || {}) if parent_meta && !parent_meta.empty?
      end

      opts
    end
  end
end
