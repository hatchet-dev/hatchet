import json

from dotenv import load_dotenv

from hatchet_sdk import Context, CreateWorkflowVersionOpts, Hatchet

load_dotenv()

hatchet = Hatchet(debug=True)


@hatchet.workflow(on_events=["user:create"])
class MyWorkflow:
    def __init__(self):
        self.my_value = "test"

    @hatchet.step()
    def step1(self, context: Context):
        test = context.playground("test", "test")
        test2 = context.playground("test2", 100)
        test3 = context.playground("test3", None)

        print(test)
        print(test2)

        print("executed step1")
        pass

    @hatchet.step(parents=["step1"], timeout="4s")
    def step2(self, context):
        print("started step2")
        context.sleep(1)
        print("finished step2")


workflow = MyWorkflow()
worker = hatchet.worker("test-worker", max_runs=4)
worker.register_workflow(workflow)

# workflow1 = hatchet.client.admin.put_workflow(
#     "workflow-copy-2",
#     MyWorkflow(),
#     overrides=CreateWorkflowVersionOpts(
#         cron_triggers=["* * * * *"],
#         cron_input=json.dumps({"test": "test"}),
#     ),
# )

# print(workflow1)

worker.start()
