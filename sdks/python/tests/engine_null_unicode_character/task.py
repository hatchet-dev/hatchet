from pydantic import BaseModel

from hatchet_sdk import Context, EmptyModel, Hatchet

hatchet = Hatchet(debug=True)


class Message(BaseModel):
    content: str


@hatchet.task(input_validator=Message)
async def engine_null_unicode_rejection(input: Message, ctx: Context) -> dict[str, str]:
    return {"result": "Hello\x00World", "message": input.content}
