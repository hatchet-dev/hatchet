import json
from dataclasses import field
from enum import Enum
from typing import TYPE_CHECKING, Any

from pydantic import BaseModel, ConfigDict, Field, field_validator, model_validator

from hatchet_sdk.utils.opentelemetry import OTelAttribute
from hatchet_sdk.utils.typing import JSONSerializableMapping

if TYPE_CHECKING:
    from hatchet_sdk.config import ClientConfig

ActionKey = str


class ActionPayload(BaseModel):
    model_config = ConfigDict(extra="allow")

    input: JSONSerializableMapping = Field(default_factory=dict)
    parents: dict[str, JSONSerializableMapping] = Field(default_factory=dict)
    overrides: JSONSerializableMapping = Field(default_factory=dict)
    user_data: JSONSerializableMapping = Field(default_factory=dict)
    step_run_errors: dict[str, str] = Field(default_factory=dict)
    triggered_by: str | None = None
    triggers: JSONSerializableMapping = Field(default_factory=dict)
    filter_payload: JSONSerializableMapping = Field(default_factory=dict)

    @field_validator(
        "input",
        "parents",
        "overrides",
        "user_data",
        "step_run_errors",
        "filter_payload",
        mode="before",
    )
    @classmethod
    def validate_fields(cls, v: Any) -> Any:
        return v or {}

    @model_validator(mode="after")
    def validate_filter_payload(self) -> "ActionPayload":
        self.filter_payload = self.triggers.get("filter_payload", {}) or {}

        return self


class ActionType(str, Enum):
    START_STEP_RUN = "START_STEP_RUN"
    CANCEL_STEP_RUN = "CANCEL_STEP_RUN"
    START_GET_GROUP_KEY = "START_GET_GROUP_KEY"


class Action(BaseModel):
    worker_id: str
    tenant_id: str
    workflow_run_id: str
    workflow_id: str | None = None
    workflow_version_id: str | None = None
    get_group_key_run_id: str
    job_id: str
    job_name: str
    job_run_id: str
    step_id: str
    step_run_id: str
    action_id: str
    action_type: ActionType
    retry_count: int
    action_payload: ActionPayload
    additional_metadata: JSONSerializableMapping = field(default_factory=dict)

    child_workflow_index: int | None = None
    child_workflow_key: str | None = None
    parent_workflow_run_id: str | None = None

    priority: int | None = None

    def _dump_payload_to_str(self) -> str:
        try:
            return json.dumps(self.action_payload.model_dump(), default=str)
        except Exception:
            return str(self.action_payload)

    def get_otel_attributes(self, config: "ClientConfig") -> dict[str, str | int]:
        try:
            payload_str = json.dumps(self.action_payload.model_dump(), default=str)
        except Exception:
            payload_str = str(self.action_payload)

        attrs: dict[OTelAttribute, str | int | None] = {
            OTelAttribute.TENANT_ID: self.tenant_id,
            OTelAttribute.WORKER_ID: self.worker_id,
            OTelAttribute.WORKFLOW_RUN_ID: self.workflow_run_id,
            OTelAttribute.STEP_ID: self.step_id,
            OTelAttribute.STEP_RUN_ID: self.step_run_id,
            OTelAttribute.RETRY_COUNT: self.retry_count,
            OTelAttribute.PARENT_WORKFLOW_RUN_ID: self.parent_workflow_run_id,
            OTelAttribute.CHILD_WORKFLOW_INDEX: self.child_workflow_index,
            OTelAttribute.CHILD_WORKFLOW_KEY: self.child_workflow_key,
            OTelAttribute.ACTION_PAYLOAD: payload_str,
            OTelAttribute.WORKFLOW_NAME: self.job_name,
            OTelAttribute.ACTION_NAME: self.action_id,
            OTelAttribute.GET_GROUP_KEY_RUN_ID: self.get_group_key_run_id,
            OTelAttribute.WORKFLOW_ID: self.workflow_id,
            OTelAttribute.WORKFLOW_VERSION_ID: self.workflow_version_id,
        }

        return {
            f"hatchet.{k.value}": v
            for k, v in attrs.items()
            if v and k not in config.otel.excluded_attributes
        }

    @property
    def key(self) -> ActionKey:
        """
        This key is used to uniquely identify a single step run by its id + retry count.
        It's used when storing references to a task, a context, etc. in a dictionary so that
        we can look up those items in the dictionary by a unique key.
        """
        if self.action_type == ActionType.START_GET_GROUP_KEY:
            return f"{self.get_group_key_run_id}/{self.retry_count}"
        return f"{self.step_run_id}/{self.retry_count}"
