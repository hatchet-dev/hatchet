from hatchet_sdk import Hatchet
from dotenv import load_dotenv

load_dotenv()

hatchet = Hatchet(debug=True)


@hatchet.workflow(on_events=["man:create"])
class ManualTriggerWorkflow:
    @hatchet.step()
    def step1(self, context):
        res = context.overrides('res', "HELLO")

        context.sleep(3)
        print("executed step1")
        return {"step1": "data1 "+res}

    @hatchet.step(parents=["step1"], timeout='4s')
    def step2(self, context):
        print("started step2")
        context.sleep(1)
        print("finished step2")
        return {"step2": "data2"}

    # @hatchet.step()
    # def stepb(self, context):
    #     res = context.overrides('res', "HELLO")

    #     context.sleep(3)
    #     print("executed step1")
    #     return {"step1": "data1 "+res}

    # @hatchet.step(parents=["stepb"], timeout='4s')
    # def stepc(self, context):
    #     print("started step2")
    #     context.sleep(1)
    #     print("finished step2")
    #     return {"step2": "data2"}


workflow = ManualTriggerWorkflow()
worker = hatchet.worker('manual-worker', max_threads=4)
worker.register_workflow(workflow)

worker.start()
