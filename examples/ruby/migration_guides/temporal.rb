# frozen_string_literal: true

# Hatchet side of the "From Temporal to Hatchet" migration guide. Every snippet
# rendered in that guide's Ruby tab comes from a marked region in this file.

# > Hatchet worker
require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

def main
  worker = HATCHET.worker(
    "order-worker",
    slots: 10,
    workflows: [VALIDATE_ORDER, CHARGE_ORDER, FULFILL_ORDER, PROCESS_ORDER]
  )
  worker.start
end

# Stand-ins for the systems a real order pipeline would talk to. Everything
# else in this file is Hatchet code that runs as written.
module Payments
  def self.charge(order_id, amount_cents)
    { "transaction_id" => "txn-#{order_id}", "amount_cents" => amount_cents }
  end
end

module Fulfillment
  def self.ship(order_id, transaction_id)
    "shp-#{order_id}-#{transaction_id}"
  end
end

module Mailer
  def self.deliver(template, user_id)
    "#{template}:#{user_id}"
  end
end

VALIDATE_ORDER = HATCHET.task(name: "validate-order", execution_timeout: 30) do |input, _ctx|
  { "order_id" => input["order_id"], "valid" => input["amount_cents"].to_i.positive? }
end

# > Hatchet task definition
CHARGE_ORDER = HATCHET.task(name: "charge-order", execution_timeout: 30) do |input, _ctx|
  charge = Payments.charge(input["order_id"], input["amount_cents"])

  {
    "charged" => true,
    "transaction_id" => charge["transaction_id"],
    "amount_cents" => charge["amount_cents"]
  }
end

FULFILL_ORDER = HATCHET.task(name: "fulfill-order", execution_timeout: 30) do |input, _ctx|
  { "shipment_id" => Fulfillment.ship(input["order_id"], input["transaction_id"]) }
end

# > Hatchet workflow as durable task
PROCESS_ORDER = HATCHET.durable_task(name: "ProcessOrder", execution_timeout: 300) do |input, _ctx|
  validated = VALIDATE_ORDER.run(input)
  raise Hatchet::NonRetryableError, "order #{input["order_id"]} failed validation" unless validated["valid"]

  charged = CHARGE_ORDER.run(input)
  fulfilled = FULFILL_ORDER.run(input.merge("transaction_id" => charged["transaction_id"]))

  {
    "order_id" => input["order_id"],
    "transaction_id" => charged["transaction_id"],
    "shipment_id" => fulfilled["shipment_id"]
  }
end

SEND_WELCOME_EMAIL = HATCHET.task(name: "send-welcome-email") do |input, _ctx|
  { "message_id" => Mailer.deliver("welcome", input["user_id"]) }
end

SEND_FOLLOWUP_EMAIL = HATCHET.task(name: "send-followup-email") do |input, _ctx|
  { "message_id" => Mailer.deliver("followup", input["user_id"]) }
end

# > Hatchet durable task with sleep
# The execution timeout has to cover the whole wall-clock span of the run,
# sleeps included, so a three-day sleep needs a timeout longer than three days.
ONBOARDING_FLOW = HATCHET.durable_task(
  name: "OnboardingFlow",
  execution_timeout: 604_800
) do |input, ctx|
  welcome = SEND_WELCOME_EMAIL.run(input)

  ctx.sleep_for(duration: 259_200)

  followup = SEND_FOLLOWUP_EMAIL.run(input)

  {
    "user_id" => input["user_id"],
    "welcome_message_id" => welcome["message_id"],
    "followup_message_id" => followup["message_id"]
  }
end

# > Hatchet task invocation
def enqueue_order(order_id, amount_cents)
  ref = PROCESS_ORDER.run_no_wait({ "order_id" => order_id, "amount_cents" => amount_cents })

  # The run is enqueued and its id is available before any work starts.
  puts "enqueued run #{ref.workflow_run_id}"

  ref.result
end

# > Hatchet retries and timeouts
CHARGE_ORDER_WITH_RETRIES = HATCHET.task(
  name: "charge-order-with-retries",
  # Hatchet counts retries, not attempts: this is 11 attempts in total.
  retries: 10,
  backoff_factor: 2.0,
  backoff_max_seconds: 10,
  execution_timeout: 30,
  schedule_timeout: 600
) do |input, ctx|
  charge = Payments.charge(input["order_id"], input["amount_cents"])

  { "transaction_id" => charge["transaction_id"], "attempt" => ctx.retry_count + 1 }
end

# > Hatchet durable sleep
# A sleeping durable task is evicted and releases its worker slot, so the wait
# costs no capacity however long it is.
WAIT_A_DAY = HATCHET.durable_task(name: "WaitADay", execution_timeout: 90_000) do |input, ctx|
  ctx.sleep_for(duration: 86_400)

  { "order_id" => input["order_id"], "resumed_at" => Time.now.utc.to_i }
end

# > Hatchet event push
def grant_approval(correlation_id)
  HATCHET.events.push(
    "approval:granted",
    { "correlation_id" => correlation_id }
  )
end

# > Hatchet durable event wait
APPROVAL_FLOW = HATCHET.durable_task(name: "ApprovalFlow", execution_timeout: 86_400) do |input, ctx|
  # The CEL expression does the job a Temporal workflow id used to do, so
  # correlate on an opaque id you generate rather than user-supplied text.
  ctx.wait_for(
    "event",
    Hatchet::UserEventCondition.new(
      event_key: "approval:granted",
      expression: "input.correlation_id == '#{input["correlation_id"]}'"
    )
  )

  FULFILL_ORDER.run(input)
end

# > Hatchet fan out children
PROCESS_ITEM = HATCHET.task(name: "process-item") do |input, _ctx|
  { "item" => input["item"], "processed" => true }
end

FAN_OUT_ORDER_ITEMS = HATCHET.durable_task(name: "FanOutOrderItems", execution_timeout: 300) do |input, _ctx|
  items = input["items"] || []

  # Ruby's trigger methods are synchronous, so fan out with the bulk API rather
  # than one call per child.
  results = PROCESS_ITEM.run_many(
    items.map { |item| PROCESS_ITEM.create_bulk_run_item(input: { "item" => item }) }
  )

  { "results" => results }
end

# > Hatchet cron declaration
# Ruby declares crons on a workflow rather than on a standalone task.
WEEKLY_REPORT = HATCHET.workflow(
  name: "WeeklyReport",
  on_crons: ["0 9 * * 1"]
)

WEEKLY_REPORT.task(:generate) do |input, _ctx|
  { "kind" => input["kind"] || "weekly", "generated_at" => Time.now.utc.to_i }
end

# > Hatchet runtime schedules
def schedule_weekly_report
  HATCHET.cron.create(
    workflow_name: "WeeklyReport",
    cron_name: "weekly-report-acme",
    expression: "0 9 * * 1",
    input: { "kind" => "weekly" }
  )

  HATCHET.scheduled.create(
    workflow_name: "WeeklyReport",
    trigger_at: Time.now + 86_400,
    input: { "kind" => "weekly" },
    additional_metadata: { "customer_id" => "acme" }
  )
end

# > Hatchet concurrency and rate limits
# One in-flight run per customer, newest cancels the oldest.
SYNC_CUSTOMER = HATCHET.workflow(
  name: "SyncCustomer",
  concurrency: Hatchet::ConcurrencyExpression.new(
    expression: "input.customer_id",
    max_runs: 1,
    limit_strategy: :cancel_in_progress
  )
)

SYNC_CUSTOMER.task(:sync) do |input, _ctx|
  { "customer_id" => input["customer_id"], "synced" => true }
end

# A global budget shared by every worker, not per-process.
CALL_MODEL = HATCHET.task(
  name: "call-model",
  rate_limits: [Hatchet::RateLimit.new(static_key: "openai", units: 1)]
) do |input, _ctx|
  { "prompt" => input["prompt"], "completion" => "..." }
end

# The static key above has to exist before any task consumes it.
def declare_model_rate_limit
  HATCHET.rate_limits.put(key: "openai", limit: 100, duration: :minute)
end

# > Hatchet DAG workflow
ORDER_WORKFLOW = HATCHET.workflow(name: "ProcessOrderDag")

VALIDATE = ORDER_WORKFLOW.task(:validate, execution_timeout: 30) do |input, _ctx|
  { "order_id" => input["order_id"], "valid" => input["amount_cents"].to_i.positive? }
end

CHARGE = ORDER_WORKFLOW.task(:charge, parents: [VALIDATE], execution_timeout: 30) do |input, ctx|
  validated = ctx.task_output(VALIDATE)
  raise Hatchet::NonRetryableError, "order #{input["order_id"]} failed validation" unless validated["valid"]

  charge = Payments.charge(validated["order_id"], input["amount_cents"])

  { "transaction_id" => charge["transaction_id"], "amount_cents" => charge["amount_cents"] }
end

ORDER_WORKFLOW.task(:fulfill, parents: [CHARGE], execution_timeout: 30) do |input, ctx|
  charged = ctx.task_output(CHARGE)

  { "shipment_id" => Fulfillment.ship(input["order_id"], charged["transaction_id"]) }
end

# > Hatchet context logging
LOG_CHARGE = HATCHET.task(name: "log-charge") do |input, ctx|
  ctx.log("charging order #{input["order_id"]}")

  { "order_id" => input["order_id"], "logged" => true }
end

main if __FILE__ == $PROGRAM_NAME
