# frozen_string_literal: true

require_relative "hatchet/version"
require_relative "hatchet/config"

# Ruby SDK for Hatchet workflow engine
#
# @see https://docs.hatchet.run for Hatchet documentation
module Hatchet
  class Error < StandardError; end

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

  require_relative "hatchet/exceptions"
  require_relative "hatchet/engine_version"
  require_relative "hatchet/eviction_policy"
  require_relative "hatchet/concurrency"
  require_relative "hatchet/idempotency"
  require_relative "hatchet/conditions"
  require_relative "hatchet/condition_converter"
  require_relative "hatchet/rate_limit"
  require_relative "hatchet/batch"
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

  require_relative "hatchet/connection"

  $LOAD_PATH.unshift(File.join(__dir__, "hatchet", "contracts")) unless $LOAD_PATH.include?(File.join(__dir__, "hatchet", "contracts"))
  require_relative "hatchet/contracts/dispatcher/dispatcher_pb"
  require_relative "hatchet/contracts/dispatcher/dispatcher_services_pb"
  require_relative "hatchet/contracts/events/events_pb"
  require_relative "hatchet/contracts/events/events_services_pb"
  require_relative "hatchet/contracts/workflows/workflows_pb"
  require_relative "hatchet/contracts/workflows/workflows_services_pb"
  require_relative "hatchet/contracts/v1/shared/condition_pb"
  require_relative "hatchet/contracts/v1/shared/trigger_pb"
  require_relative "hatchet/contracts/v1/dispatcher_pb"
  require_relative "hatchet/contracts/v1/dispatcher_services_pb"
  require_relative "hatchet/contracts/v1/workflows_pb"
  require_relative "hatchet/contracts/v1/workflows_services_pb"

  require_relative "hatchet/clients/grpc/dispatcher"
  require_relative "hatchet/clients/grpc/admin"
  require_relative "hatchet/clients/grpc/event_client"

  require_relative "hatchet/worker/action_listener"
  require_relative "hatchet/worker/workflow_run_listener"
  require_relative "hatchet/worker/durable_eviction/cache"
  require_relative "hatchet/worker/durable_eviction/manager"
  require_relative "hatchet/worker/durable_event_listener"
  require_relative "hatchet/worker/runner"

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
    # @option options [Hash] :worker_preset_labels Default labels applied to all workers
    #
    # @raise [Error] if token or configuration is missing or invalid
    def initialize(**options)
      @debug = options.delete(:debug) || false
      @config = Config.new(**options)
    end

    def rest_client
      @rest_client ||= Hatchet::Clients.rest_client(@config)
    end

    # The events client, which you can use to push events to Hatchet to trigger
    # event-driven workflows.
    # @return [Hatchet::Features::Events]
    def events
      @events ||= Hatchet::Features::Events.new(rest_client, event_grpc, @config)
    end

    # The runs client is a client for interacting with task and workflow runs
    # within Hatchet.
    # @return [Hatchet::Features::Runs]
    def runs
      @runs ||= Hatchet::Features::Runs.new(rest_client, @config, client: self)
    end

    # The tenant client is a client for reading information about the tenant
    # you're operating in.
    # @return [Hatchet::Features::Tenant]
    def tenant
      @tenant ||= Hatchet::Features::Tenant.new(rest_client, @config)
    end

    # The logs client is a client for interacting with Hatchet's logs API.
    # @return [Hatchet::Features::Logs]
    def logs
      @logs ||= Hatchet::Features::Logs.new(rest_client, @config)
    end

    # The workers client is a client for managing workers programmatically
    # within Hatchet.
    # @return [Hatchet::Features::Workers]
    def workers
      @workers ||= Hatchet::Features::Workers.new(rest_client, @config)
    end

    # The CEL client is a client for debugging CEL expressions within Hatchet.
    # @return [Hatchet::Features::CEL]
    def cel
      @cel ||= Hatchet::Features::CEL.new(rest_client, @config)
    end

    # The workflows client is a client for managing workflow declarations
    # programmatically within Hatchet. Note that workflows are the declaration,
    # _not_ the individual runs; if you're looking for runs, use the runs
    # client instead.
    # @return [Hatchet::Features::Workflows]
    def workflows
      @workflows ||= Hatchet::Features::Workflows.new(rest_client, @config)
    end

    # The filters client is a client for managing filters within Hatchet, which
    # scope event triggers to workflows using CEL expressions.
    # @return [Hatchet::Features::Filters]
    def filters
      @filters ||= Hatchet::Features::Filters.new(rest_client, @config)
    end

    # The metrics client is a client for reading metrics out of Hatchet's
    # metrics API.
    # @return [Hatchet::Features::Metrics]
    def metrics
      @metrics ||= Hatchet::Features::Metrics.new(rest_client, @config)
    end

    # The rate limits client is a wrapper for Hatchet's gRPC API that makes it
    # easier to work with rate limits in Hatchet.
    # @return [Hatchet::Features::RateLimits]
    def rate_limits
      @rate_limits ||= Hatchet::Features::RateLimits.new(admin_grpc, @config)
    end

    # The cron client is a client for managing cron workflow triggers within
    # Hatchet.
    # @return [Hatchet::Features::Cron]
    def cron
      @cron ||= Hatchet::Features::Cron.new(rest_client, @config)
    end

    # The scheduled client is a client for managing scheduled workflow runs
    # within Hatchet.
    # @return [Hatchet::Features::Scheduled]
    def scheduled
      @scheduled ||= Hatchet::Features::Scheduled.new(rest_client, @config)
    end

    # Define a Hatchet workflow, which can then declare tasks and be run,
    # scheduled, and so on.
    #
    # @param name [String] The name of the workflow
    # @option opts [Array<String>] :on_events ([]) A list of event triggers for the workflow - events which cause the workflow to be run
    # @option opts [Array<String>] :on_crons ([]) A list of cron triggers for the workflow
    # @option opts [ConcurrencyExpression, Array<ConcurrencyExpression>, nil] :concurrency (nil) A concurrency object (or list of them) controlling the concurrency settings for this workflow
    # @option opts [Integer, nil] :default_priority (nil) The default priority of the workflow. Higher values will cause runs of this workflow to have priority in scheduling over other, lower priority ones
    # @option opts [Hash, nil] :task_defaults (nil) Default task settings for this workflow
    # @option opts [Array<DefaultFilter>] :default_filters ([]) A list of filters to create when the workflow is created
    # @option opts [Symbol, nil] :sticky (nil) A sticky strategy for the workflow, either +:soft+ or +:hard+
    # @option opts [TTLBasedIdempotencyConfig, StatusBasedIdempotencyConfig, nil] :idempotency (nil) An idempotency configuration for the workflow
    # @return [Hatchet::Workflow] The created workflow object, which can be used to declare tasks, run the workflow, and so on
    #
    # @example
    #   wf = hatchet.workflow(name: "MyWorkflow")
    #   wf.task(:step1) { |input, ctx| { "value" => 42 } }
    def workflow(name:, **opts)
      Workflow.new(name: name, client: self, **opts)
    end

    # Create a standalone Hatchet task. The task is automatically wrapped in a
    # single-task workflow, so it can be run, scheduled, and registered on a
    # worker just like a workflow. The block receives the run's input and a
    # {Context} object.
    #
    # @param name [String] The name of the task
    # @option opts [Array<String>] :on_events ([]) A list of event triggers for the task - events which cause the task to be run
    # @option opts [Array<DefaultFilter>] :default_filters ([]) A list of filters to create when the task is created
    # @option opts [TTLBasedIdempotencyConfig, StatusBasedIdempotencyConfig, nil] :idempotency (nil) An idempotency configuration for the task
    # @param opts [Hash] Any other keyword arguments (+retries:+, +execution_timeout:+, +concurrency:+, and so on) are forwarded to the task declaration - see {Workflow#task} for the full list
    # @yield [input, ctx] The task execution block
    # @return [Hatchet::Task] The created task object, which can be run, scheduled, and registered on a worker
    #
    # @example
    #   my_task = hatchet.task(name: "my_task") { |input, ctx| { "result" => "done" } }
    def task(name:, **opts, &block)
      wf = Workflow.new(name: name, client: self,
                        on_events: opts.delete(:on_events) || [],
                        default_filters: opts.delete(:default_filters) || [],
                        idempotency: opts.delete(:idempotency),)
      wf.task(name, **opts, &block)
    end

    # Create a standalone batch task (auto-wraps in a single-task workflow).
    #
    # Batch tasks buffer concurrent runs until Hatchet flushes the batch (size reached or
    # flush interval), then invoke the block once with all buffered inputs keyed by each
    # run's task-run external id. The block must return a Hash mapping each id to its
    # output, or use +broadcast_output+ on the batch config to return the same result to
    # all callers. retries is always forced to 0 for batch tasks.
    #
    # Preview: batch tasks are in beta and may change in future releases.
    #
    # @param name [String] The name of the task
    # @param batch [Hatchet::BatchTaskConfig] The batch configuration (+max_size+, flush interval, +broadcast_output+)
    # @param opts [Hash] Any other keyword arguments (+on_events:+, +idempotency:+, and so on) are forwarded to {#task}
    # @yield [inputs, ctx] The batch execution block, receiving a Hash of task-run external id => input
    # @return [Hatchet::Task] The created batch task object
    #
    # @example
    #   batch = hatchet.batch_task(name: "my_batch", batch: Hatchet::BatchTaskConfig.new(max_size: 3)) do |inputs, ctx|
    #     inputs.transform_values { |input| { "result" => input["message"].upcase } }
    #   end
    def batch_task(name:, batch:, **opts, &block)
      task(name: name, batch: batch, **opts, &block)
    end

    # Create a standalone _durable_ Hatchet task, which works using Hatchet's
    # durable execution capabilities. Durable tasks receive a {DurableContext}
    # with additional methods like +sleep_for+ and +wait_for+.
    #
    # @param name [String] The name of the task
    # @param eviction_policy [Hatchet::EvictionPolicy, nil] Eviction policy for this
    #   durable task. Defaults to {Hatchet::DEFAULT_DURABLE_TASK_EVICTION_POLICY}
    #   (15-minute TTL, capacity-eviction enabled). Pass ``nil`` to disable
    #   eviction entirely for this task.
    # @param opts [Hash] Any other keyword arguments (+retries:+, +execution_timeout:+, and so on) are forwarded to the task declaration - see {Workflow#task} for the full list
    # @yield [input, ctx] The task execution block
    # @return [Hatchet::Task] The created durable task object
    def durable_task(name:, eviction_policy: Hatchet::DEFAULT_DURABLE_TASK_EVICTION_POLICY, **opts, &block)
      wf = Workflow.new(name: name, client: self,
                        on_events: opts.delete(:on_events) || [],
                        default_filters: opts.delete(:default_filters) || [],)
      wf.durable_task(name, eviction_policy: eviction_policy, **opts, &block)
    end

    # Create a Hatchet worker on which to run workflows.
    #
    # @param name [String] The name of the worker
    # @option opts [Array<Workflow, Task>] :workflows ([]) A list of workflows (or standalone tasks) to register on the worker
    # @option opts [Integer] :slots (10) Slot count for standard tasks, i.e. the number of tasks the worker can run concurrently
    # @option opts [Integer, nil] :durable_slots (nil) Slot count for durable tasks; defaults to +slots+ if not provided
    # @option opts [Hash] :labels ({}) A hash of labels to assign to the worker, for use with worker affinity; merged with the client's +worker_preset_labels+
    # @return [Hatchet::Worker] The created worker object, which exposes an instance
    #   method +start+ which can be called to start the worker (blocking until
    #   shutdown), and +stop+ to request a graceful shutdown
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
