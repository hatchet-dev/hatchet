from hatchet_sdk import Context, DurableContext, EmptyModel, Hatchet

try:
    from .mock_llm import mock_orchestrator_llm, mock_specialist_llm
except ImportError:
    from mock_llm import mock_orchestrator_llm, mock_specialist_llm

hatchet = Hatchet(debug=True)

research_wf = hatchet.workflow(name="ResearchSpecialist")
writing_wf = hatchet.workflow(name="WritingSpecialist")
code_wf = hatchet.workflow(name="CodeSpecialist")


# > Step 01 Specialist Agents
@research_wf.task(execution_timeout="3m")
async def research(input: dict, ctx: Context) -> dict:
    return {"result": mock_specialist_llm(input["task"], "research")}


@writing_wf.task(execution_timeout="2m")
async def write(input: dict, ctx: Context) -> dict:
    return {"result": mock_specialist_llm(input["task"], "writing")}


@code_wf.task(execution_timeout="2m")
async def code(input: dict, ctx: Context) -> dict:
    return {"result": mock_specialist_llm(input["task"], "code")}


specialists = {
    "research": research_wf,
    "writing": writing_wf,
    "code": code_wf,
}


# > Step 02 Orchestrator Loop
@hatchet.durable_task(name="MultiAgentOrchestrator", execution_timeout="15m")
async def orchestrator(input: EmptyModel, ctx: DurableContext) -> dict:
    messages = [{"role": "user", "content": input["goal"]}]

    for _ in range(10):
        response = mock_orchestrator_llm(messages)

        if response["done"]:
            return {"result": response["content"]}

        specialist_wf = specialists.get(response["tool_call"]["name"])
        if not specialist_wf:
            raise ValueError(f"Unknown specialist: {response['tool_call']['name']}")

        result = await specialist_wf.aio_run(input={
            "task": response["tool_call"]["args"]["task"],
            "context": "\n".join(m["content"] for m in messages),
        })

        messages.append({"role": "assistant", "content": f"Called {response['tool_call']['name']}"})
        messages.append({"role": "tool", "content": result["result"]})

    return {"result": "Max iterations reached"}


def main() -> None:
    # > Step 03 Run Worker
    worker = hatchet.worker(
        "multi-agent-worker",
        workflows=[research_wf, writing_wf, code_wf, orchestrator],
        slots=10,
    )
    worker.start()


if __name__ == "__main__":
    main()
