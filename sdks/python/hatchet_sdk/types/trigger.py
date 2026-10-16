from pydantic import BaseModel, Field

from hatchet_sdk.types.labels import DesiredWorkerLabel
from hatchet_sdk.types.priority import Priority
from hatchet_sdk.utils.typing import JSONSerializableMapping


class ScheduleTriggerWorkflowOptions(BaseModel):
    child_key: str | None = None
    additional_metadata: JSONSerializableMapping = Field(default_factory=dict)
    priority: Priority | None = None


class TriggerWorkflowOptions(ScheduleTriggerWorkflowOptions):
    desired_worker_id: str | None = None
    sticky: bool = False
    desired_worker_labels: list[DesiredWorkerLabel] | None = None


class WorkflowRunTriggerConfig(BaseModel):
    workflow_name: str
    input: str | None
    options: TriggerWorkflowOptions


class BulkPushEventWithMetadata(BaseModel):
    key: str
    payload: JSONSerializableMapping = Field(default_factory=dict)
    additional_metadata: JSONSerializableMapping = Field(default_factory=dict)
    priority: Priority | None = None
    scope: str | None = None
