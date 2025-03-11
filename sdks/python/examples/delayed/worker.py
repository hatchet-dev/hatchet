from datetime import datetime, timedelta

from pydantic import BaseModel

from hatchet_sdk import BaseWorkflow, Context, Hatchet

hatchet = Hatchet(debug=True)


class PrinterInput(BaseModel):
    message: str


print_schedule_wf = hatchet.declare_workflow(
    on_events=["printer:schedule"], input_validator=PrinterInput
)
print_printer_wf = hatchet.declare_workflow(input_validator=PrinterInput)


class PrintSchedule(BaseWorkflow):
    config = print_schedule_wf.config

    @hatchet.step()
    def schedule(self, context: Context) -> None:
        now = datetime.now()
        print(f"the time is \t {now.strftime('%H:%M:%S')}")
        future_time = now + timedelta(seconds=15)
        print(f"scheduling for \t {future_time.strftime('%H:%M:%S')}")

        input = print_schedule_wf.get_workflow_input(context)

        print_printer_wf.schedule([future_time], input=input)


class PrintPrinter(BaseWorkflow):
    config = print_printer_wf.config

    @hatchet.step()
    def step1(self, context: Context) -> None:
        now = datetime.now()
        print(f"printed at \t {now.strftime('%H:%M:%S')}")
        print(f"message \t {print_printer_wf.get_workflow_input(context).message}")


def main() -> None:
    worker = hatchet.worker("delayed-worker", max_runs=4)
    worker.register_workflow(PrintSchedule())
    worker.register_workflow(PrintPrinter())

    worker.start()


if __name__ == "__main__":
    main()
