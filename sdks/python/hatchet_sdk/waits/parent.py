from typing import Generic

from pydantic import BaseModel

from hatchet_sdk.contracts.v1.shared.condition_pb2 import ParentOverrideMatchCondition
from hatchet_sdk.runnables.task import Task
from hatchet_sdk.runnables.types import R, TWorkflowInput
from hatchet_sdk.waits.base import Condition


class Parent(BaseModel, Generic[TWorkflowInput, R]):
    parent: Task[TWorkflowInput, R]
    expression: str | None = None


class ParentCondition(Condition, Generic[TWorkflowInput, R]):
    parent: Task[TWorkflowInput, R]

    def to_pb(self) -> ParentOverrideMatchCondition:
        return ParentOverrideMatchCondition(
            base=self.base.to_pb(),
            parent_readable_id=self.parent.name,
        )
