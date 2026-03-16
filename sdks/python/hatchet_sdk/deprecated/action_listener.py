"""Legacy GetActionListenerRequest using slots: int (pre-slot-config engines)."""

from pydantic import BaseModel, ConfigDict, Field, model_validator

from hatchet_sdk.contracts.dispatcher_pb2 import WorkerLabels
from hatchet_sdk.types.labels import WorkerLabel


class LegacyGetActionListenerRequest(BaseModel):
    model_config = ConfigDict(arbitrary_types_allowed=True)

    worker_name: str
    services: list[str]
    actions: list[str]
    slots: int
    raw_labels: list[WorkerLabel] = Field(default_factory=list)

    labels: dict[str, WorkerLabels] = Field(default_factory=dict)

    @model_validator(mode="after")
    def validate_labels(self) -> "LegacyGetActionListenerRequest":
        self.labels = {}

        for label in self.raw_labels:
            if label.key is not None:
                self.labels[label.key] = label.to_proto()

        return self
