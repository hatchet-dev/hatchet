from pydantic import BaseModel, ConfigDict


class DesiredWorkerLabel(BaseModel):
    model_config = ConfigDict(arbitrary_types_allowed=True)

    value: str | int
    required: bool = False
    weight: int | None = None
    comparator: int | None = None
