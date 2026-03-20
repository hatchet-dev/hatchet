from hatchet_sdk import Context, DurableContext, EmptyModel, Hatchet

try:
    from .mock_llm import mock_evaluate, mock_generate
except ImportError:
    from mock_llm import mock_evaluate, mock_generate

hatchet = Hatchet()

generator_wf = hatchet.workflow(name="GenerateDraft")
evaluator_wf = hatchet.workflow(name="EvaluateDraft")


# > Step 01 Define Tasks
@generator_wf.task()
async def generate_draft(input: dict, ctx: Context) -> dict:
    prompt = (
        f"Improve this draft.\n\nDraft: {input['previous_draft']}\nFeedback: {input['feedback']}"
        if input.get("feedback")
        else f'Write a social media post about "{input["topic"]}" for {input["audience"]}. Under 100 words.'
    )
    return {"draft": mock_generate(prompt)}


@evaluator_wf.task()
async def evaluate_draft(input: dict, ctx: Context) -> dict:
    return mock_evaluate(input["draft"])


# !!


# > Step 02 Optimization Loop
@hatchet.durable_task(name="EvaluatorOptimizer", execution_timeout="5m")
async def evaluator_optimizer(input: EmptyModel, ctx: DurableContext) -> dict:
    max_iterations = 3
    threshold = 0.8
    draft = ""
    feedback = ""

    for i in range(max_iterations):
        generated = await generator_wf.aio_run(
            input={
                "topic": input["topic"],
                "audience": input["audience"],
                "previous_draft": draft or None,
                "feedback": feedback or None,
            }
        )
        draft = generated["draft"]

        evaluation = await evaluator_wf.aio_run(
            input={"draft": draft, "topic": input["topic"], "audience": input["audience"]}
        )

        if evaluation["score"] >= threshold:
            return {"draft": draft, "iterations": i + 1, "score": evaluation["score"]}
        feedback = evaluation["feedback"]

    return {"draft": draft, "iterations": max_iterations, "score": -1}


# !!


def main() -> None:
    # > Step 03 Run Worker
    worker = hatchet.worker(
        "evaluator-optimizer-worker",
        workflows=[generator_wf, evaluator_wf, evaluator_optimizer],
        slots=5,
    )
    worker.start()
    # !!


if __name__ == "__main__":
    main()
