# frozen_string_literal: true

module Hatchet
  module WorkerRuntime
    # The highest satisfied_order received per (task_external_id, invocation_count). Reported in
    # worker status requests so that after a reconnect the server resumes ordered delivery where
    # this worker left off instead of waiting for orders it already sent.
    #
    # Not thread-safe; the durable event listener guards it with its own lock.
    class ReceivedSatisfiedOrders
      def initialize
        @highest_by_invocation = {}
      end

      def record(completed)
        return unless completed.has_satisfied_order?

        invocation = [completed.ref.durable_task_external_id, completed.ref.invocation_count]
        @highest_by_invocation[invocation] = [@highest_by_invocation.fetch(invocation, 0), completed.satisfied_order].max
      end

      # callback_keys are [task_external_id, invocation_count, branch_id, node_id]
      def to_proto_for(callback_keys)
        invocations = callback_keys.map { |(task_external_id, invocation_count, _, _)| [task_external_id, invocation_count] }

        invocations.uniq.filter_map do |(durable_task_external_id, invocation_count)|
          highest_received = @highest_by_invocation[[durable_task_external_id, invocation_count]]
          next if highest_received.nil?

          ::V1::DurableTaskReceivedSatisfiedOrder.new(
            durable_task_external_id: durable_task_external_id,
            invocation_count: invocation_count,
            highest_received_satisfied_order: highest_received,
          )
        end
      end

      def delete_through(durable_task_external_id, invocation_count)
        @highest_by_invocation.delete_if do |(task_external_id, recorded_invocation_count), _|
          task_external_id == durable_task_external_id && recorded_invocation_count <= invocation_count
        end
      end

      def clear
        @highest_by_invocation.clear
      end
    end
  end
end
