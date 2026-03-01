from hatchet_sdk import Context, DurableContext, EmptyModel, Hatchet

try:
    from .mock_classifier import mock_classify, mock_reply
except ImportError:
    from mock_classifier import mock_classify, mock_reply

hatchet = Hatchet(debug=True)

classify_wf = hatchet.workflow(name="ClassifyMessage")
support_wf = hatchet.workflow(name="HandleSupport")
sales_wf = hatchet.workflow(name="HandleSales")
default_wf = hatchet.workflow(name="HandleDefault")


# > Step 01 Classify Task
@classify_wf.task()
async def classify_message(input: dict, ctx: Context) -> dict:
    return {"category": mock_classify(input["message"])}
# !!


# > Step 02 Specialist Tasks
@support_wf.task()
async def handle_support(input: dict, ctx: Context) -> dict:
    return {"response": mock_reply(input["message"], "support"), "category": "support"}


@sales_wf.task()
async def handle_sales(input: dict, ctx: Context) -> dict:
    return {"response": mock_reply(input["message"], "sales"), "category": "sales"}


@default_wf.task()
async def handle_default(input: dict, ctx: Context) -> dict:
    return {"response": mock_reply(input["message"], "other"), "category": "other"}
# !!


# > Step 03 Router Task
@hatchet.durable_task(name="MessageRouter", execution_timeout="2m")
async def message_router(input: EmptyModel, ctx: DurableContext) -> dict:
    classification = await classify_wf.aio_run(input={"message": input["message"]})

    if classification["category"] == "support":
        return await support_wf.aio_run(input={"message": input["message"]})
    if classification["category"] == "sales":
        return await sales_wf.aio_run(input={"message": input["message"]})
    return await default_wf.aio_run(input={"message": input["message"]})
# !!


def main() -> None:
    # > Step 04 Run Worker
    worker = hatchet.worker(
        "routing-worker",
        workflows=[classify_wf, support_wf, sales_wf, default_wf, message_router],
        slots=5,
    )
    worker.start()
    # !!


if __name__ == "__main__":
    main()
