from datetime import timedelta

from hatchet_sdk import DurableContext, EmptyModel, Hatchet

hatchet = Hatchet(debug=True)


# > Durable Sleep
@hatchet.durable_task(name="DurableSleepTask")
async def durable_sleep_task(input: EmptyModel, ctx: DurableContext) -> None:
    res = await ctx.aio_sleep_for(timedelta(seconds=5))

    print("got result", res)


# !!


def main() -> None:
    worker = hatchet.worker("durable-sleep-worker", workflows=[durable_sleep_task])
    worker.start()


if __name__ == "__main__":
    main()
