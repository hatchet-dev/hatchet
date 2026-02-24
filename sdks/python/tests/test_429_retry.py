"""Unit tests for HTTP 429 Too Many Requests retry behavior."""

import pytest

from hatchet_sdk.clients.rest.exceptions import (
    ApiException,
    NotFoundException,
    ServiceException,
    TooManyRequestsException,
)
from hatchet_sdk.clients.rest.tenacity_utils import tenacity_should_retry
from hatchet_sdk.config import TenacityConfig


class FakeHttpResponse:
    """Minimal fake HTTP response for testing ApiException.from_response()."""

    def __init__(self, status: int, reason: str = "", data: bytes = b"") -> None:
        self.status = status
        self.reason = reason
        self.data = data

    def getheaders(self) -> dict[str, str]:
        return {}


def test_from_response__429_raises_too_many_requests_exception() -> None:
    """ApiException.from_response() should raise TooManyRequestsException for status 429."""
    http_resp = FakeHttpResponse(status=429, reason="Too Many Requests")

    with pytest.raises(TooManyRequestsException) as exc_info:
        ApiException.from_response(http_resp=http_resp, body=None, data=None)

    exc = exc_info.value
    assert exc.status == 429
    assert exc.reason == "Too Many Requests"


def test_default__429_not_retried() -> None:
    """By default (no config), TooManyRequestsException should NOT be retried."""
    exc = TooManyRequestsException(status=429, reason="Too Many Requests")
    assert tenacity_should_retry(exc) is False


def test_optin__429_retried_when_enabled() -> None:
    """TooManyRequestsException should be retried when retry_429=True."""
    exc = TooManyRequestsException(status=429, reason="Too Many Requests")
    config = TenacityConfig(retry_429=True)
    assert tenacity_should_retry(exc, config) is True


def test_regression__service_exception_still_retried() -> None:
    """ServiceException (5xx) should still be retried."""
    exc = ServiceException(status=500, reason="Internal Server Error")
    assert tenacity_should_retry(exc) is True


def test_regression__not_found_exception_still_retried() -> None:
    """NotFoundException (404) should still be retried."""
    exc = NotFoundException(status=404, reason="Not Found")
    assert tenacity_should_retry(exc) is True


def test_config__default_retry_429_is_false() -> None:
    """retry_429 should default to False."""
    config = TenacityConfig()
    assert config.retry_429 is False
