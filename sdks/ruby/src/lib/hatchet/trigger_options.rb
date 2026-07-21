# frozen_string_literal: true

module Hatchet
  # Options for triggering a workflow run
  #
  # @example Trigger with metadata and priority
  #   Hatchet::TriggerWorkflowOptions.new(
  #     additional_metadata: { "user_id" => "123" },
  #     priority: 3
  #   )
  class TriggerWorkflowOptions
    # @return [Hash, nil] Additional metadata to attach to the run
    attr_reader :additional_metadata

    # @return [String, nil] Deduplication key
    attr_reader :key

    # @return [Integer, nil] Priority level (1-4, higher = more priority)
    attr_reader :priority

    # @return [String, nil] Parent workflow run ID (auto-set from context vars)
    attr_reader :parent_id

    # @return [String, nil] Parent step run ID (auto-set from context vars)
    attr_reader :parent_step_run_id

    # @return [Integer, nil] Child index for deterministic replay
    attr_reader :child_index

    # @return [Boolean, nil] Whether to use sticky scheduling
    attr_reader :sticky

    # @return [Hash, nil] Desired worker labels for scheduling
    attr_reader :desired_worker_labels

    # @param additional_metadata [Hash, nil] Metadata to attach to the run
    # @param key [String, nil] Deduplication key
    # @param priority [Integer, nil] Priority level
    # @param parent_id [String, nil] Parent workflow run ID
    # @param parent_step_run_id [String, nil] Parent step run ID
    # @param child_index [Integer, nil] Child index
    # @param sticky [Boolean, nil] Enable sticky scheduling
    # @param desired_worker_labels [Hash, nil] Worker labels for scheduling
    def initialize(
      additional_metadata: nil,
      key: nil,
      priority: nil,
      parent_id: nil,
      parent_step_run_id: nil,
      child_index: nil,
      sticky: nil,
      desired_worker_labels: nil
    )
      @additional_metadata = additional_metadata
      @key = key
      @priority = priority
      @parent_id = parent_id
      @parent_step_run_id = parent_step_run_id
      @child_index = child_index
      @sticky = sticky
      @desired_worker_labels = desired_worker_labels
    end

    # @return [Hash]
    def to_h
      h = {}
      h[:additional_metadata] = @additional_metadata if @additional_metadata
      h[:key] = @key if @key
      h[:priority] = @priority if @priority
      h[:parent_id] = @parent_id if @parent_id
      h[:parent_step_run_id] = @parent_step_run_id if @parent_step_run_id
      h[:child_index] = @child_index if @child_index
      h[:sticky] = @sticky unless @sticky.nil?
      h[:desired_worker_labels] = @desired_worker_labels if @desired_worker_labels
      h
    end
  end

  # Options for scheduling a workflow trigger
  class ScheduleTriggerWorkflowOptions < TriggerWorkflowOptions
  end
end
