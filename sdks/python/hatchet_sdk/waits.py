from abc import ABC, abstractmethod
from datetime import timedelta
from enum import Enum
from typing import Generic
from uuid import uuid4

from pydantic import BaseModel, Field

from hatchet_sdk.contracts.v1.shared.condition_pb2 import Action as ProtoAction
from hatchet_sdk.contracts.v1.shared.condition_pb2 import (
    BaseMatchCondition,
    ParentOverrideMatchCondition,
    SleepMatchCondition,
    UserEventMatchCondition,
)
from hatchet_sdk.runnables.task import Task
from hatchet_sdk.runnables.types import R, TWorkflowInput
from hatchet_sdk.utils.proto_enums import convert_python_enum_to_proto


class Action(Enum):
    CREATE = 0
    QUEUE = 1
    CANCEL = 2
    SKIP = 3


class BaseCondition(BaseModel):
    event_key: str | None = None
    readable_data_key: str | None = None
    action: Action | None = None
    or_group_id: str | None = None
    expression: str | None = None

    def to_pb(self) -> BaseMatchCondition:
        return BaseMatchCondition(
            event_key=self.event_key,
            readable_data_key=self.readable_data_key,
            action=(
                str(x)
                if (x := convert_python_enum_to_proto(self.action, ProtoAction))
                else None
            ),
            or_group_id=self.or_group_id,
            expression=self.expression,
        )


class Condition(BaseModel, ABC):
    base: BaseCondition

    @abstractmethod
    def to_pb(
        self,
    ) -> UserEventMatchCondition | ParentOverrideMatchCondition | SleepMatchCondition:
        pass


class UserEventCondition(Condition):
    event_key: str
    expression: str

    def to_pb(self) -> UserEventMatchCondition:
        return UserEventMatchCondition(
            base=self.base.to_pb(),
            user_event_key=self.event_key,
        )


class ParentCondition(Condition, Generic[TWorkflowInput, R]):
    parent: Task[TWorkflowInput, R]

    def to_pb(self) -> ParentOverrideMatchCondition:
        return ParentOverrideMatchCondition(
            base=self.base.to_pb(),
            parent_readable_id=self.parent.name,
        )


class SleepCondition(Condition):
    duration: timedelta

    def to_pb(self) -> SleepMatchCondition:
        return SleepMatchCondition(
            base=self.base.to_pb(),
            sleep_for=str(self.duration.seconds),
        )


def generate_or_group_id() -> str:
    return str(uuid4())


class OrGroup(BaseModel):
    or_group_id: str = Field(default_factory=generate_or_group_id)
    conditions: list[Condition]


def or_(*conditions: Condition) -> OrGroup:
    return OrGroup(conditions=list(conditions))
