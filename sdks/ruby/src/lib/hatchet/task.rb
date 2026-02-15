# frozen_string_literal: true

require "securerandom"

module Hatchet
  # Represents a task within a workflow (or a standalone task).
  #
  # Tasks are the basic unit of work in Hatchet. They can be defined as part of
  # a workflow or as standalone tasks. Each task has a block that executes the
  # task logic, receiving the workflow input and a context object.
  #
  # @example Task defined in a workflow
  #   step1 = workflow.task(:step1) { |input, ctx| { "result" => "done" } }
  #
  # @example Standalone task
  #   task = hatchet.task(name: "my_task") { |input, ctx| { "result" => "done" } }
  class Task
    DURATION_MAP = {
      second: :SECOND, minute: :MINUTE, hour: :HOUR,
      day: :DAY, week: :WEEK, month: :MONTH, year: :YEAR,
    }.freeze

    COMPARATOR_MAP = {
      equal: :EQUAL, not_equal: :NOT_EQUAL,
      greater_than: :GREATER_THAN, greater_than_or_equal: :GREATER_THAN_OR_EQUAL,
      less_than: :LESS_THAN, less_than_or_equal: :LESS_THAN_OR_EQUAL,
    }.freeze

    # @return [Symbol, String] Task name
    attr_reader :name

    # @return [Array<Task, Symbol>] Parent task references
    attr_reader :parents

    # @return [Integer, nil] Execution timeout in seconds
    attr_reader :execution_timeout

    # @return [Integer, nil] Schedule timeout in seconds
    attr_reader :schedule_timeout

    # @return [Integer, nil] Maximum number of retries
    attr_reader :retries

    # @return [Integer, nil] Maximum backoff seconds between retries
    attr_reader :backoff_max_seconds

    # @return [Float, nil] Backoff factor between retries
    attr_reader :backoff_factor

    # @return [Array<RateLimit>] Rate limits applied to this task
    attr_reader :rate_limits

    # @return [Array<ConcurrencyExpression>, ConcurrencyExpression, nil] Task-level concurrency
    attr_reader :concurrency

    # @return [Hash, nil] Desired worker labels for scheduling
    attr_reader :desired_worker_labels

    # @return [Array] Wait-for conditions
    attr_reader :wait_for

    # @return [Array] Skip-if conditions
    attr_reader :skip_if

    # @return [Boolean] Whether this is a durable task
    attr_reader :durable

    # @return [Proc, nil] The task execution block
    attr_reader :fn

    # @return [Workflow, nil] The owning workflow
    attr_reader :workflow

    # @return [Hatchet::Client, nil] The Hatchet client
    attr_reader :client

    # @return [Hash, nil] Dependency providers
    attr_reader :deps

    # @param name [Symbol, String] Task name
    # @param parents [Array<Task, Symbol>] Parent tasks
    # @param execution_timeout [Integer, nil] Execution timeout in seconds
    # @param schedule_timeout [Integer, nil] Schedule timeout in seconds
    # @param retries [Integer, nil] Max retries
    # @param backoff_max_seconds [Integer, nil] Max backoff seconds
    # @param backoff_factor [Float, nil] Backoff multiplier
    # @param rate_limits [Array<RateLimit>] Rate limits
    # @param concurrency [Array<ConcurrencyExpression>, ConcurrencyExpression, nil]
    # @param desired_worker_labels [Hash, nil]
    # @param wait_for [Array] Wait conditions
    # @param skip_if [Array] Skip conditions
    # @param durable [Boolean] Whether this is a durable task
    # @param workflow [Workflow, nil] The owning workflow
    # @param client [Hatchet::Client, nil] The client
    # @param deps [Hash, nil] Dependency providers
    # @param block [Proc] The task execution block
    def initialize(
      name:,
      parents: [],
      execution_timeout: nil,
      schedule_timeout: nil,
      retries: nil,
      backoff_max_seconds: nil,
      backoff_factor: nil,
      rate_limits: [],
      concurrency: nil,
      desired_worker_labels: nil,
      wait_for: [],
      skip_if: [],
      durable: false,
      workflow: nil,
      client: nil,
      deps: nil,
      &block
    )
      @name = name.to_sym
      @parents = parents
      @execution_timeout = execution_timeout
      @schedule_timeout = schedule_timeout
      @retries = retries
      @backoff_max_seconds = backoff_max_seconds
      @backoff_factor = backoff_factor
      @rate_limits = rate_limits
      @concurrency = concurrency
      @desired_worker_labels = desired_worker_labels
      @wait_for = wait_for
      @skip_if = skip_if
      @durable = durable
      @workflow = workflow
      @client = client
      @deps = deps
      # Convert Proc to lambda to avoid LocalJumpError on bare `return`
      @fn = block
    end

    # Execute the task with the given input and context
    #
    # @param input [Hash] Task input
    # @param context [Context] Task context
    # @return [Object] Task output
    def call(input, context)
      raise Error, "No block defined for task #{@name}" unless @fn

      @fn.call(input, context)
    end

    # Convert this task to a V1::CreateTaskOpts protobuf message.
    #
    # @param service_name [String] The workflow service name (namespaced)
    # @return [V1::CreateTaskOpts]
    def to_proto(service_name)
      opts = {
        readable_id: @name.to_s,
        action: "#{service_name}:#{@name}",
        inputs: "{}",
        parents: parent_names,
        retries: @retries || 0,
      }

      # Timeout as duration string (e.g. "60s")
      opts[:timeout] = duration_to_expr(@execution_timeout) if @execution_timeout

      # Schedule timeout
      opts[:schedule_timeout] = duration_to_expr(@schedule_timeout) if @schedule_timeout

      # Rate limits
      opts[:rate_limits] = @rate_limits.map { |rl| rate_limit_to_proto(rl) } if @rate_limits && !@rate_limits.empty?

      # Worker labels
      opts[:worker_labels] = build_worker_labels_map(@desired_worker_labels) if @desired_worker_labels && !@desired_worker_labels.empty?

      # Backoff settings
      opts[:backoff_factor] = @backoff_factor if @backoff_factor
      opts[:backoff_max_seconds] = @backoff_max_seconds if @backoff_max_seconds

      # Task-level concurrency
      if @concurrency
        conc_list = @concurrency.is_a?(Array) ? @concurrency : [@concurrency]
        opts[:concurrency] = conc_list.map(&:to_proto)
      end

      # Conditions (wait_for, skip_if)
      conditions_proto = conditions_to_proto
      opts[:conditions] = conditions_proto if conditions_proto

      ::V1::CreateTaskOpts.new(**opts)
    end

    # Run this task (or its owning workflow) synchronously.
    #
    # For standalone tasks the result is automatically unwrapped so that the
    # caller receives the task output directly (e.g. `{"result" => "done"}`)
    # rather than the workflow-level output keyed by task name
    # (e.g. `{"my_task" => {"result" => "done"}}`).
    #
    # @param input [Hash] Input data
    # @param options [TriggerWorkflowOptions, nil] Trigger options
    # @return [Hash] The task output
    def run(input = {}, options: nil)
      target = @workflow || self
      raise Error, "No client associated with task #{@name}" unless effective_client

      result = effective_client.admin.trigger_workflow(target, input, options: options)
      extract_result(result)
    end

    # Run this task without waiting for the result.
    #
    # Returns a {TaskRunRef} whose +result+ method automatically unwraps the
    # task output, matching the behaviour of {#run}.
    #
    # @param input [Hash] Input data
    # @param options [TriggerWorkflowOptions, nil] Trigger options
    # @return [TaskRunRef]
    def run_no_wait(input = {}, options: nil)
      target = @workflow || self
      raise Error, "No client associated with task #{@name}" unless effective_client

      ref = effective_client.admin.trigger_workflow_no_wait(target, input, options: options)
      TaskRunRef.new(workflow_run_ref: ref, task_name: @name)
    end

    # Run many instances of this task in bulk
    #
    # @param items [Array<Hash>] Bulk run items
    # @param return_exceptions [Boolean] Whether to return exceptions instead of raising
    # @return [Array] Results (each unwrapped to the task output)
    def run_many(items, return_exceptions: false)
      raise Error, "No client associated with task #{@name}" unless effective_client

      results = effective_client.admin.trigger_workflow_many(self, items, return_exceptions: return_exceptions)
      results.map { |r| r.is_a?(Exception) ? r : extract_result(r) }
    end

    # Run many instances without waiting for results
    #
    # @param items [Array<Hash>] Bulk run items
    # @return [Array<TaskRunRef>]
    def run_many_no_wait(items)
      raise Error, "No client associated with task #{@name}" unless effective_client

      refs = effective_client.admin.trigger_workflow_many_no_wait(self, items)
      refs.map { |ref| TaskRunRef.new(workflow_run_ref: ref, task_name: @name) }
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

    # Execute task in unit test mode with mocked context
    #
    # @param input [Hash] Task input
    # @param additional_metadata [Hash] Metadata for the context
    # @param retry_count [Integer] Simulated retry count
    # @param parent_outputs [Hash] Mocked parent task outputs
    # @return [Object] Task output
    def mock_run(input:, additional_metadata: {}, retry_count: 0, parent_outputs: {})
      ctx = Context.new(
        workflow_run_id: "mock-run-id",
        step_run_id: "mock-step-run-id",
        action: nil,
        client: nil,
        additional_metadata: additional_metadata,
        retry_count: retry_count,
        parent_outputs: parent_outputs,
      )

      call(input, ctx)
    end

    # @return [String] The workflow ID (for API calls)
    def id
      @workflow&.id || @name.to_s
    end

    private

    def effective_client
      @client || @workflow&.client
    end

    # Extract this task's output from the workflow-level output.
    #
    # The API returns output keyed by task readable_id, e.g.
    #   {"my_task" => {"result" => "done"}}
    # This method unwraps it to just the task output:
    #   {"result" => "done"}
    #
    # @param result [Hash, Object] The workflow run output
    # @return [Hash, Object] The task-specific output
    def extract_result(result)
      return result unless result.is_a?(Hash)

      result[@name.to_s] || result
    end

    # Get parent task names as strings
    def parent_names
      @parents.map do |p|
        case p
        when Task then p.name.to_s
        when Symbol then p.to_s
        when String then p
        else p.to_s
        end
      end
    end

    # Convert a timeout value (Integer seconds or String) to a duration expression
    def duration_to_expr(value)
      return value if value.is_a?(String)

      "#{value}s"
    end

    # Convert a RateLimit to a V1::CreateTaskRateLimit proto
    def rate_limit_to_proto(rate_limit)
      args = {}

      if rate_limit.respond_to?(:static_key) && rate_limit.static_key
        args[:key] = rate_limit.static_key
      elsif rate_limit.respond_to?(:dynamic_key) && rate_limit.dynamic_key
        args[:key] = rate_limit.dynamic_key
        args[:key_expr] = rate_limit.dynamic_key
      end

      args[:units] = rate_limit.units if rate_limit.respond_to?(:units)

      # Always set duration (default MINUTE)
      dur = rate_limit.respond_to?(:duration) && rate_limit.duration ? rate_limit.duration : :minute
      args[:duration] = DURATION_MAP[dur] || :SECOND

      # Always set limit_values_expr (default "-1")
      limit_val = rate_limit.respond_to?(:limit) && rate_limit.limit ? rate_limit.limit : -1
      args[:limit_values_expr] = limit_val.to_s

      ::V1::CreateTaskRateLimit.new(**args)
    end

    # Build the worker_labels map for the proto
    def build_worker_labels_map(labels)
      labels.each_with_object({}) do |(k, v), map|
        dwl = if v.is_a?(Hash)
                dwl_args = {}
                dwl_args[:str_value] = v[:str_value].to_s if v[:str_value]
                dwl_args[:int_value] = v[:int_value] if v[:int_value]
                dwl_args[:required] = v[:required] if v.key?(:required)
                dwl_args[:weight] = v[:weight] if v[:weight]

                dwl_args[:comparator] = COMPARATOR_MAP[v[:comparator]] || :EQUAL if v[:comparator]

                ::V1::DesiredWorkerLabels.new(**dwl_args)
              elsif v.is_a?(Integer)
                ::V1::DesiredWorkerLabels.new(int_value: v)
              else
                ::V1::DesiredWorkerLabels.new(str_value: v.to_s)
              end
        map[k.to_s] = dwl
      end
    end

    # Convert wait_for and skip_if conditions to a TaskConditions proto
    def conditions_to_proto
      return nil if (@wait_for.nil? || @wait_for.empty?) && (@skip_if.nil? || @skip_if.empty?)

      sleep_conditions = []
      user_event_conditions = []

      # Process wait_for conditions (action = QUEUE)
      (@wait_for || []).each do |cond|
        process_condition(cond, :QUEUE, sleep_conditions, user_event_conditions)
      end

      # Process skip_if conditions (action = SKIP)
      (@skip_if || []).each do |cond|
        process_condition(cond, :SKIP, sleep_conditions, user_event_conditions)
      end

      return nil if sleep_conditions.empty? && user_event_conditions.empty?

      ::V1::TaskConditions.new(
        sleep_conditions: sleep_conditions,
        user_event_conditions: user_event_conditions,
      )
    end

    # Process a single condition into the appropriate proto list.
    # Delegates to ConditionConverter for shared logic.
    def process_condition(cond, action, sleep_conditions, user_event_conditions)
      ConditionConverter.convert_condition(
        cond,
        action: action,
        sleep_conditions: sleep_conditions,
        user_event_conditions: user_event_conditions,
        proto_method: :to_proto,
        proto_arg: action,
      )
    end
  end
end
