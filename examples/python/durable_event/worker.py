from hatchet_sdk import DurableContext, EmptyModel, Hatchet, UserEventCondition

hatchet = Hatchet(debug=True)

EVENT_KEY = "user:update"


# ❓ Durable Event
@hatchet.durable_task(name="DurableEventTask")
async def durable_event_task(input: EmptyModel, ctx: DurableContext) -> None:
    res = await ctx.aio_wait_for(
        "event",
        UserEventCondition(event_key="user:update"),
    )

    print("got event", res)


# !!


@hatchet.durable_task(name="DurableEventWithFilterTask")
async def durable_event_task_with_filter(
    input: EmptyModel, ctx: DurableContext
) -> None:
    # ❓ Durable Event With Filter
    res = await ctx.aio_wait_for(
        "event",
        UserEventCondition(
            event_key="user:update", expression="input.user_id == '1234'"
        ),
    )
    # !!

    print("got event", res)


def main() -> None:
    worker = hatchet.worker(
        "durable-event-worker",
        workflows=[durable_event_task, durable_event_task_with_filter],
    )
    worker.start()


if __name__ == "__main__":
    main()
