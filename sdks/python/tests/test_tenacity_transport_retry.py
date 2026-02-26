"""Unit tests for tenacity transport error retry behavior.

Tests verify:
1. Default behavior: RestTransportError is NOT retried (even for GET)
2. Opt-in behavior: RestTransportError retried for configured methods only
3. Existing HTTP error retry behavior unchanged
4. HTTP method is read from exception's http_method attribute
"""

import pytest

from hatchet_sdk.clients.rest.exceptions import (
    NotFoundException,
    RestTimeoutError,
    RestTransportError,
    ServiceException,
)
from hatchet_sdk.clients.rest.tenacity_utils import tenacity_should_retry
from hatchet_sdk.config import TenacityConfig, HTTPMethod

# --- Default behavior tests (transport errors NOT retried) ---


@pytest.mark.parametrize(
    "exc_class",
    [RestTransportError, RestTimeoutError],
    ids=["base-class", "subclass"],
)
def test_default__transport_errors_not_retried(exc_class: type) -> None:
    """By default, RestTransportError and subclasses should not be retried."""
    exc = exc_class(status=0, reason="timeout", http_method=HTTPMethod.GET)
    config = TenacityConfig()
    assert tenacity_should_retry(exc, config) is False


# --- Opt-in behavior tests (transport errors retried for allowed methods) ---


@pytest.mark.parametrize(
    "method", [HTTPMethod.GET, HTTPMethod.DELETE], ids=["get", "delete"]
)
def test_optin__idempotent_methods_retried(method: HTTPMethod) -> None:
    """When enabled, GET and DELETE requests with transport errors should be retried."""
    exc = RestTimeoutError(status=0, reason="timeout", http_method=method)
    config = TenacityConfig(retry_transport_errors=True)
    assert tenacity_should_retry(exc, config) is True


@pytest.mark.parametrize(
    "method",
    [HTTPMethod.POST, HTTPMethod.PUT, HTTPMethod.PATCH],
    ids=["post", "put", "patch"],
)
def test_optin__non_idempotent_methods_not_retried(method: HTTPMethod) -> None:
    """Non-idempotent requests should not be retried even when transport retry is enabled."""
    exc = RestTimeoutError(status=0, reason="timeout", http_method=method)
    config = TenacityConfig(retry_transport_errors=True)
    assert tenacity_should_retry(exc, config) is False


def test_optin__custom_methods_list() -> None:
    """Custom retry_transport_methods should be honored."""
    exc = RestTimeoutError(status=0, reason="timeout", http_method=HTTPMethod.POST)
    config = TenacityConfig(
        retry_transport_errors=True,
        retry_transport_methods=[HTTPMethod.POST],
    )
    assert tenacity_should_retry(exc, config) is True


def test_optin__custom_methods_excludes_default() -> None:
    """Custom retry_transport_methods can exclude default methods like GET."""
    exc = RestTimeoutError(status=0, reason="timeout", http_method=HTTPMethod.GET)
    config = TenacityConfig(
        retry_transport_errors=True,
        retry_transport_methods=[HTTPMethod.DELETE],
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


# --- Edge cases for retry behavior ---


def test_edge__no_http_method_not_retried() -> None:
    """If http_method is None, should not retry even with retry_transport_errors=True."""
    exc = RestTimeoutError(status=0, reason="timeout", http_method=None)
    config = TenacityConfig(retry_transport_errors=True)
    assert tenacity_should_retry(exc, config) is False


def test_edge__enum_method_matching() -> None:
    """Method matching uses HTTPMethod enum values directly."""
    exc = RestTimeoutError(status=0, reason="timeout", http_method=HTTPMethod.GET)
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
    assert set(config.retry_transport_methods) == {HTTPMethod.GET, HTTPMethod.DELETE}
