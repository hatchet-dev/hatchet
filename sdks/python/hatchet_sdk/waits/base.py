from enum import Enum

from pydantic import BaseModel

from hatchet_sdk.contracts.v1.shared.condition_pb2 import Action as ProtoAction
from hatchet_sdk.contracts.v1.shared.condition_pb2 import BaseMatchCondition
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


class Condition(BaseModel):
    base: BaseCondition
