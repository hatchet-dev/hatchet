from hatchet_sdk import Hatchet

hatchet = Hatchet(debug=True)


# > Step 03 Subscribe Client
# Client triggers the task and subscribes to the stream.
async def run_and_subscribe() -> None:
    run = await hatchet.runs.aio_create(workflow_name="stream_task", input={})
    async for chunk in hatchet.runs.subscribe_to_stream(run.run.metadata.id):
        print(chunk)


