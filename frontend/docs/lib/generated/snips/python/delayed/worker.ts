import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "python",
  "content": "from datetime import datetime, timedelta\n\nfrom pydantic import BaseModel\n\nfrom hatchet_sdk import Context, Hatchet\n\nhatchet = Hatchet(debug=True)\n\n\nclass PrinterInput(BaseModel):\n    message: str\n\n\nprint_schedule_wf = hatchet.workflow(\n    name=\"PrintScheduleWorkflow\",\n    input_validator=PrinterInput,\n)\nprint_printer_wf = hatchet.workflow(\n    name=\"PrintPrinterWorkflow\", input_validator=PrinterInput\n)\n\n\n@print_schedule_wf.task()\ndef schedule(input: PrinterInput, ctx: Context) -> None:\n    now = datetime.now()\n    print(f\"the time is \\t {now.strftime('%H:%M:%S')}\")\n    future_time = now + timedelta(seconds=15)\n    print(f\"scheduling for \\t {future_time.strftime('%H:%M:%S')}\")\n\n    print_printer_wf.schedule(future_time, input=input)\n\n\n@print_schedule_wf.task()\ndef step1(input: PrinterInput, ctx: Context) -> None:\n    now = datetime.now()\n    print(f\"printed at \\t {now.strftime('%H:%M:%S')}\")\n    print(f\"message \\t {input.message}\")\n\n\ndef main() -> None:\n    worker = hatchet.worker(\n        \"delayed-worker\", slots=4, workflows=[print_schedule_wf, print_printer_wf]\n    )\n\n    worker.start()\n\n\nif __name__ == \"__main__\":\n    main()\n",
  "source": "out/python/delayed/worker.py",
  "blocks": {},
  "highlights": {}
};

export default snippet;
