from openai import AsyncOpenAI
from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    openai_api_key: str = "fake-key"


settings = Settings()
client = AsyncOpenAI(api_key=settings.openai_api_key)
