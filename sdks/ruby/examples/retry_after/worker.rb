# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

# > Retry after an upstream-provided delay
RETRY_AFTER_UPSTREAM_DELAY = HATCHET.task(name: "RetryAfterUpstreamDelay", retries: 5) do |input, ctx|
  if ctx.retry_count < input["failing_attempts"]
    raise Hatchet::RetryAfterError.new("upstream rate limited", after: 2)
  end

  { "attempt" => ctx.retry_count }
end

# !!

# > Exponential backoff with full jitter
RETRY_AFTER_EXPONENTIAL_BACKOFF = HATCHET.task(name: "RetryAfterExponentialBackoff", retries: 3) do |input, ctx|
  if ctx.retry_count < input["failing_attempts"]
    raise Hatchet::RetryAfterError.new("upstream unavailable", after: rand * [2**ctx.retry_count, 60].min)
  end

  { "attempt" => ctx.retry_count }
end

# !!

def main
  worker = HATCHET.worker(
    "retry-after-worker",
    workflows: [RETRY_AFTER_UPSTREAM_DELAY, RETRY_AFTER_EXPONENTIAL_BACKOFF]
  )
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
