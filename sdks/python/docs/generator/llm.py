from openai import AsyncOpenAI
from pydantic_settings import BaseSettings

from docs.generator.prompts import create_prompt_messages


class Settings(BaseSettings):
    openai_api_key: str = "fake-key"


settings = Settings()
client = AsyncOpenAI(api_key=settings.openai_api_key)


async def parse_markdown(original_markdown: str) -> str | None:
    response = await client.chat.completions.create(
        model="gpt-4o", messages=create_prompt_messages(original_markdown)
    )

    return response.choices[0].message.content
