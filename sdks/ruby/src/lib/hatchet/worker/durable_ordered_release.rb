# frozen_string_literal: true

module Hatchet
  module WorkerRuntime
    # Ordered release of ``entry_completed`` responses for durable task
    # invocations, mixed into {DurableEventListener}.
    #
    # The engine stamps a contiguous per-task ``satisfied_order`` on durable
    # event log entries as they are satisfied. Releasing completions to user
    # code strictly in that order (gated on the previously woken continuation
    # parking again) makes the wake order deterministic, so replays reproduce
    # the originally recorded interleaving instead of racing into spurious
    # non-determinism errors.
    module DurableOrderedRelease
      # How long the ordered-release gate stays closed waiting for a woken
      # continuation to park (register its next awaited entry) before being
      # forced open with a loud warning.
      DEFAULT_PARK_TIMEOUT_SECONDS = 5.0
      # How long a hole in the satisfied-order sequence may persist (while
      # later completions are held) before the invocation's waiters are failed
      # with a NonDeterminismError instead of hanging.
      DEFAULT_GAP_TIMEOUT_SECONDS = 60.0

      # Per-invocation state serializing the release of ordered
      # ``entry_completed`` responses. Completions are released to user code in
      # ``satisfied_order``; after a release wakes a parked continuation,
      # further releases are held until that continuation parks again
      # (registers its next awaited entry), or the park timeout elapses.
      #
      # @!attribute held
      #   @return [Hash{Integer => Array(Array, Hash)}] order => [callback_key, result]
      # @!attribute released
      #   @return [Integer] highest satisfied order released so far
      # @!attribute wakes
      #   @return [Integer] continuations woken by a gated release which have
      #     not yet parked; the gate is open iff zero
      # @!attribute wake_since
      #   @return [Float, nil] monotonic time wakes last left zero
      # @!attribute gap_since
      #   @return [Float, nil] monotonic time held first became blocked on a
      #     missing order; nil when not blocked
      OrderedReleaseGate = Struct.new(:held, :released, :wakes, :wake_since, :gap_since, keyword_init: true)

      # The durable task function for the given invocation returned (or
      # otherwise has no running continuations): release any gate held on its
      # behalf so remaining ordered completions flow to late registrations.
      def notify_invocation_quiesced(durable_task_external_id, invocation_count)
        gate_key = [durable_task_external_id, invocation_count]

        @mu.synchronize do
          gate = @gates[gate_key]
          next unless gate

          gate.wakes = 0
          pump_gate(gate_key, gate)
        end
      end

      private

      # Hand a completion to a registered waiter, or buffer it for late
      # registration. Returns true if a parked continuation was woken.
      def deliver_completion(key, result)
        @mu.synchronize do
          queue = @pending_callbacks.delete(key)
          if queue
            queue << [:ok, result]
            true
          else
            @buffered_completions[key] = [Time.now, result]
            false
          end
        end
      end

      # Route an ordered completion through the invocation's gate.
      def handle_ordered_completion(gate_key, order, key, result)
        @mu.synchronize do
          gate = (@gates[gate_key] ||= OrderedReleaseGate.new(held: {}, released: 0, wakes: 0))

          if order <= gate.released
            # re-delivery of an already-released completion (e.g. after
            # reconnect): bypass the gate.
            deliver_completion(key, result)
          else
            gate.held[order] = [key, result]
            pump_gate(gate_key, gate)
          end
        end
      end

      # Release contiguously ordered completions while the gate is open.
      # Callers must hold @mu.
      def pump_gate(_gate_key, gate)
        while gate.wakes.zero?
          held = gate.held.delete(gate.released + 1)
          break unless held

          gate.released += 1

          key, result = held
          next unless deliver_completion(key, result)

          # the release woke a parked continuation: hold further releases
          # until it parks again. if nobody was waiting, the completion was
          # buffered for a continuation that is still running; keep pumping so
          # a parked continuation awaiting a later order is not deadlocked.
          gate.wakes += 1
          gate.wake_since = monotonic_now
        end

        if !gate.held.empty? && gate.wakes.zero?
          gate.gap_since ||= monotonic_now
        else
          gate.gap_since = nil
        end
      end

      # A continuation of the given invocation parked (registered its next
      # awaited entry without a buffered result): open the gate for the next
      # ordered release.
      def notify_parked(gate_key)
        @mu.synchronize do
          gate = @gates[gate_key]
          next unless gate

          gate.wakes -= 1 if gate.wakes.positive?
          pump_gate(gate_key, gate)
        end
      end

      # Enforce the park and gap timeouts on all ordered-release gates.
      def sweep_gates
        failures = []

        @mu.synchronize do
          now = monotonic_now

          @gates.keys.each do |gate_key| # rubocop:disable Style/HashEachMethods
            gate = @gates[gate_key]

            if gate.wakes.positive? && gate.wake_since && now - gate.wake_since > @park_timeout_s
              @logger&.warn(
                "durable task #{gate_key[0]} (invocation #{gate_key[1]}): continuation did not " \
                "park within #{@park_timeout_s}s after a gated release; forcing the completion " \
                "gate open. durable task code should not perform unrecorded blocking work " \
                "between durable operations",
              )
              gate.wakes = 0
              pump_gate(gate_key, gate)
            end

            next if gate.held.empty? || gate.wakes.positive? || gate.gap_since.nil?
            next unless now - gate.gap_since > @gap_timeout_s

            missing_order = gate.released + 1
            exc = Hatchet::NonDeterminismError.new(
              "completion with satisfied order #{missing_order} was never delivered while " \
              "later completions #{gate.held.keys.sort} arrived; the recorded history likely " \
              "diverged from the current code",
              task_external_id: gate_key[0],
              invocation_count: gate_key[1],
              node_id: missing_order,
            )
            @logger&.error(exc.message)

            @gates.delete(gate_key)
            failures << [gate_key, exc]
          end
        end

        failures.each { |gate_key, exc| fail_invocation_waiters(gate_key, exc) }
      end

      # Deliver an error to every pending callback and event ack belonging to
      # the given invocation.
      def fail_invocation_waiters(gate_key, exc)
        @mu.synchronize do
          @pending_callbacks.each_key do |k|
            next unless k[0] == gate_key[0] && k[1] == gate_key[1]

            @pending_callbacks.delete(k)&.<<([:err, exc])
          end

          @pending_event_acks.delete(gate_key)&.<<([:err, exc])
        end
      end

      def monotonic_now
        Process.clock_gettime(Process::CLOCK_MONOTONIC)
      end
    end
  end
end
