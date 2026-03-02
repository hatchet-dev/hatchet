# frozen_string_literal: true

# > LoggingWorkflow

require "hatchet-sdk"
require "logger"

logger = Logger.new($stdout)
logger.level = Logger::INFO

HATCHET = Hatchet::Client.new(debug: true) unless defined?(HATCHET)

LOGGING_WORKFLOW = HATCHET.workflow(name: "LoggingWorkflow")

LOGGING_WORKFLOW.task(:root_logger) do |input, ctx|
  12.times do |i|
    logger.info("executed step1 - #{i}")
    logger.info({ "step1" => "step1" }.inspect)

    sleep 0.1
  end

  { "status" => "success" }
end


# > ContextLogger
LOGGING_WORKFLOW.task(:context_logger) do |input, ctx|
  12.times do |i|
    ctx.log("executed step1 - #{i}")
    ctx.log({ "step1" => "step1" }.inspect)

    sleep 0.1
  end

  { "status" => "success" }
end


def main
  worker = HATCHET.worker("logger-worker", slots: 5, workflows: [LOGGING_WORKFLOW])
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
