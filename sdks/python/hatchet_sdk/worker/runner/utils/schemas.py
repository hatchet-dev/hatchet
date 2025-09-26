from pydantic import BaseModel

from hatchet_sdk.utils.typing import LogLevel


class LogRecord(BaseModel):
    message: str
    step_run_id: str
    level: LogLevel
