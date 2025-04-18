from typing import ParamSpec, TypeVar, cast

from openai.types.chat import (
    ChatCompletionMessageParam,
    ChatCompletionSystemMessageParam,
    ChatCompletionUserMessageParam,
)

T = TypeVar("T")
P = ParamSpec("P")
R = TypeVar("R")


SYSTEM_PROMPT = """
You're an SDK documentation expert working on improving the readability of Hatchet's Python SDK documentation. You will be given
a markdown file, and your task is to fix any broken MDX so it can be used as a page on our Nextra documentation site.

In your work, follow these instructions:

1. Strip any unnecessary paragraph characters, but do not change any actual code, sentences, or content. You should keep the documentation as close to the original as possible, meaning that you should not generate new content, you should not consolidate existing content, you should not rearrange content, and so on.
2. Return only the content. You should not enclode the markdown in backticks or any other formatting.
3. You must ensure that MDX will render any tables correctly. One thing in particular to be on the lookout for is the use of the pipe `|` in type hints in the tables. For example, `int | None` is the Python type `Optional[int]` and should render in a single column with an escaped pipe character.
4. All code blocks should be formatted as `python`.
"""


def create_prompt_messages(
    user_prompt_content: str,
) -> list[ChatCompletionMessageParam]:
    return cast(
        list[ChatCompletionMessageParam],
        [
            ChatCompletionSystemMessageParam(content=SYSTEM_PROMPT, role="system"),
            ChatCompletionUserMessageParam(content=user_prompt_content, role="user"),
        ],
    )
