from pydantic import BaseModel


class DesiredWorkerLabel(BaseModel):
    value: str | int
    required: bool = False
    weight: int | None = None
    comparator: int | None = None
