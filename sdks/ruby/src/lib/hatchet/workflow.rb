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

    # Define a task within this workflow
    #
    # @param name [Symbol, String] Task name
    # @param opts [Hash] Task options (parents:, execution_timeout:, retries:, etc.)
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

    # Define a durable task within this workflow
    #
    # @param name [Symbol, String] Task name
    # @param opts [Hash] Task options
    # @yield [input, ctx] The task execution block
    # @return [Task] The created durable task
    def durable_task(name, **opts, &)
      task(name, durable: true, **opts, &)
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
      task_protos = @tasks.values.map { |t| t.to_proto(service_name) }

      # On-failure task
      on_failure_proto = @on_failure&.to_proto(service_name)

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

      ::V1::CreateWorkflowVersionRequest.new(**args)
    end

    # Run this workflow synchronously
    #
    # @param input [Hash] Workflow input
    # @param options [TriggerWorkflowOptions, nil] Trigger options
    # @return [Hash] The workflow run output
    def run(input = {}, options: nil)
      raise Error, "No client associated with workflow #{@name}" unless @client

      @client.admin.trigger_workflow(self, input, options: options)
    end

    # Run this workflow without waiting for the result
    #
    # @param input [Hash] Workflow input
    # @param options [TriggerWorkflowOptions, nil] Trigger options
    # @return [WorkflowRunRef]
    def run_no_wait(input = {}, options: nil)
      raise Error, "No client associated with workflow #{@name}" unless @client

      @client.admin.trigger_workflow_no_wait(self, input, options: options)
    end

    # Run many instances of this workflow in bulk
    #
    # @param items [Array<Hash>] Bulk run items
    # @param return_exceptions [Boolean] Return exceptions instead of raising
    # @return [Array] Results
    def run_many(items, return_exceptions: false)
      raise Error, "No client associated with workflow #{@name}" unless @client

      @client.admin.trigger_workflow_many(self, items, return_exceptions: return_exceptions)
    end

    # Run many instances without waiting for results
    #
    # @param items [Array<Hash>] Bulk run items
    # @return [Array<WorkflowRunRef>]
    def run_many_no_wait(items)
      raise Error, "No client associated with workflow #{@name}" unless @client

      @client.admin.trigger_workflow_many_no_wait(self, items)
    end

    # Create a bulk run item for use with run_many
    #
    # @param input [Hash] Input data
    # @param key [String, nil] Deduplication key
    # @param options [TriggerWorkflowOptions, nil] Trigger options
    # @return [Hash] Bulk run item
    def create_bulk_run_item(input: {}, key: nil, options: nil)
      item = { input: input }
      item[:key] = key if key
      item[:options] = options if options
      item
    end

    # Schedule this workflow for future execution
    #
    # @param time [Time] When to execute
    # @param input [Hash] Workflow input
    # @param options [ScheduleTriggerWorkflowOptions, nil] Schedule options
    # @return [Object] Schedule result
    def schedule(time, input: {}, options: nil)
      raise Error, "No client associated with workflow #{@name}" unless @client

      @client.admin.schedule_workflow(self, time, input: input, options: options)
    end

    # Create a cron trigger for this workflow
    #
    # @param cron_name [String] Name for the cron
    # @param expression [String] Cron expression
    # @param input [Hash] Workflow input
    # @return [Object] Cron result
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
