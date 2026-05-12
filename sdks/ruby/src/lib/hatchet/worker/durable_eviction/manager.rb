# frozen_string_literal: true

require "monitor"

require_relative "cache"

module Hatchet
  module WorkerRuntime
    module DurableEviction
      # Configuration for the background eviction loop.
      class DurableEvictionConfig
        # @return [Float] Seconds between eviction checks.
        attr_reader :check_interval
        # @return [Integer] Slots to reserve from capacity-eviction decisions.
        attr_reader :reserve_slots
        # @return [Float] Minimum seconds a run must have been waiting before it
        #   becomes eligible for capacity-based eviction.
        attr_reader :min_wait_for_capacity_eviction

        def initialize(check_interval: 1.0, reserve_slots: 0, min_wait_for_capacity_eviction: 10.0)
          @check_interval = check_interval
          @reserve_slots = reserve_slots
          @min_wait_for_capacity_eviction = min_wait_for_capacity_eviction
          freeze
        end
      end

      DEFAULT_DURABLE_EVICTION_CONFIG = DurableEvictionConfig.new

      # Orchestrates durable-task eviction.
      #
      # Runs a background thread that periodically selects an eviction candidate
      # from the cache, asks the server to evict it, and then interrupts the
      # local task thread.
      #
      # Mirrors :class:`hatchet_sdk.worker.durable_eviction.manager.DurableEvictionManager`.
      class DurableEvictionManager
        # @return [DurableEvictionCache]
        attr_reader :cache

        # @param durable_slots [Integer]
        # @param cancel_local [Proc] Called with the action key when the manager
        #   decides to evict a local run (invoked after the server ACK).
        # @param request_eviction_with_ack [Proc] Called with (action_key, DurableRunRecord)
        #   to send the eviction RPC to the server and block until acknowledged.
        # @param config [DurableEvictionConfig]
        # @param cache [DurableEvictionCache, nil]
        # @param logger [Logger, nil]
        def initialize(
          durable_slots:,
          cancel_local:,
          request_eviction_with_ack:,
          config: DEFAULT_DURABLE_EVICTION_CONFIG,
          cache: nil,
          logger: nil
        )
          @durable_slots = durable_slots
          @cancel_local = cancel_local
          @request_eviction_with_ack = request_eviction_with_ack
          @config = config
          @cache = cache || DurableEvictionCache.new
          @logger = logger

          @thread = nil
          @tick_monitor = Monitor.new
          @stopped = false
        end

        # Start the background eviction ticker. Idempotent.
        def start
          return if @thread&.alive?

          @stopped = false
          @thread = Thread.new { run_loop }
        end

        # Signal the background thread to stop. Does not join.
        def stop
          @stopped = true
          thread = @thread
          return unless thread&.alive?

          begin
            thread.wakeup
          rescue ThreadError
            nil
          end
        end

        # Register a new durable run invocation. Takes the current time from the
        # system clock.
        def register_run(key, step_run_id:, invocation_count:, eviction_policy:)
          @cache.register_run(
            key,
            step_run_id: step_run_id,
            invocation_count: invocation_count,
            now: now,
            eviction_policy: eviction_policy,
          )
        end

        # Unregister a durable run invocation.
        def unregister_run(key)
          @cache.unregister_run(key)
        end

        # Mark the run as waiting (increments the wait counter).
        def mark_waiting(key, wait_kind:, resource_id:)
          @cache.mark_waiting(key, now: now, wait_kind: wait_kind, resource_id: resource_id)
        end

        # Mark the run as active (decrements the wait counter).
        def mark_active(key)
          @cache.mark_active(key, now: now)
        end

        # Handle a server-initiated eviction notification for a stale invocation.
        def handle_server_eviction(step_run_id, invocation_count)
          key = @cache.find_key_by_step_run_id(step_run_id)
          return unless key

          rec = @cache.get(key)
          return if rec && rec.invocation_count != invocation_count

          @logger&.info(
            "DurableEvictionManager: server-initiated eviction for " \
            "step_run_id=#{step_run_id} invocation_count=#{invocation_count}",
          )
          evict_run(key)
        end

        # Evict every currently-waiting durable run. Used during graceful shutdown.
        #
        # @return [Integer] number of runs evicted
        def evict_all_waiting
          stop

          waiting = @cache.all_waiting
          evicted = 0

          waiting.each do |rec|
            rec.eviction_reason = DurableEvictionCache.build_eviction_reason(
              EvictionCause::WORKER_SHUTDOWN, rec,
            )

            @logger&.debug(
              "DurableEvictionManager: shutdown-evicting durable run " \
              "step_run_id=#{rec.step_run_id} wait_kind=#{rec.wait_kind} " \
              "resource_id=#{rec.wait_resource_id}",
            )

            begin
              @request_eviction_with_ack.call(rec.key, rec)
            rescue StandardError => e
              @logger&.error(
                "DurableEvictionManager: failed to send eviction for " \
                "step_run_id=#{rec.step_run_id}: #{e.class}: #{e.message}",
              )
            end

            # Always cancel locally even if the server ACK failed, so the
            # future settles and shutdown doesn't hang.
            evict_run(rec.key)
            evicted += 1
          end

          evicted
        end

        private

        def evict_run(key)
          @cancel_local.call(key)
          unregister_run(key)
        end

        def run_loop
          until @stopped
            sleep @config.check_interval
            break if @stopped

            tick_safe
          end
        rescue StandardError => e
          @logger&.error("DurableEvictionManager: run_loop exited: #{e.class}: #{e.message}")
        end

        def tick_safe
          tick
        rescue StandardError => e
          @logger&.error("DurableEvictionManager: error in eviction loop: #{e.class}: #{e.message}")
        end

        def tick
          @tick_monitor.synchronize do
            evicted_this_tick = []

            loop do
              key = @cache.select_eviction_candidate(
                now: now,
                durable_slots: @durable_slots,
                reserve_slots: @config.reserve_slots,
                min_wait_for_capacity_eviction: @config.min_wait_for_capacity_eviction,
              )
              return if key.nil?
              return if evicted_this_tick.include?(key)

              evicted_this_tick << key

              rec = @cache.get(key)
              next if rec.nil?
              next if rec.eviction_policy.nil?

              @logger&.debug(
                "DurableEvictionManager: evicting durable run " \
                "step_run_id=#{rec.step_run_id} wait_kind=#{rec.wait_kind} " \
                "resource_id=#{rec.wait_resource_id} ttl=#{rec.eviction_policy.ttl} " \
                "capacity_allowed=#{rec.eviction_policy.allow_capacity_eviction}",
              )

              @request_eviction_with_ack.call(key, rec)
              evict_run(key)
            end
          end
        end

        def now
          Time.now
        end
      end
    end
  end
end
