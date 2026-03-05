"""Mock LLM for multi-agent orchestration - no external API dependencies."""

_orchestrator_call_count = 0


def mock_orchestrator_llm(messages: list[dict]) -> dict:
    global _orchestrator_call_count
    _orchestrator_call_count += 1
    if _orchestrator_call_count == 1:
        return {"done": False, "content": "", "tool_call": {"name": "research", "args": {"task": "Find key facts about the topic"}}}
    if _orchestrator_call_count == 2:
        return {"done": False, "content": "", "tool_call": {"name": "writing", "args": {"task": "Write a summary from the research"}}}
    return {"done": True, "content": "Here is the final report combining research and writing."}


def mock_specialist_llm(task: str, role: str) -> str:
    return f"[{role}] Completed: {task}"
