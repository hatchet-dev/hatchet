from datetime import timedelta

from pydantic import BaseModel

from hatchet_sdk.contracts.v1.shared.condition_pb2 import SleepMatchCondition
from hatchet_sdk.waits.base import Condition


class Sleep(BaseModel):
    duration: timedelta


class SleepCondition(Condition):
    duration: timedelta

    def to_pb(self) -> SleepMatchCondition:
        return SleepMatchCondition(
            base=self.base.to_pb(),
            sleep_for=str(self.duration.seconds),
        )
