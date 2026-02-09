from logging import getLogger
from types import MappingProxyType
from typing import Any

logger = getLogger(__name__)


HATCHET_PYDANTIC_SENTINEL = MappingProxyType({"is_hatchet_serialization_context": True})
""" Sentinel value used to identify Hatchet serialization contexts in pydantic ValidationInfo.context.

Use `is_in_hatchet_serialization_context` instead of this directly
"""


def is_in_hatchet_serialization_context(context: Any) -> bool:
    """
    Determines from pydantic ValidationInfo.context whether we are in a Hatchet serialization context.

    Example:
    ```python

    ```
    """
    if context == HATCHET_PYDANTIC_SENTINEL:
        return True
    if hasattr(context, "context"):
        logger.warning(
            "The serialization context has a nested context attribute"
            ", which indicates that it's the ValidationInfo being tested and not ValidationInfo.context."
        )
    return False
