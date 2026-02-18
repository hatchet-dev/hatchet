from enum import Enum


class SlotType(str, Enum):
    DEFAULT = "default"
    DURABLE = "durable"
