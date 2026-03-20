from hatchet_sdk import Hatchet

hatchet = Hatchet()


# > Step 03 Subscribe Client
# Client triggers the task and subscribes to the stream.
async def run_and_subscribe():
    run = await hatchet.runs.create(workflow_name="stream_task", input={})
    async for chunk in hatchet.runs.subscribe_to_stream(run.run_id):
        print(chunk)


