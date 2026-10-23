from datetime import timedelta

from hatchet_sdk import (
    DurableContext,
    Hatchet,
    SleepCondition,
    OrGroup,
    UserEventCondition,
)

hatchet = Hatchet()


# > Durable Sleep
@hatchet.durable_task(name="DurableSleepTask")
async def durable_sleep_task(input: None, ctx: DurableContext) -> None:
    res = await ctx.aio_wait_for(
        "foo",
        SleepCondition(duration=timedelta(seconds=2)),
        UserEventCondition(event_key="my-event"),
        OrGroup(
            [
                SleepCondition(duration=timedelta(seconds=5)),
                UserEventCondition(event_key="my-event-2"),
            ]
        ),
    )

    print("got result", res)


# !!


def main() -> None:
    worker = hatchet.worker("durable-sleep-worker", workflows=[durable_sleep_task])
    worker.start()


if __name__ == "__main__":
    main()
