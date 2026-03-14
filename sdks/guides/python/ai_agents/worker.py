from hatchet_sdk import (
    ConcurrencyExpression,
    ConcurrencyLimitStrategy,
    DurableContext,
    Hatchet,
)
from pydantic import BaseModel

from .llm_service import ChatMessage, get_llm_service
from .tool_service import get_tool_service

hatchet = Hatchet(debug=True)


class AgentInput(BaseModel):
    query: str = "Hello"
    session_id: str | None = None


# > Step 01 Define Agent Task
@hatchet.durable_task(
    name="ReasoningLoopAgent",
    input_validator=AgentInput,
    concurrency=ConcurrencyExpression(
        expression="input.session_id != null ? string(input.session_id) : 'constant'",
        max_runs=1,
        limit_strategy=ConcurrencyLimitStrategy.CANCEL_IN_PROGRESS,
    ),
)
async def agent_task(input: AgentInput, ctx: DurableContext) -> dict:
    """Agent loop: reason, act, observe. Streams output, survives restarts."""
    return await agent_reasoning_loop(input.query)
# !!


# > Step 02 Reasoning Loop
async def agent_reasoning_loop(query: str) -> dict:
    llm = get_llm_service()
    tools = get_tool_service()
    messages = [ChatMessage(role="user", content=query)]
    for _ in range(10):
        resp = llm.complete(messages)
        if resp.done:
            return {"response": resp.content}
        for tc in resp.tool_calls:
            result = tools.run(tc.name, tc.args)
            messages.append(ChatMessage(role="tool", content=result))
    return {"response": "Max iterations reached"}
# !!


# > Step 03 Stream Response
@hatchet.durable_task(
    name="StreamingAgentTask",
    input_validator=AgentInput,
    concurrency=ConcurrencyExpression(
        expression="input.session_id != null ? string(input.session_id) : 'constant'",
        max_runs=1,
        limit_strategy=ConcurrencyLimitStrategy.CANCEL_IN_PROGRESS,
    ),
)
async def streaming_agent(input: AgentInput, ctx: DurableContext) -> dict:
    """Stream tokens to the client as they're generated."""
    tokens = ["Hello", " ", "world", "!"]
    for t in tokens:
        await ctx.aio_put_stream(t)
    return {"done": True}


# !!


def main() -> None:
    # > Step 04 Run Worker
    worker = hatchet.worker(
        "agent-worker",
        workflows=[agent_task, streaming_agent],
        slots=5,
    )
    worker.start()
    # !!


if __name__ == "__main__":
    main()
