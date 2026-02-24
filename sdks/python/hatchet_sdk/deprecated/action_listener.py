"""Legacy GetActionListenerRequest using slots: int (pre-slot-config engines)."""

from pydantic import BaseModel, ConfigDict, Field, model_validator

from hatchet_sdk.contracts.dispatcher_pb2 import WorkerLabels


class LegacyGetActionListenerRequest(BaseModel):
    model_config = ConfigDict(arbitrary_types_allowed=True)

    worker_name: str
    services: list[str]
    actions: list[str]
    slots: int
    raw_labels: dict[str, str | int] = Field(default_factory=dict)

    labels: dict[str, WorkerLabels] = Field(default_factory=dict)

    @model_validator(mode="after")
    def validate_labels(self) -> "LegacyGetActionListenerRequest":
        self.labels = {}

        for key, value in self.raw_labels.items():
            if isinstance(value, int):
                self.labels[key] = WorkerLabels(int_value=value)
            else:
                self.labels[key] = WorkerLabels(str_value=str(value))

        return self
