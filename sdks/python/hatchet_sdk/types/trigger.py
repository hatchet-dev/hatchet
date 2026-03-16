import warnings
from warnings import warn

from pydantic import BaseModel, Field, field_validator, model_validator

from hatchet_sdk.types.labels import DesiredWorkerLabel
from hatchet_sdk.types.priority import Priority
from hatchet_sdk.utils.typing import JSONSerializableMapping


class ScheduleWorkflowOptions(BaseModel):
    child_key: str | None = None
    additional_metadata: JSONSerializableMapping = Field(default_factory=dict)
    priority: int | Priority | None = None


class ScheduleTriggerWorkflowOptions(ScheduleWorkflowOptions):
    parent_id: str | None = None
    parent_step_run_id: str | None = None
    child_index: int | None = None
    namespace: str | None = None

    @model_validator(mode="after")
    def validate_options(self) -> "ScheduleTriggerWorkflowOptions":
        if self.parent_id is not None:
            warn(
                "The `parent_id` property is internal and should not be used directly. It will be removed in v2.0.0.",
                DeprecationWarning,
                stacklevel=2,
            )

        if self.parent_step_run_id is not None:
            warnings.warn(
                "The `parent_step_run_id` property is internal and should not be used directly. It will be removed in v2.0.0.",
                DeprecationWarning,
                stacklevel=2,
            )

        if self.namespace:
            warnings.warn(
                "The `namespace` parameter is deprecated and will be removed in v2.0.0. The namespace should be set on the `ClientConfig`.",
                DeprecationWarning,
                stacklevel=2,
            )

        return self


class RunWorkflowOptions(ScheduleTriggerWorkflowOptions):
    desired_worker_id: str | None = None
    sticky: bool = False
    key: str | None = None
    desired_worker_label: (
        dict[str, DesiredWorkerLabel] | list[DesiredWorkerLabel] | None
    ) = None


class TriggerWorkflowOptions(RunWorkflowOptions):
    @model_validator(mode="after")
    def validate_options(self) -> "TriggerWorkflowOptions":
        warn(
            "`TriggerWorkflowOptions` is deprecated. It will be removed in v2.0.0. Use `RunWorkflowOptions` instead.",
            DeprecationWarning,
            stacklevel=2,
        )

        return self


class WorkflowRunTriggerConfig(BaseModel):
    workflow_name: str
    input: str | None
    options: RunWorkflowOptions
    key: str | None = None


class PushEventOptions(BaseModel):
    additional_metadata: JSONSerializableMapping = Field(default_factory=dict)
    namespace: str | None = None
    priority: int | Priority | None = None
    scope: str | None = None

    @field_validator("namespace", mode="before")
    @classmethod
    def validate_namespace(cls, v: str | None) -> str | None:
        if v:
            warnings.warn(
                "The `namespace` parameter is deprecated and will be removed in v2.0.0. The namespace should be set on the `ClientConfig`.",
                DeprecationWarning,
                stacklevel=2,
            )

        return v


class BulkPushEventWithMetadata(BaseModel):
    key: str = ""
    payload: JSONSerializableMapping = Field(default_factory=dict)
    additional_metadata: JSONSerializableMapping = Field(default_factory=dict)
    priority: int | Priority | None = None
    scope: str | None = None


class BulkPushEventOptions(BulkPushEventWithMetadata):
    namespace: str | None = None

    @field_validator("namespace", mode="before")
    @classmethod
    def validate_namespace(cls, v: str | None) -> str | None:
        if v:
            warnings.warn(
                "The `namespace` parameter is deprecated and will be removed in v2.0.0. The namespace should be set on the `ClientConfig`.",
                DeprecationWarning,
                stacklevel=2,
            )
        return v
