"""API response object."""

from __future__ import annotations

from typing import TYPE_CHECKING, Generic, TypeVar

from pydantic import BaseModel, Field, StrictBytes, StrictInt

if TYPE_CHECKING:
    from collections.abc import Mapping

T = TypeVar("T")


class ApiResponse(BaseModel, Generic[T]):
    """
    API response object
    """

    status_code: StrictInt = Field(description="HTTP status code")
    headers: Mapping[str, str] | None = Field(None, description="HTTP headers")
    data: T = Field(description="Deserialized data given the data type")
    raw_data: StrictBytes = Field(description="Raw data (HTTP response body)")

    model_config = {"arbitrary_types_allowed": True}
