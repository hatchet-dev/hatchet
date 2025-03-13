from datetime import datetime, timedelta

from pydantic import BaseModel

from hatchet_sdk import Context, Hatchet

hatchet = Hatchet(debug=True)


class PrinterInput(BaseModel):
    message: str


print_schedule_wf = hatchet.workflow(
    name="PrintScheduleWorkflow",
    input_validator=PrinterInput,
)
print_printer_wf = hatchet.workflow(
    name="PrintPrinterWorkflow", input_validator=PrinterInput
)


@print_schedule_wf.task()
def schedule(input: PrinterInput, ctx: Context) -> None:
    now = datetime.now()
    print(f"the time is \t {now.strftime('%H:%M:%S')}")
    future_time = now + timedelta(seconds=15)
    print(f"scheduling for \t {future_time.strftime('%H:%M:%S')}")

    print_printer_wf.schedule([future_time], input=input)


@print_schedule_wf.task()
def step1(input: PrinterInput, ctx: Context) -> None:
    now = datetime.now()
    print(f"printed at \t {now.strftime('%H:%M:%S')}")
    print(f"message \t {input.message}")


def main() -> None:
    worker = hatchet.worker(
        "delayed-worker", slots=4, workflows=[print_schedule_wf, print_printer_wf]
    )

    worker.start()


if __name__ == "__main__":
    main()
