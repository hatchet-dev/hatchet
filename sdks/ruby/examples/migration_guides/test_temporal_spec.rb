# frozen_string_literal: true

require_relative "../spec_helper"
require_relative "temporal"

# These examples are not registered on the shared example worker, so everything
# here runs the task bodies directly rather than through the engine: `mock_run`
# for ordinary tasks, and a stubbed durable context for the durable ones.
RSpec.describe "MigrationGuideTemporal" do
  let(:order_input) { { "order_id" => "ord-123", "amount_cents" => 4999 } }

  describe "task definition" do
    it "charges an order and returns a structured output" do
      result = CHARGE_ORDER.mock_run(input: order_input)

      expect(result).to eq(
        "charged" => true,
        "transaction_id" => "txn-ord-123",
        "amount_cents" => 4999
      )
    end

    it "reports the attempt number from the retry count" do
      result = CHARGE_ORDER_WITH_RETRIES.mock_run(input: order_input, retry_count: 2)

      expect(result["attempt"]).to eq(3)
    end

    it "carries the retry and timeout options onto the task definition" do
      expect(CHARGE_ORDER_WITH_RETRIES.retries).to eq(10)
      expect(CHARGE_ORDER_WITH_RETRIES.backoff_factor).to eq(2.0)
      expect(CHARGE_ORDER_WITH_RETRIES.backoff_max_seconds).to eq(10)
      expect(CHARGE_ORDER_WITH_RETRIES.execution_timeout).to eq(30)
      expect(CHARGE_ORDER_WITH_RETRIES.schedule_timeout).to eq(600)
    end
  end

  describe "workflow as durable task" do
    it "runs validate, charge and fulfill in order" do
      called = []

      allow(VALIDATE_ORDER).to receive(:run) do |input|
        called << [:validate, input]
        { "order_id" => input["order_id"], "valid" => true }
      end
      allow(CHARGE_ORDER).to receive(:run) do |input|
        called << [:charge, input]
        { "charged" => true, "transaction_id" => "txn-ord-123", "amount_cents" => 4999 }
      end
      allow(FULFILL_ORDER).to receive(:run) do |input|
        called << [:fulfill, input]
        { "shipment_id" => "shp-ord-123-txn-ord-123" }
      end

      result = PROCESS_ORDER.call(order_input, durable_context)

      expect(called.map(&:first)).to eq(%i[validate charge fulfill])
      expect(called.last[1]["transaction_id"]).to eq("txn-ord-123")
      expect(result).to eq(
        "order_id" => "ord-123",
        "transaction_id" => "txn-ord-123",
        "shipment_id" => "shp-ord-123-txn-ord-123"
      )
    end

    it "does not charge an order that fails validation" do
      allow(VALIDATE_ORDER).to receive(:run).and_return({ "order_id" => "ord-123", "valid" => false })
      allow(CHARGE_ORDER).to receive(:run)

      expect { PROCESS_ORDER.call(order_input, durable_context) }
        .to raise_error(Hatchet::NonRetryableError, /ord-123 failed validation/)
      expect(CHARGE_ORDER).not_to have_received(:run)
    end

    it "is registered as a durable task" do
      expect(PROCESS_ORDER.durable).to be true
    end
  end

  describe "durable task with sleep" do
    # The real sleep is three days, so the context is stubbed and only the
    # requested duration is asserted on.
    it "emails, sleeps for three days, then emails again" do
      ctx = durable_context
      allow(SEND_WELCOME_EMAIL).to receive(:run).and_return({ "message_id" => "welcome:u-1" })
      allow(SEND_FOLLOWUP_EMAIL).to receive(:run).and_return({ "message_id" => "followup:u-1" })

      result = ONBOARDING_FLOW.call({ "user_id" => "u-1" }, ctx)

      expect(ctx).to have_received(:sleep_for).with(duration: 259_200).once
      expect(result).to eq(
        "user_id" => "u-1",
        "welcome_message_id" => "welcome:u-1",
        "followup_message_id" => "followup:u-1"
      )
    end

    it "gives the run an execution timeout that outlasts its sleep" do
      expect(ONBOARDING_FLOW.execution_timeout).to be > 259_200
    end
  end

  describe "durable sleep" do
    it "sleeps for one day on the durable context" do
      ctx = durable_context

      result = WAIT_A_DAY.call(order_input, ctx)

      expect(ctx).to have_received(:sleep_for).with(duration: 86_400)
      expect(result["order_id"]).to eq("ord-123")
    end
  end

  describe "durable event wait" do
    it "waits on a correlated approval event before fulfilling" do
      waited = nil
      ctx = instance_double(Hatchet::DurableContext)
      allow(ctx).to receive(:wait_for) { |key, condition| waited = [key, condition] }
      allow(FULFILL_ORDER).to receive(:run).and_return({ "shipment_id" => "shp-1" })

      APPROVAL_FLOW.call(order_input.merge("correlation_id" => "c-abc"), ctx)

      key, condition = waited
      expect(key).to eq("event")
      expect(condition).to be_a(Hatchet::UserEventCondition)
      expect(condition.event_key).to eq("approval:granted")
      expect(condition.expression).to eq("input.correlation_id == 'c-abc'")
      expect(FULFILL_ORDER).to have_received(:run)
    end
  end

  describe "fan out children" do
    it "spawns one child run per item in a single bulk call" do
      allow(PROCESS_ITEM).to receive(:run_many) do |items|
        items.map { |item| { "item" => item[:input]["item"], "processed" => true } }
      end

      result = FAN_OUT_ORDER_ITEMS.call({ "items" => %w[a b c] }, durable_context)

      expect(PROCESS_ITEM).to have_received(:run_many).once
      expect(result["results"].map { |r| r["item"] }).to eq(%w[a b c])
    end

    it "processes a single item" do
      expect(PROCESS_ITEM.mock_run(input: { "item" => "a" }))
        .to eq("item" => "a", "processed" => true)
    end
  end

  describe "cron declaration" do
    it "declares the cron on the workflow" do
      expect(WEEKLY_REPORT.on_crons).to eq(["0 9 * * 1"])
    end

    it "generates a report" do
      result = WEEKLY_REPORT.tasks[:generate].mock_run(input: { "kind" => "weekly" })

      expect(result["kind"]).to eq("weekly")
    end
  end

  describe "concurrency and rate limits" do
    it "limits concurrent runs per customer" do
      concurrency = SYNC_CUSTOMER.concurrency

      expect(concurrency.expression).to eq("input.customer_id")
      expect(concurrency.max_runs).to eq(1)
      expect(concurrency.limit_strategy).to eq(:cancel_in_progress)
    end

    it "consumes a unit of the global static rate limit" do
      rate_limit = CALL_MODEL.rate_limits.first

      expect(rate_limit.static_key).to eq("openai")
      expect(rate_limit.units).to eq(1)
    end
  end

  describe "DAG workflow" do
    it "chains validate, charge and fulfill through parent outputs" do
      validated = VALIDATE.mock_run(input: order_input)
      expect(validated).to eq("order_id" => "ord-123", "valid" => true)

      charged = CHARGE.mock_run(
        input: order_input,
        parent_outputs: { "validate" => validated }
      )
      expect(charged).to eq("transaction_id" => "txn-ord-123", "amount_cents" => 4999)

      fulfilled = ORDER_WORKFLOW.tasks[:fulfill].mock_run(
        input: order_input,
        parent_outputs: { "charge" => charged }
      )
      expect(fulfilled).to eq("shipment_id" => "shp-ord-123-txn-ord-123")
    end

    it "refuses to charge when the upstream validation failed" do
      expect do
        CHARGE.mock_run(
          input: order_input,
          parent_outputs: { "validate" => { "order_id" => "ord-123", "valid" => false } }
        )
      end.to raise_error(Hatchet::NonRetryableError, /ord-123 failed validation/)
    end

    it "declares the graph edges" do
      expect(CHARGE.parents).to eq([VALIDATE])
      expect(ORDER_WORKFLOW.tasks[:fulfill].parents).to eq([CHARGE])
    end
  end

  describe "context logging" do
    it "writes a line to the run log sink" do
      ctx = instance_double(Hatchet::Context)
      allow(ctx).to receive(:log)

      result = LOG_CHARGE.call(order_input, ctx)

      expect(ctx).to have_received(:log).with("charging order ord-123")
      expect(result).to eq("order_id" => "ord-123", "logged" => true)
    end
  end

  def durable_context
    ctx = instance_double(Hatchet::DurableContext)
    allow(ctx).to receive(:sleep_for)
    allow(ctx).to receive(:wait_for)
    ctx
  end
end
