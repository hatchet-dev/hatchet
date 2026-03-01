from hatchet_sdk import (
    ConcurrencyExpression,
    ConcurrencyLimitStrategy,
    DurableContext,
    EmptyModel,
    Hatchet,
)

try:
    from .llm_service import get_llm_service
    from .tool_service import get_tool_service
except ImportError:
    from llm_service import get_llm_service
    from tool_service import get_tool_service

hatchet = Hatchet(debug=True)


# > Step 01 Define Agent Task
@hatchet.durable_task(
    name="AgentTask",
    concurrency=ConcurrencyExpression(
        expression="input.session_id != null ? string(input.session_id) : 'constant'",
        max_runs=1,
        limit_strategy=ConcurrencyLimitStrategy.CANCEL_IN_PROGRESS,
    ),
)
async def agent_task(input: EmptyModel, ctx: DurableContext) -> dict:
    """Agent loop: reason, act, observe. Streams output, survives restarts."""
    query = "Hello"
    if isinstance(input, dict) and input.get("query"):
        query = str(input["query"])
    elif hasattr(input, "query") and input.query:
        query = str(input.query)
    return await agent_reasoning_loop(query)
# !!


# > Step 02 Reasoning Loop
async def agent_reasoning_loop(query: str) -> dict:
    llm = get_llm_service()
    tools = get_tool_service()
    messages = [{"role": "user", "content": query}]
    for _ in range(10):
        resp = llm.complete(messages)
        if resp.get("done"):
            return {"response": resp["content"]}
        for tc in resp.get("tool_calls", []):
            result = tools.run(tc["name"], tc.get("args", {}))
            messages.append({"role": "tool", "content": result})
    return {"response": "Max iterations reached"}
# !!


# > Step 03 Stream Response
@hatchet.durable_task(
    name="StreamingAgentTask",
    concurrency=ConcurrencyExpression(
        expression="input.session_id != null ? string(input.session_id) : 'constant'",
        max_runs=1,
        limit_strategy=ConcurrencyLimitStrategy.CANCEL_IN_PROGRESS,
    ),
)
async def streaming_agent(input: EmptyModel, ctx: DurableContext) -> dict:
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
