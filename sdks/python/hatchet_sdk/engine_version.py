from enum import Enum


class MinEngineVersion(str, Enum):
    """Minimum engine version required for a given feature."""

    SLOT_CONFIG = "v0.78.23"
    DURABLE_EVICTION = "v0.80.0"
