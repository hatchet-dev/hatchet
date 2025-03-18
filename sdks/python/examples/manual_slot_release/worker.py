from hatchet_sdk import Hatchet, Context, EmptyModel
import time

hatchet = Hatchet()

# ❓ SlotRelease

slot_release_workflow = hatchet.workflow(name="SlotReleaseWorkflow")

@slot_release_workflow.task()
def step1(input: EmptyModel, ctx: Context):
    print('RESOURCE INTENSIVE PROCESS')
    time.sleep(10)

    # 👀 Release the slot after the resource-intensive process, so that other steps can run
    ctx.release_slot()

    print("NON RESOURCE INTENSIVE PROCESS")
    return {"status": "success"}

# ‼️