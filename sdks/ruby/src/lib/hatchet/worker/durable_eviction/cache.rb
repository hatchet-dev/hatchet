# frozen_string_literal: true

require "monitor"

module Hatchet
  module WorkerRuntime
    module DurableEviction
      # Eviction causes produced by :class:`DurableEvictionCache`.
      module EvictionCause
        TTL_EXCEEDED = :ttl_exceeded
        CAPACITY_PRESSURE = :capacity_pressure
        WORKER_SHUTDOWN = :worker_shutdown
      end

      # Per-run state tracked by the cache.
      #
      # ``wait_count`` is ref-counted so concurrent waits over the same durable
      # run don't prematurely clear the waiting flag when one child completes
      # before the others.
      class DurableRunRecord
        attr_reader :key, :step_run_id, :invocation_count, :eviction_policy, :registered_at
        attr_accessor :waiting_since, :wait_kind, :wait_resource_id, :eviction_reason, :wait_count

        # @param key [String] The action key uniquely identifying this step run invocation.
        # @param step_run_id [String]
        # @param invocation_count [Integer]
        # @param eviction_policy [Hatchet::EvictionPolicy, nil]
        # @param registered_at [Time]
        def initialize(key:, step_run_id:, invocation_count:, eviction_policy:, registered_at:)
          @key = key
          @step_run_id = step_run_id
          @invocation_count = invocation_count
          @eviction_policy = eviction_policy
          @registered_at = registered_at
          @waiting_since = nil
          @wait_kind = nil
          @wait_resource_id = nil
          @eviction_reason = nil
          @wait_count = 0
        end

        def waiting?
          @wait_count.positive?
        end
      end

      # Thread-safe in-memory cache of waiting durable task invocations.
      #
      # Mirrors :class:`hatchet_sdk.worker.durable_eviction.cache.DurableEvictionCache`
      # from the Python SDK. All public methods lock an internal monitor.
      class DurableEvictionCache
        def initialize
          @runs = {}
          @monitor = Monitor.new
        end

        # Register a new durable run invocation.
        def register_run(key, step_run_id:, invocation_count:, now:, eviction_policy:)
          @monitor.synchronize do
            @runs[key] = DurableRunRecord.new(
              key: key,
              step_run_id: step_run_id,
              invocation_count: invocation_count,
              eviction_policy: eviction_policy,
              registered_at: now,
            )
          end
        end

        # Unregister a durable run invocation.
        def unregister_run(key)
          @monitor.synchronize { @runs.delete(key) }
        end

        # Fetch the record for a given key.
        # @return [DurableRunRecord, nil]
        def get(key)
          @monitor.synchronize { @runs[key] }
        end

        # @return [Array<DurableRunRecord>]
        def all_waiting
          @monitor.synchronize { @runs.values.select(&:waiting?) }
        end

        # @param step_run_id [String]
        # @return [String, nil] the action key for the matching record
        def find_key_by_step_run_id(step_run_id)
          @monitor.synchronize do
            @runs.each do |k, rec|
              return k if rec.step_run_id == step_run_id
            end
            nil
          end
        end

        # Mark the run as waiting (ref-counted). Increments the wait counter and
        # stores the wait metadata on the record.
        def mark_waiting(key, now:, wait_kind:, resource_id:)
          @monitor.synchronize do
            rec = @runs[key]
            return unless rec

            rec.wait_count += 1
            rec.waiting_since = now if rec.wait_count == 1
            rec.wait_kind = wait_kind
            rec.wait_resource_id = resource_id
          end
        end

        # Mark the run as active (decrement the wait counter). Floors at zero so
        # unmatched +mark_active+ calls never underflow.
        def mark_active(key, now:)
          @monitor.synchronize do
            rec = @runs[key]
            return unless rec

            rec.wait_count = [rec.wait_count - 1, 0].max
            if rec.wait_count.zero?
              rec.waiting_since = nil
              rec.wait_kind = nil
              rec.wait_resource_id = nil
            end
          end
        end

        # Select an eviction candidate, preferring TTL-eligible candidates first,
        # then capacity-pressure candidates (only when above the waiting
        # capacity threshold).
        #
        # @return [String, nil] The action key of the chosen candidate, or nil
        def select_eviction_candidate(now:, durable_slots:, reserve_slots:, min_wait_for_capacity_eviction:)
          @monitor.synchronize do
            waiting = @runs.values.select do |r|
              r.waiting? && !r.eviction_policy.nil?
            end
            return nil if waiting.empty?

            ttl_eligible = waiting.select do |r|
              policy = r.eviction_policy
              policy&.ttl && r.waiting_since && (now - r.waiting_since) >= policy.ttl
            end

            unless ttl_eligible.empty?
              chosen = ttl_eligible.min_by do |r|
                [r.eviction_policy ? r.eviction_policy.priority : 0, r.waiting_since || now]
              end
              ttl = chosen.eviction_policy&.ttl
              chosen.eviction_reason = DurableEvictionCache.build_eviction_reason(
                EvictionCause::TTL_EXCEEDED, chosen, ttl: ttl,
              )
              return chosen.key
            end

            return nil unless capacity_pressure?(durable_slots, reserve_slots, waiting.length)

            capacity_candidates = waiting.select do |r|
              r.eviction_policy&.allow_capacity_eviction &&
                r.waiting_since &&
                (now - r.waiting_since) >= min_wait_for_capacity_eviction
            end
            return nil if capacity_candidates.empty?

            chosen = capacity_candidates.min_by do |r|
              [r.eviction_policy ? r.eviction_policy.priority : 0, r.waiting_since || now]
            end
            chosen.eviction_reason = DurableEvictionCache.build_eviction_reason(
              EvictionCause::CAPACITY_PRESSURE, chosen,
            )
            chosen.key
          end
        end

        # Build a human-readable eviction reason string.
        def self.build_eviction_reason(cause, rec, ttl: nil)
          wait_desc = rec.wait_kind || "unknown"
          wait_desc = "#{wait_desc}(#{rec.wait_resource_id})" if rec.wait_resource_id

          case cause
          when EvictionCause::TTL_EXCEEDED
            ttl_str = ttl ? " (#{ttl}s)" : ""
            "Wait TTL#{ttl_str} exceeded while waiting on #{wait_desc}"
          when EvictionCause::CAPACITY_PRESSURE
            "Worker at capacity while waiting on #{wait_desc}"
          when EvictionCause::WORKER_SHUTDOWN
            "Worker shutdown while waiting on #{wait_desc}"
          else
            raise ArgumentError, "Unknown eviction cause: #{cause}"
          end
        end

        private

        def capacity_pressure?(durable_slots, reserve_slots, waiting_count)
          return false if durable_slots <= 0

          max_waiting = durable_slots - reserve_slots
          return false if max_waiting <= 0

          waiting_count >= max_waiting
        end
      end
    end
  end
end
