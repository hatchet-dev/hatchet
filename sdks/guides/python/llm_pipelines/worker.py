from hatchet_sdk import Context, Hatchet
from pydantic import BaseModel

try:
    from .llm_service import get_llm_service
except ImportError:
    from llm_service import get_llm_service

hatchet = Hatchet(debug=True)


# > Step 01 Define Pipeline
class PipelineInput(BaseModel):
    prompt: str


llm_wf = hatchet.workflow(name="LLMPipeline", input_validator=PipelineInput)


@llm_wf.task()
async def prompt_task(input: PipelineInput, ctx: Context) -> dict:
    return {"prompt": input.prompt}


# !!


# > Step 02 Prompt Task
def _build_prompt(user_input: str, context: str = "") -> str:
    return f"Process the following: {user_input}" + (f"\nContext: {context}" if context else "")
# !!


# > Step 03 Validate Task
@llm_wf.task(parents=[prompt_task])
async def generate_task(input: PipelineInput, ctx: Context) -> dict:
    prev = ctx.task_output(prompt_task)
    output = get_llm_service().generate(prev["prompt"])
    if not output.get("valid"):
        raise ValueError("Validation failed")
    return output


# !!


def main() -> None:
    # > Step 04 Run Worker
    worker = hatchet.worker(
        "llm-pipeline-worker",
        workflows=[llm_wf],
    )
    worker.start()
    # !!


if __name__ == "__main__":
    main()
