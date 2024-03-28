import time

from dotenv import load_dotenv

from hatchet_sdk import Context, Hatchet

load_dotenv()

hatchet = Hatchet()


@hatchet.workflow(on_events=["user:create"], schedule_timeout="10m")
class LoggingWorkflow:
    @hatchet.step()
    def logger(self, context: Context):

        for i in range(1000):
            context.log(f"Logging message {i}")

        return {
            "step1": "completed",
        }


workflow = LoggingWorkflow()
worker = hatchet.worker("logging-worker-py")
worker.register_workflow(workflow)

worker.start()
