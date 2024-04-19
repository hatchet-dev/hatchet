from datetime import datetime, timedelta

from dotenv import load_dotenv

from hatchet_sdk import Context, Hatchet

load_dotenv()

hatchet = Hatchet(debug=True)


@hatchet.workflow(on_events=["printer:schedule"])
class PrintSchedule:
    @hatchet.step()
    def schedule(self, context: Context):
        now = datetime.now()
        print(f"the time is \t {now.strftime('%H:%M:%S')}")
        future_time = now + timedelta(seconds=15)
        print(f"scheduling for \t {future_time.strftime('%H:%M:%S')}")

        hatchet.client.admin.schedule_workflow(
            "PrintPrinter", [future_time], context.workflow_input()
        )


@hatchet.workflow()
class PrintPrinter:
    @hatchet.step()
    def step1(self, context: Context):
        now = datetime.now()
        print(f"printed at \t {now.strftime('%H:%M:%S')}")
        print(f"message \t {context.workflow_input()['message']}")


worker = hatchet.worker("test-worker", max_runs=4)
worker.register_workflow(PrintSchedule())
worker.register_workflow(PrintPrinter())

worker.start()
