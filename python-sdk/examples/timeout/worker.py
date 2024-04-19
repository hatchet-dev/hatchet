from dotenv import load_dotenv

from hatchet_sdk import Hatchet

load_dotenv()

hatchet = Hatchet(debug=True)


@hatchet.workflow(on_events=["user:create"])
class TimeoutWorkflow:
    def __init__(self):
        self.my_value = "test"

    @hatchet.step(timeout="4s")
    def timeout(self, context):
        try:
            print("started step2")
            context.sleep(5)
            print("finished step2")
        except Exception as e:
            print("caught an exception: " + str(e))
            raise e


workflow = TimeoutWorkflow()
worker = hatchet.worker("timeout-worker", max_runs=4)
worker.register_workflow(workflow)

worker.start()
