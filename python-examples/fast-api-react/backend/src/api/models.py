from pydantic import BaseModel
from typing import List


class Message(BaseModel):
    role: str
    content: str


class MessageRequest(BaseModel):
    messages: List[Message]
    url: str
