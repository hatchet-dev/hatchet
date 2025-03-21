from abc import ABC, abstractmethod
from enum import Enum
from typing import TYPE_CHECKING
from uuid import uuid4

from pydantic import BaseModel, Field

from hatchet_sdk.contracts.v1.shared.condition_pb2 import Action as ProtoAction
from hatchet_sdk.contracts.v1.shared.condition_pb2 import (
    BaseMatchCondition,
    ParentOverrideMatchCondition,
    SleepMatchCondition,
    UserEventMatchCondition,
)
from hatchet_sdk.utils.proto_enums import convert_python_enum_to_proto
from hatchet_sdk.utils.timedelta_to_expression import Duration, timedelta_to_expr

if TYPE_CHECKING:
    from hatchet_sdk.runnables.task import Task
    from hatchet_sdk.runnables.types import R, TWorkflowInput


def generate_or_group_id() -> str:
    return str(uuid4())


class Action(Enum):
    CREATE = 0
    QUEUE = 1
    CANCEL = 2
    SKIP = 3


class BaseCondition(BaseModel):
    readable_data_key: str
    action: Action | None = None
    or_group_id: str = Field(default_factory=generate_or_group_id)
    expression: str | None = None

    def to_pb(self) -> BaseMatchCondition:
        return BaseMatchCondition(
            readable_data_key=self.readable_data_key,
            action=convert_python_enum_to_proto(self.action, ProtoAction),  # type: ignore[arg-type]
            or_group_id=self.or_group_id,
            expression=self.expression,
        )


class Condition(ABC):
    def __init__(self, base: BaseCondition):
        self.base = base

    @abstractmethod
    def to_pb(
        self,
    ) -> UserEventMatchCondition | ParentOverrideMatchCondition | SleepMatchCondition:
        pass


class SleepCondition(Condition):
    def __init__(self, duration: Duration) -> None:
        super().__init__(
            BaseCondition(
                readable_data_key=f"sleep:{timedelta_to_expr(duration)}",
            )
        )

        self.duration = duration

    def to_pb(self) -> SleepMatchCondition:
        return SleepMatchCondition(
            base=self.base.to_pb(),
            sleep_for=timedelta_to_expr(self.duration),
        )


class UserEventCondition(Condition):
    def __init__(self, event_key: str, expression: str | None = None) -> None:
        super().__init__(
            BaseCondition(
                readable_data_key=event_key,
                expression=expression,
            )
        )

        self.event_key = event_key
        self.expression = expression

    def to_pb(self) -> UserEventMatchCondition:
        return UserEventMatchCondition(
            base=self.base.to_pb(),
            user_event_key=self.event_key,
        )


class ParentCondition(Condition):
    def __init__(
        self, parent: "Task[TWorkflowInput, R]", expression: str | None = None
    ) -> None:
        super().__init__(
            BaseCondition(readable_data_key=parent.name, expression=expression)
        )

        self.parent = parent

    def to_pb(self) -> ParentOverrideMatchCondition:
        return ParentOverrideMatchCondition(
            base=self.base.to_pb(),
            parent_readable_id=self.parent.name,
        )


class OrGroup:
    def __init__(self, conditions: list[Condition]) -> None:
        self.or_group_id = generate_or_group_id()
        self.conditions = conditions


def or_(*conditions: Condition) -> OrGroup:
    return OrGroup(conditions=list(conditions))
