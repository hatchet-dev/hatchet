"""Unit tests for tenacity transport error retry behavior.

Tests verify:
1. Default behavior: RestTransportError is NOT retried (even for GET)
2. Opt-in behavior: RestTransportError retried for configured methods only
3. Existing HTTP error retry behavior unchanged
4. Method extraction from exception reason strings
"""

import pytest

from hatchet_sdk.clients.rest.exceptions import (
    NotFoundException,
    RestConnectionError,
    RestProtocolError,
    RestTimeoutError,
    RestTLSError,
    RestTransportError,
    ServiceException,
)
from hatchet_sdk.clients.rest.tenacity_utils import (
    _extract_method_from_reason,
    tenacity_should_retry,
)
from hatchet_sdk.config import TenacityConfig

# --- Default behavior tests (transport errors NOT retried) ---


@pytest.mark.parametrize(
    "exc_class",
    [RestTransportError, RestTimeoutError],
    ids=["base-class", "subclass"],
)
def test_default__transport_errors_not_retried(exc_class: type) -> None:
    """By default, RestTransportError and subclasses should not be retried."""
    exc = exc_class(status=0, reason="method=GET\nurl=http://test")
    config = TenacityConfig()
    assert tenacity_should_retry(exc, config) is False


# --- Opt-in behavior tests (transport errors retried for allowed methods) ---


@pytest.mark.parametrize(
    "method",
    ["GET", "DELETE"],
    ids=["get", "delete"],
)
def test_optin__idempotent_methods_retried(method: str) -> None:
    """When enabled, GET and DELETE requests with transport errors should be retried."""
    exc = RestTimeoutError(status=0, reason=f"method={method}\nurl=http://test")
    config = TenacityConfig(retry_transport_errors=True)
    assert tenacity_should_retry(exc, config) is True


@pytest.mark.parametrize(
    "method",
    ["POST", "PUT", "PATCH"],
    ids=["post", "put", "patch"],
)
def test_optin__non_idempotent_methods_not_retried(method: str) -> None:
    """Non-idempotent requests should not be retried even when transport retry is enabled."""
    exc = RestTimeoutError(status=0, reason=f"method={method}\nurl=http://test")
    config = TenacityConfig(retry_transport_errors=True)
    assert tenacity_should_retry(exc, config) is False


def test_optin__custom_methods_list() -> None:
    """Custom retry_transport_methods should be honored."""
    exc = RestTimeoutError(status=0, reason="method=POST\nurl=http://test")
    config = TenacityConfig(
        retry_transport_errors=True,
        retry_transport_methods=["POST"],
    )
    assert tenacity_should_retry(exc, config) is True


def test_optin__custom_methods_excludes_default() -> None:
    """Custom retry_transport_methods can exclude default methods like GET."""
    exc = RestTimeoutError(status=0, reason="method=GET\nurl=http://test")
    config = TenacityConfig(
        retry_transport_errors=True,
        retry_transport_methods=["DELETE"],
    )
    assert tenacity_should_retry(exc, config) is False


# --- Regression tests: existing HTTP error retry behavior unchanged ---


@pytest.mark.parametrize(
    ("exc", "desc"),
    [
        (ServiceException(status=500, reason="Internal Server Error"), "5xx"),
        (NotFoundException(status=404, reason="Not Found"), "404"),
    ],
    ids=["service-exception", "not-found"],
)
def test_regression__http_errors_still_retried(exc: Exception, desc: str) -> None:
    """ServiceException (5xx) and NotFoundException (404) should still be retried."""
    config = TenacityConfig()
    assert tenacity_should_retry(exc, config) is True


def test_regression__backward_compat_no_config() -> None:
    """ServiceException should be retried even without config (backward compat)."""
    exc = ServiceException(status=500, reason="Internal Server Error")
    assert tenacity_should_retry(exc) is True


# --- Unit tests for _extract_method_from_reason ---


@pytest.mark.parametrize(
    ("reason", "expected"),
    [
        ("method=GET\nurl=http://test", "GET"),
        ("method=POST\nurl=http://test", "POST"),
        ("method=delete\nurl=http://test", "delete"),
        ("prefix method=PUT suffix", "PUT"),
        ("some error without method", None),
        ("method=\nurl=http://test", None),
        ("", None),
        (None, None),
    ],
    ids=[
        "get-uppercase",
        "post-uppercase",
        "lowercase-preserved",
        "embedded-in-text",
        "no-method-field",
        "empty-method-value",
        "empty-string",
        "none",
    ],
)
def test_extract_method__parses_reason(
    reason: str | None, expected: str | None
) -> None:
    """_extract_method_from_reason should correctly parse HTTP method from reason."""
    assert _extract_method_from_reason(reason) == expected


# --- Edge cases for retry behavior ---


@pytest.mark.parametrize(
    "reason",
    ["some error without method", "", None],
    ids=["no-method-field", "empty-string", "none"],
)
def test_edge__unparseable_reason_not_retried(reason: str | None) -> None:
    """If method cannot be extracted from reason, should not retry."""
    exc = RestTimeoutError(status=0, reason=reason)
    config = TenacityConfig(retry_transport_errors=True)
    assert tenacity_should_retry(exc, config) is False


def test_edge__case_insensitive_method_matching() -> None:
    """Method matching should be case-insensitive."""
    exc = RestTimeoutError(status=0, reason="method=get\nurl=http://test")
    config = TenacityConfig(retry_transport_errors=True)
    assert tenacity_should_retry(exc, config) is True


# --- Config defaults tests ---


def test_config__default_retry_transport_errors_is_false() -> None:
    """retry_transport_errors should default to False."""
    config = TenacityConfig()
    assert config.retry_transport_errors is False


def test_config__default_retry_transport_methods() -> None:
    """retry_transport_methods should default to GET and DELETE."""
    config = TenacityConfig()
    assert set(config.retry_transport_methods) == {"GET", "DELETE"}
