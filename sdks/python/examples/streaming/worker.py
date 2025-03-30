import asyncio

from hatchet_sdk import Context, EmptyModel, Hatchet

hatchet = Hatchet(debug=True)

# ❓ Streaming

streaming_workflow = hatchet.workflow(name="StreamingWorkflow")


@streaming_workflow.task()
async def step1(input: EmptyModel, ctx: Context) -> None:
    for i in range(10):
        ctx.put_stream(f"Processing {i}")
        await asyncio.sleep(1)


def main() -> None:
    worker = hatchet.worker("test-worker", workflows=[streaming_workflow])
    worker.start()


# ‼️

if __name__ == "__main__":
    main()
