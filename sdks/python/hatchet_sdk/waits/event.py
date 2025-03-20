from pydantic import BaseModel

from hatchet_sdk.contracts.v1.shared.condition_pb2 import UserEventMatchCondition
from hatchet_sdk.waits.base import Condition


class UserEvent(BaseModel):
    event_key: str
    expression: str


class UserEventCondition(Condition):
    event_key: str
    expression: str

    def to_pb(self) -> UserEventMatchCondition:
        return UserEventMatchCondition(
            base=self.base.to_pb(),
            user_event_key=self.event_key,
        )
