# frozen_string_literal: true

module Hatchet
  # Represents a workflow definition with one or more tasks arranged in a DAG.
  #
  # @example Define a simple workflow
  #   wf = hatchet.workflow(name: "MyWorkflow")
  #   step1 = wf.task(:step1) { |input, ctx| { "value" => 42 } }
  #   wf.task(:step2, parents: [step1]) { |input, ctx|
  #     { "result" => ctx.task_output(step1)["value"] + 1 }
  #   }
  class Workflow
    # @return [String] Workflow name
    attr_reader :name

    # @return [Hash<Symbol, Task>] Map of task name to Task object
    attr_reader :tasks

    # @return [Array<String>] Event keys that trigger this workflow
    attr_reader :on_events

    # @return [Array<String>] Cron expressions that trigger this workflow
    attr_reader :on_crons

    # @return [Array<ConcurrencyExpression>, ConcurrencyExpression, nil] Workflow-level concurrency
    attr_reader :concurrency

    # @return [Integer, nil] Default priority for runs (1-4)
    attr_reader :default_priority

    # @return [Hash, nil] Default task settings
    attr_reader :task_defaults

    # @return [Array<DefaultFilter>] Default filters for event triggers
    attr_reader :default_filters

    # @return [Symbol, nil] Sticky strategy (:soft, :hard)
    attr_reader :sticky

    # @return [Hatchet::Client, nil] The Hatchet client
    attr_reader :client

    # @return [Task, nil] The on_failure task
    attr_reader :on_failure

    # @return [Task, nil] The on_success task
    attr_reader :on_success

    # @return [Hatchet::TTLBasedIdempotencyConfig, Hatchet::StatusBasedIdempotencyConfig, nil] Idempotency configuration
    attr_reader :idempotency

    # @return [String, nil] The workflow ID writer (set after registration)
    attr_writer :id

    # @param name [String] Workflow name
    # @param on_events [Array<String>] Event trigger keys
    # @param on_crons [Array<String>] Cron trigger expressions
    # @param concurrency [Array<ConcurrencyExpression>, ConcurrencyExpression, nil]
    # @param default_priority [Integer, nil] Default priority
    # @param task_defaults [Hash, nil] Default task settings
    # @param default_filters [Array<DefaultFilter>] Default filters
    # @param sticky [Symbol, nil] Sticky strategy
    # @param idempotency [Hatchet::TTLBasedIdempotencyConfig, Hatchet::StatusBasedIdempotencyConfig, nil] Idempotency configuration
    # @param client [Hatchet::Client, nil] The client
    def initialize(
      name:,
      on_events: [],
      on_crons: [],
      concurrency: nil,
      default_priority: nil,
      task_defaults: nil,
      default_filters: [],
      sticky: nil,
      idempotency: nil,
      client: nil
    )
      @name = name
      @tasks = {}
      @on_events = on_events
      @on_crons = on_crons
      @concurrency = concurrency
      @default_priority = default_priority
      @task_defaults = task_defaults
      @default_filters = default_filters
      @sticky = sticky
      @idempotency = idempotency
      @client = client
      @on_failure = nil
      @on_success = nil
      @id = nil
    end

    # Get the workflow ID (UUID). If not already set, lazily resolves it
    # by looking up the workflow by name via the REST API.
    #
    # @return [String, nil] The workflow UUID
    def id
      @id ||= resolve_workflow_id
    end

    # Define a task within this workflow. The block receives the workflow input
    # and a {Context} object, and its return value (a Hash) becomes the task
    # output.
    #
    # @param name [Symbol, String] The name of the task
    # @option opts [Array<Task, Symbol>] :parents ([]) A list of tasks that are parents of the task. Note: parents must be defined before their children
    # @option opts [Integer, String, nil] :execution_timeout (nil) The maximum time to wait for the task to complete, in seconds or as a duration string (e.g. "60s")
    # @option opts [Integer, String, nil] :schedule_timeout (nil) The maximum time to wait for the task to be scheduled
    # @option opts [Integer, nil] :retries (nil) The number of times to retry the task before failing
    # @option opts [Float, nil] :backoff_factor (nil) The backoff factor for controlling exponential backoff in retries
    # @option opts [Integer, nil] :backoff_max_seconds (nil) The maximum number of seconds to allow retries with exponential backoff to continue
    # @option opts [Array<RateLimit>] :rate_limits ([]) A list of rate limit configurations for the task
    # @option opts [ConcurrencyExpression, Array<ConcurrencyExpression>, nil] :concurrency (nil) A concurrency expression (or list of them) controlling the concurrency settings for this task
    # @option opts [Hash, nil] :desired_worker_labels (nil) A hash of desired worker labels that determine to which worker the task should be assigned
    # @option opts [Array] :wait_for ([]) A list of conditions that must be met before the task can run
    # @option opts [Array] :skip_if ([]) A list of conditions that, if met, will cause the task to be skipped
    # @option opts [Hash, nil] :deps (nil) Dependency providers to inject into the task's context
    # @yield [input, ctx] The task execution block
    # @return [Task] The created task
    def task(name, **opts, &)
      t = Task.new(
        name: name,
        workflow: self,
        client: @client,
        **opts,
        &
      )
      @tasks[t.name] = t
      t
    end

    # Define a durable task within this workflow.
    #
    # @param name [Symbol, String] Task name
    # @param eviction_policy [Hatchet::EvictionPolicy, nil] Eviction policy for this
    #   durable task. Defaults to {Hatchet::DEFAULT_DURABLE_TASK_EVICTION_POLICY}
    #   (15-minute TTL, capacity-eviction enabled). Pass ``nil`` to disable
    #   eviction entirely for this task.
    # @param opts [Hash] Other Task options forwarded to {#task}.
    # @yield [input, ctx] The task execution block
    # @return [Task] The created durable task
    def durable_task(name, eviction_policy: Hatchet::DEFAULT_DURABLE_TASK_EVICTION_POLICY, **opts, &)
      task(name, durable: true, eviction_policy: eviction_policy, **opts, &)
    end

    # Define a batch task within this workflow.
    #
    # Batch tasks buffer concurrent runs until Hatchet flushes the batch (size reached or
    # flush interval), then invoke the block once with all buffered inputs keyed by each
    # run's task-run external id. The block must return a Hash mapping each id to its
    # output, or use +broadcast_output+ on the batch config to return the same result to
    # all callers. retries is always forced to 0 for batch tasks.
    #
    # Preview: batch tasks are in beta and may change in future releases.
    #
    # @param name [Symbol, String] Task name
    # @param batch [Hatchet::BatchTaskConfig] Batch configuration
    # @param opts [Hash] Other Task options forwarded to {#task}.
    # @yield [inputs, ctx] The batch execution block, receiving a Hash of task-run external id => input
    # @return [Task] The created batch task
    def batch_task(name, batch:, **opts, &)
      task(name, batch: batch, **opts, &)
    end

    # Define an on_failure task for this workflow
    #
    # @param opts [Hash] Task options
    # @yield [input, ctx] The on_failure task block
    # @return [Task]
    def on_failure_task(**opts, &)
      @on_failure = Task.new(
        name: :on_failure,
        workflow: self,
        client: @client,
        **opts,
        &
      )
    end

    # Define an on_success task for this workflow
    #
    # @param opts [Hash] Task options
    # @yield [input, ctx] The on_success task block
    # @return [Task]
    def on_success_task(**opts, &)
      @on_success = Task.new(
        name: :on_success,
        workflow: self,
        client: @client,
        **opts,
        &
      )
    end

    # Convert this workflow to a V1::CreateWorkflowVersionRequest protobuf message.
    #
    # @param config [Hatchet::Config] The Hatchet configuration (for namespacing)
    # @return [V1::CreateWorkflowVersionRequest]
    def to_proto(config)
      service_name = config.apply_namespace(@name.downcase)

      # Namespace event triggers
      event_triggers = @on_events.map { |e| config.apply_namespace(e) }

      # Convert tasks to proto
      task_protos = @tasks.values.map { |t| t.to_proto(service_name, config: config) }

      # On-failure task
      on_failure_proto = @on_failure&.to_proto(service_name, config: config)

      # Build concurrency
      concurrency_proto = nil
      concurrency_arr = []

      if @concurrency
        conc_list = @concurrency.is_a?(Array) ? @concurrency : [@concurrency]

        if conc_list.length == 1
          concurrency_proto = conc_list.first.to_proto
        else
          concurrency_arr = conc_list.map(&:to_proto)
        end
      end

      # Sticky strategy
      sticky_proto = nil
      if @sticky
        sticky_map = { soft: :SOFT, hard: :HARD }
        sticky_proto = sticky_map[@sticky]
      end

      # Default filters
      filter_protos = (@default_filters || []).map(&:to_proto)

      args = {
        name: config.apply_namespace(@name),
        event_triggers: event_triggers,
        cron_triggers: @on_crons || [],
        tasks: task_protos,
      }

      args[:concurrency] = concurrency_proto if concurrency_proto
      args[:concurrency_arr] = concurrency_arr unless concurrency_arr.empty?
      args[:on_failure_task] = on_failure_proto if on_failure_proto
      args[:sticky] = sticky_proto if sticky_proto
      args[:default_priority] = @default_priority if @default_priority
      args[:default_filters] = filter_protos unless filter_protos.empty?

      args[:idempotency] = @idempotency.to_proto if @idempotency

      ::V1::CreateWorkflowVersionRequest.new(**args)
    end

    # Run this workflow synchronously and wait for it to complete.
    #
    # @param input [Hash] The input data for the workflow
    # @param options [TriggerWorkflowOptions, nil] Additional options for workflow execution, such as +additional_metadata:+ and +priority:+
    # @return [Hash] The workflow run output, keyed by task name (e.g. `{"step1" => {...}, "step2" => {...}}`)
    # @raise [Hatchet::Error] If no client is associated with the workflow
    # @raise [Hatchet::FailedRunError] If the workflow run failed
    def run(input = {}, options: nil)
      raise Error, "No client associated with workflow #{@name}" unless @client

      @client.admin.trigger_workflow(self, input, options: options)
    end

    # Trigger a workflow run without waiting for it to complete. Useful for
    # starting a run and immediately returning a reference to it without
    # blocking while the workflow runs.
    #
    # @param input [Hash] The input data for the workflow
    # @param options [TriggerWorkflowOptions, nil] Additional options for workflow execution
    # @return [WorkflowRunRef] A reference to the workflow run, whose +result+ method blocks until the run completes
    # @raise [Hatchet::Error] If no client is associated with the workflow
    def run_no_wait(input = {}, options: nil)
      raise Error, "No client associated with workflow #{@name}" unless @client

      @client.admin.trigger_workflow_no_wait(self, input, options: options)
    end

    # Run this workflow in bulk and wait for all runs to complete. Runs are
    # triggered via bulk gRPC triggering (batched by 1000) and results are
    # collected concurrently.
    #
    # @param items [Array<Hash>] A list of bulk run items, as created by {#create_bulk_run_item}
    # @param return_exceptions [Boolean] If +true+, exceptions are returned as part of the results instead of being raised
    # @return [Array] A list of results for each workflow run
    # @raise [Hatchet::Error] If no client is associated with the workflow
    def run_many(items, return_exceptions: false)
      raise Error, "No client associated with workflow #{@name}" unless @client

      @client.admin.trigger_workflow_many(self, items, return_exceptions: return_exceptions)
    end

    # Run this workflow in bulk without waiting for the runs to complete.
    #
    # @param items [Array<Hash>] A list of bulk run items, as created by {#create_bulk_run_item}
    # @return [Array<WorkflowRunRef>] A list of references to the triggered workflow runs
    # @raise [Hatchet::Error] If no client is associated with the workflow
    def run_many_no_wait(items)
      raise Error, "No client associated with workflow #{@name}" unless @client

      @client.admin.trigger_workflow_many_no_wait(self, items)
    end

    # Create a bulk run item for this workflow, intended to be used with the
    # {#run_many} methods.
    #
    # @param input [Hash] The input data for the workflow
    # @param key [String, nil] The key for the workflow run, used for identification and deduplication
    # @param options [TriggerWorkflowOptions, nil] Additional options for the workflow run
    # @return [Hash] A bulk run item that can be passed to the +run_many+ methods
    def create_bulk_run_item(input: {}, key: nil, options: nil)
      item = { input: input }
      item[:key] = key if key
      item[:options] = options if options
      item
    end

    # Schedule this workflow to run at a specific time.
    #
    # @param time [Time] When to execute the workflow
    # @param input [Hash] The input data for the workflow
    # @param options [ScheduleTriggerWorkflowOptions, nil] Additional schedule options
    # @return [Object] The schedule response from the Hatchet engine
    # @raise [Hatchet::Error] If no client is associated with the workflow
    def schedule(time, input: {}, options: nil)
      raise Error, "No client associated with workflow #{@name}" unless @client

      @client.admin.schedule_workflow(self, time, input: input, options: options)
    end

    # Create a cron trigger for this workflow.
    #
    # @param cron_name [String] The name of the cron job
    # @param expression [String] The cron expression that defines the schedule
    # @param input [Hash] The input data for the workflow
    # @return [Object] The created cron workflow trigger
    # @raise [Hatchet::Error] If no client is associated with the workflow
    def create_cron(cron_name, expression, input: {})
      raise Error, "No client associated with workflow #{@name}" unless @client

      @client.cron.create(
        workflow_name: @name,
        cron_name: cron_name,
        expression: expression,
        input: input,
      )
    end

    private

    # Resolve the workflow UUID by looking up the workflow by name via the REST API.
    #
    # @return [String, nil] The workflow UUID, or nil if not found or no client
    def resolve_workflow_id
      return nil unless @client

      result = @client.workflows.list(workflow_name: @name)
      rows = result.rows
      return nil if rows.nil? || rows.empty?

      rows.first.metadata&.id
    rescue StandardError
      nil
    end
  end
end
