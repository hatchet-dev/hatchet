"""Unit tests for tenacity transport error retry behavior.

Tests verify:
1. Default behavior: RestTransportError is NOT retried (even for GET)
2. Opt-in behavior: RestTransportError retried for configured methods only
3. Existing HTTP error retry behavior unchanged
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
from hatchet_sdk.clients.rest.tenacity_utils import tenacity_should_retry
from hatchet_sdk.config import TenacityConfig

# --- Default behavior tests (transport errors NOT retried) ---


def test_default__rest_transport_error_not_retried() -> None:
    """By default, RestTransportError should NOT be retried."""
    exc = RestTransportError(status=0, reason="method=GET\nurl=http://test")
    config = TenacityConfig()
    assert tenacity_should_retry(exc, config) is False


def test_default__rest_timeout_error_not_retried() -> None:
    """By default, RestTimeoutError should NOT be retried."""
    exc = RestTimeoutError(status=0, reason="method=GET\nurl=http://test")
    config = TenacityConfig()
    assert tenacity_should_retry(exc, config) is False


def test_default__rest_connection_error_not_retried() -> None:
    """By default, RestConnectionError should NOT be retried."""
    exc = RestConnectionError(status=0, reason="method=GET\nurl=http://test")
    config = TenacityConfig()
    assert tenacity_should_retry(exc, config) is False


def test_default__rest_tls_error_not_retried() -> None:
    """By default, RestTLSError should NOT be retried."""
    exc = RestTLSError(status=0, reason="method=GET\nurl=http://test")
    config = TenacityConfig()
    assert tenacity_should_retry(exc, config) is False


def test_default__rest_protocol_error_not_retried() -> None:
    """By default, RestProtocolError should NOT be retried."""
    exc = RestProtocolError(status=0, reason="method=GET\nurl=http://test")
    config = TenacityConfig()
    assert tenacity_should_retry(exc, config) is False


# --- Opt-in behavior tests (transport errors retried for allowed methods) ---


def test_optin__get_request_retried() -> None:
    """When enabled, GET requests with transport errors should be retried."""
    exc = RestTimeoutError(status=0, reason="method=GET\nurl=http://test")
    config = TenacityConfig(retry_transport_errors=True)
    assert tenacity_should_retry(exc, config) is True


def test_optin__delete_request_retried() -> None:
    """When enabled, DELETE requests with transport errors should be retried."""
    exc = RestConnectionError(status=0, reason="method=DELETE\nurl=http://test")
    config = TenacityConfig(retry_transport_errors=True)
    assert tenacity_should_retry(exc, config) is True


def test_optin__post_request_not_retried() -> None:
    """POST requests should NOT be retried even when transport retry is enabled."""
    exc = RestTimeoutError(status=0, reason="method=POST\nurl=http://test")
    config = TenacityConfig(retry_transport_errors=True)
    assert tenacity_should_retry(exc, config) is False


def test_optin__put_request_not_retried() -> None:
    """PUT requests should NOT be retried even when transport retry is enabled."""
    exc = RestConnectionError(status=0, reason="method=PUT\nurl=http://test")
    config = TenacityConfig(retry_transport_errors=True)
    assert tenacity_should_retry(exc, config) is False


def test_optin__patch_request_not_retried() -> None:
    """PATCH requests should NOT be retried even when transport retry is enabled."""
    exc = RestProtocolError(status=0, reason="method=PATCH\nurl=http://test")
    config = TenacityConfig(retry_transport_errors=True)
    assert tenacity_should_retry(exc, config) is False


def test_optin__custom_methods_list() -> None:
    """Custom retry_transport_methods should be honored."""
    exc = RestTimeoutError(status=0, reason="method=POST\nurl=http://test")
    config = TenacityConfig(
        retry_transport_errors=True,
        retry_transport_methods=["POST"],  # allow POST explicitly
    )
    assert tenacity_should_retry(exc, config) is True


def test_optin__custom_methods_excludes_get() -> None:
    """Custom retry_transport_methods can exclude GET."""
    exc = RestTimeoutError(status=0, reason="method=GET\nurl=http://test")
    config = TenacityConfig(
        retry_transport_errors=True,
        retry_transport_methods=["DELETE"],  # only DELETE, not GET
    )
    assert tenacity_should_retry(exc, config) is False


# --- Regression tests: existing HTTP error retry behavior unchanged ---


def test_regression__service_exception_still_retried() -> None:
    """ServiceException (5xx) should still be retried."""
    exc = ServiceException(status=500, reason="Internal Server Error")
    config = TenacityConfig()
    assert tenacity_should_retry(exc, config) is True


def test_regression__not_found_exception_still_retried() -> None:
    """NotFoundException (404) should still be retried."""
    exc = NotFoundException(status=404, reason="Not Found")
    config = TenacityConfig()
    assert tenacity_should_retry(exc, config) is True


def test_regression__service_exception_retried_without_config() -> None:
    """ServiceException should be retried even without config (backward compat)."""
    exc = ServiceException(status=500, reason="Internal Server Error")
    # Call without config to test backward compatibility
    assert tenacity_should_retry(exc) is True


# --- Edge cases ---


def test_edge__missing_method_in_reason_not_retried() -> None:
    """If method cannot be extracted from reason, should not retry."""
    exc = RestTimeoutError(status=0, reason="some error without method")
    config = TenacityConfig(retry_transport_errors=True)
    assert tenacity_should_retry(exc, config) is False


def test_edge__empty_reason_not_retried() -> None:
    """If reason is empty, should not retry."""
    exc = RestConnectionError(status=0, reason="")
    config = TenacityConfig(retry_transport_errors=True)
    assert tenacity_should_retry(exc, config) is False


def test_edge__none_reason_not_retried() -> None:
    """If reason is None, should not retry."""
    exc = RestTLSError(status=0, reason=None)
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
