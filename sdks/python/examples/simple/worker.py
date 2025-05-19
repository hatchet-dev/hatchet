from hatchet_sdk import Context, EmptyModel, Hatchet
import time
import asyncio

hatchet = Hatchet(debug=True)


@hatchet.task()
async def simple(input: EmptyModel, ctx: Context) -> dict[str, str]:
    for i in range(60):
        print(f"blocking task {i}")
        time.sleep(1)

@hatchet.task()
async def non_blocking(input: EmptyModel, ctx: Context) -> dict[str, str]:
    for i in range(60):
        print(f"non-blocking task {i}")
        await asyncio.sleep(1)


def main() -> None:
    worker = hatchet.worker("test-worker", workflows=[simple, non_blocking])
    worker.start()


if __name__ == "__main__":
    main()
