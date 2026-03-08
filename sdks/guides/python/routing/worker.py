from hatchet_sdk import DurableContext, Hatchet
from pydantic import BaseModel

from .mock_classifier import mock_classify, mock_reply

hatchet = Hatchet(debug=True)


class MessageInput(BaseModel):
    message: str


# > Step 01 Classify Task
@hatchet.durable_task(name="ClassifyMessage", input_validator=MessageInput)
async def classify_message(input: MessageInput, ctx: DurableContext) -> dict:
    return {"category": mock_classify(input.message)}
# !!


# > Step 02 Specialist Tasks
@hatchet.durable_task(name="HandleSupport", input_validator=MessageInput)
async def handle_support(input: MessageInput, ctx: DurableContext) -> dict:
    return {"response": mock_reply(input.message, "support"), "category": "support"}


@hatchet.durable_task(name="HandleSales", input_validator=MessageInput)
async def handle_sales(input: MessageInput, ctx: DurableContext) -> dict:
    return {"response": mock_reply(input.message, "sales"), "category": "sales"}


@hatchet.durable_task(name="HandleDefault", input_validator=MessageInput)
async def handle_default(input: MessageInput, ctx: DurableContext) -> dict:
    return {"response": mock_reply(input.message, "other"), "category": "other"}
# !!


# > Step 03 Router Task
@hatchet.durable_task(name="MessageRouter", execution_timeout="2m", input_validator=MessageInput)
async def message_router(input: MessageInput, ctx: DurableContext) -> dict:
    classification = await classify_message.aio_run(MessageInput(message=input.message))

    if classification["category"] == "support":
        return await handle_support.aio_run(MessageInput(message=input.message))
    if classification["category"] == "sales":
        return await handle_sales.aio_run(MessageInput(message=input.message))
    return await handle_default.aio_run(MessageInput(message=input.message))
# !!


def main() -> None:
    # > Step 04 Run Worker
    worker = hatchet.worker(
        "routing-worker",
        workflows=[classify_message, handle_support, handle_sales, handle_default, message_router],
        slots=5,
    )
    worker.start()
    # !!


if __name__ == "__main__":
    main()
