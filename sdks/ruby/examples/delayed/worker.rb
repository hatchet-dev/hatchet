# frozen_string_literal: true

require "hatchet-sdk"

HATCHET = Hatchet::Client.new(debug: true)

PRINT_SCHEDULE_WF = hatchet.workflow(name: "PrintScheduleWorkflow")
PRINT_PRINTER_WF = hatchet.workflow(name: "PrintPrinterWorkflow")

PRINT_SCHEDULE_WF.task(:schedule) do |input, ctx|
  now = Time.now.utc
  puts "the time is \t #{now.strftime('%H:%M:%S')}"
  future_time = now + 15
  puts "scheduling for \t #{future_time.strftime('%H:%M:%S')}"

  PRINT_PRINTER_WF.schedule(future_time, input: input)
end

PRINT_SCHEDULE_WF.task(:step1) do |input, ctx|
  now = Time.now.utc
  puts "printed at \t #{now.strftime('%H:%M:%S')}"
  puts "message \t #{input['message']}"
end

def main
  worker = hatchet.worker(
    "delayed-worker", slots: 4, workflows: [PRINT_SCHEDULE_WF, PRINT_PRINTER_WF]
  )
  worker.start
end

main if __FILE__ == $PROGRAM_NAME
