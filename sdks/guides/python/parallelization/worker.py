import asyncio

from hatchet_sdk import Context, DurableContext, EmptyModel, Hatchet

try:
    from .mock_llm import mock_evaluate, mock_generate_content, mock_safety_check
except ImportError:
    from mock_llm import mock_evaluate, mock_generate_content, mock_safety_check

hatchet = Hatchet(debug=True)

content_wf = hatchet.workflow(name="GenerateContent")
safety_wf = hatchet.workflow(name="SafetyCheck")
evaluator_wf = hatchet.workflow(name="EvaluateContent")


# > Step 01 Parallel Tasks
@content_wf.task()
async def generate_content(input: dict, ctx: Context) -> dict:
    return {"content": mock_generate_content(input["message"])}


@safety_wf.task()
async def safety_check(input: dict, ctx: Context) -> dict:
    return mock_safety_check(input["message"])


@evaluator_wf.task()
async def evaluate_content(input: dict, ctx: Context) -> dict:
    return mock_evaluate(input["content"])
# !!


# > Step 02 Sectioning
@hatchet.durable_task(name="ParallelSectioning", execution_timeout="2m")
async def sectioning_task(input: EmptyModel, ctx: DurableContext) -> dict:
    content_result, safety_result = await asyncio.gather(
        content_wf.aio_run(input={"message": input["message"]}),
        safety_wf.aio_run(input={"message": input["message"]}),
    )

    if not safety_result["safe"]:
        return {"blocked": True, "reason": safety_result["reason"]}
    return {"blocked": False, "content": content_result["content"]}
# !!


# > Step 03 Voting
@hatchet.durable_task(name="ParallelVoting", execution_timeout="3m")
async def voting_task(input: EmptyModel, ctx: DurableContext) -> dict:
    votes = await asyncio.gather(
        evaluator_wf.aio_run(input={"content": input["content"]}),
        evaluator_wf.aio_run(input={"content": input["content"]}),
        evaluator_wf.aio_run(input={"content": input["content"]}),
    )

    approvals = sum(1 for v in votes if v["approved"])
    avg_score = sum(v["score"] for v in votes) / len(votes)

    return {"approved": approvals >= 2, "average_score": avg_score, "votes": len(votes)}
# !!


def main() -> None:
    # > Step 04 Run Worker
    worker = hatchet.worker(
        "parallelization-worker",
        workflows=[content_wf, safety_wf, evaluator_wf, sectioning_task, voting_task],
        slots=10,
    )
    worker.start()
    # !!


if __name__ == "__main__":
    main()
