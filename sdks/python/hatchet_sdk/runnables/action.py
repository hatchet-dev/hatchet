import json
from dataclasses import field
from datetime import datetime
from enum import StrEnum
from typing import TYPE_CHECKING, Any

from pydantic import (
    AliasChoices,
    BaseModel,
    ConfigDict,
    Field,
    field_validator,
    model_validator,
)

from hatchet_sdk.runnables.types import BatchMemberId
from hatchet_sdk.utils.typing import JSONSerializableMapping

if TYPE_CHECKING:
    from hatchet_sdk.config import ClientConfig
    from hatchet_sdk.utils.opentelemetry import OTelAttribute

ActionKey = str


class ActionPayload(BaseModel):
    model_config = ConfigDict(extra="allow")

    input: JSONSerializableMapping = Field(default_factory=dict)
    parents: dict[str, JSONSerializableMapping] = Field(default_factory=dict)
    overrides: JSONSerializableMapping = Field(default_factory=dict)
    user_data: JSONSerializableMapping = Field(default_factory=dict)
    task_run_errors: dict[str, str] = Field(
        default_factory=dict,
        validation_alias=AliasChoices("task_run_errors", "step_run_errors"),
    )
    triggered_by: str | None = None
    triggers: JSONSerializableMapping = Field(default_factory=dict)
    filter_payload: JSONSerializableMapping = Field(default_factory=dict)

    @field_validator(
        "input",
        "parents",
        "overrides",
        "user_data",
        "task_run_errors",
        "filter_payload",
        mode="before",
    )
    @classmethod
    def validate_fields(cls, v: Any) -> Any:  # noqa: ANN401
        return v or {}

    @model_validator(mode="after")
    def validate_filter_payload(self) -> "ActionPayload":
        self.filter_payload = self.triggers.get("filter_payload", {}) or {}

        return self


class ActionType(StrEnum):
    START_STEP_RUN = "START_STEP_RUN"
    CANCEL_STEP_RUN = "CANCEL_STEP_RUN"
    START_BATCH = "START_BATCH"


class BatchStartPayload(BaseModel):
    expected_size: int
    trigger_reason: str
    trigger_time: datetime


class BatchItemData(BaseModel):
    payload: ActionPayload
    workflow_run_id: str


class BatchEventItem(BaseModel):
    """A single task's contribution to a batched STARTED/FAILED/CANCELLED action event."""

    task_run_external_id: BatchMemberId
    payload: str | None = None
    should_not_retry: bool = False
    retry_after_ms: int | None = None


class Action(BaseModel):
    model_config = ConfigDict(extra="ignore")

    worker_id: str
    tenant_id: str
    workflow_run_id: str
    workflow_id: str | None = None
    workflow_version_id: str | None = None
    job_id: str
    job_name: str
    job_run_id: str
    task_id: str
    task_run_external_id: str
    action_id: str
    task_name: str = ""
    action_type: ActionType
    retry_count: int
    action_payload: ActionPayload
    additional_metadata: JSONSerializableMapping = field(default_factory=dict)

    # Batch metadata
    batch_id: str | None = None
    batch_key: str | None = None
    batch_start: BatchStartPayload | None = None
    # Populated for START_BATCH: maps task_run_external_id -> item data (payload + workflow_run_id)
    batch_items: dict[BatchMemberId, BatchItemData] | None = None

    child_workflow_index: int | None = None
    child_workflow_key: str | None = None
    parent_workflow_run_id: str | None = None

    priority: int | None = None
    durable_task_invocation_count: int | None = None

    triggering_event_external_id: str | None = None
    triggering_event_key: str | None = None

    def get_otel_attributes(self, config: "ClientConfig") -> dict[str, str | int]:
        try:
            payload_str = json.dumps(self.action_payload.model_dump(), default=str)
        except Exception:
            payload_str = str(self.action_payload)

        attrs: dict[OTelAttribute, str | int | None] = {
            "tenant_id": self.tenant_id,
            "worker_id": self.worker_id,
            "workflow_run_id": self.workflow_run_id,
            "step_id": self.task_id,
            "step_run_id": self.task_run_external_id,
            "retry_count": self.retry_count,
            "parent_workflow_run_id": self.parent_workflow_run_id,
            "child_workflow_index": self.child_workflow_index,
            "child_workflow_key": self.child_workflow_key,
            "payload": payload_str,
            "workflow_name": self.job_name,
            "action_name": self.action_id,
            "step_name": self.task_name,
            "workflow_id": self.workflow_id,
            "workflow_version_id": self.workflow_version_id,
            "triggering_event_external_id": self.triggering_event_external_id,
            "triggering_event_key": self.triggering_event_key,
        }

        result = {
            f"hatchet.{k}": v
            for k, v in attrs.items()
            if v and k not in config.otel.excluded_attributes
        }
        result["instrumentor"] = "hatchet"
        return result

    @property
    def key(self) -> ActionKey:
        """
        This key is used to uniquely identify a single task run by its id + retry count.
        It's used when storing references to a task, a context, etc. in a dictionary so that
        we can look up those items in the dictionary by a unique key.
        """
        if self.durable_task_invocation_count is not None:
            return f"{self.task_run_external_id}/{self.retry_count}/{self.durable_task_invocation_count}"

        if self.action_type == ActionType.START_BATCH:
            # START_BATCH does not correspond to a unique task_run_external_id; key it by batch id.
            batch_key = self.batch_id or self.action_id or "unknown"
            return f"{batch_key}/{self.retry_count or 0}"
        return f"{self.task_run_external_id}/{self.retry_count}"
