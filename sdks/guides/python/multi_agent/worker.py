from hatchet_sdk import DurableContext, Hatchet
from pydantic import BaseModel

from .mock_llm import mock_orchestrator_llm, mock_specialist_llm

hatchet = Hatchet(debug=True)


class TaskInput(BaseModel):
    task: str
    context: str = ""


class GoalInput(BaseModel):
    goal: str


# > Step 01 Specialist Agents
@hatchet.durable_task(name="ResearchSpecialist", execution_timeout="3m", input_validator=TaskInput)
async def research(input: TaskInput, ctx: DurableContext) -> dict:
    return {"result": mock_specialist_llm(input.task, "research")}


@hatchet.durable_task(name="WritingSpecialist", execution_timeout="2m", input_validator=TaskInput)
async def write(input: TaskInput, ctx: DurableContext) -> dict:
    return {"result": mock_specialist_llm(input.task, "writing")}


@hatchet.durable_task(name="CodeSpecialist", execution_timeout="2m", input_validator=TaskInput)
async def code(input: TaskInput, ctx: DurableContext) -> dict:
    return {"result": mock_specialist_llm(input.task, "code")}
# !!


specialists = {
    "research": research,
    "writing": write,
    "code": code,
}


# > Step 02 Orchestrator Loop
@hatchet.durable_task(name="MultiAgentOrchestrator", execution_timeout="15m", input_validator=GoalInput)
async def orchestrator(input: GoalInput, ctx: DurableContext) -> dict:
    messages: list[dict[str, str]] = [{"role": "user", "content": input.goal}]

    for _ in range(10):
        response = mock_orchestrator_llm(messages)

        if response["done"]:
            return {"result": response["content"]}

        specialist = specialists.get(response["tool_call"]["name"])
        if not specialist:
            raise ValueError(f"Unknown specialist: {response['tool_call']['name']}")

        result = await specialist.aio_run(input=TaskInput(
            task=response["tool_call"]["args"]["task"],
            context="\n".join(m["content"] for m in messages),
        ))

        messages.append({"role": "assistant", "content": f"Called {response['tool_call']['name']}"})
        messages.append({"role": "tool", "content": result["result"]})

    return {"result": "Max iterations reached"}
# !!


def main() -> None:
    # > Step 03 Run Worker
    worker = hatchet.worker(
        "multi-agent-worker",
        workflows=[research, write, code, orchestrator],
        slots=10,
    )
    worker.start()
    # !!


if __name__ == "__main__":
    main()
