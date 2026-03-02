"""Unit tests for REST transport exception translation.

These tests verify that urllib3 transport exceptions are correctly translated
to typed Hatchet REST exceptions while preserving:
- status=0 (no HTTP status was received)
- diagnostic information in reason (method, url, timeout)
- exception chaining via __cause__
- backward compatibility (all exceptions inherit from ApiException)
"""

from typing import Any, NoReturn, cast

import pytest
import urllib3.exceptions

from hatchet_sdk.clients.rest.configuration import Configuration
from hatchet_sdk.clients.rest.exceptions import (
    ApiException,
    RestConnectionError,
    RestProtocolError,
    RestTimeoutError,
    RestTLSError,
    RestTransportError,
)
from hatchet_sdk.clients.rest.rest import RESTClientObject


@pytest.fixture
def rest_client() -> Any:
    config = Configuration(host="http://localhost:8080")
    return cast(Any, RESTClientObject(config))


@pytest.fixture
def request_params() -> dict[str, Any]:
    return {
        "method": "GET",
        "url": "http://localhost:8080/api/test",
        "headers": {"Content-Type": "application/json"},
        "_request_timeout": 30,
    }


# --- Hierarchy tests ---


def test_hierarchy__rest_transport_error_inherits_from_api_exception() -> None:
    assert issubclass(RestTransportError, ApiException)


def test_hierarchy__rest_timeout_error_inherits_from_transport_error() -> None:
    assert issubclass(RestTimeoutError, RestTransportError)
    assert issubclass(RestTimeoutError, ApiException)


def test_hierarchy__rest_connection_error_inherits_from_transport_error() -> None:
    assert issubclass(RestConnectionError, RestTransportError)
    assert issubclass(RestConnectionError, ApiException)


def test_hierarchy__rest_tls_error_inherits_from_transport_error() -> None:
    assert issubclass(RestTLSError, RestTransportError)
    assert issubclass(RestTLSError, ApiException)


def test_hierarchy__rest_protocol_error_inherits_from_transport_error() -> None:
    assert issubclass(RestProtocolError, RestTransportError)
    assert issubclass(RestProtocolError, ApiException)


# --- Translation tests ---


def test_translation__ssl_error_raises_rest_tls_error(
    rest_client: Any,
    request_params: dict[str, Any],
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    original_exc = urllib3.exceptions.SSLError("SSL certificate verify failed")

    def mock_request(*args: Any, **kwargs: Any) -> NoReturn:
        raise original_exc

    monkeypatch.setattr(rest_client.pool_manager, "request", mock_request)

    with pytest.raises(RestTLSError) as exc_info:
        rest_client.request(**request_params)

    exc = exc_info.value
    assert exc.status == 0
    assert "SSLError" in exc.reason
    assert "method=GET" in exc.reason
    assert "url=http://localhost:8080/api/test" in exc.reason
    assert "timeout=30" in exc.reason
    assert exc.__cause__ is original_exc


def test_translation__connect_timeout_error_raises_rest_timeout_error(
    rest_client: Any,
    request_params: dict[str, Any],
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    original_exc = urllib3.exceptions.ConnectTimeoutError(
        None, "http://localhost:8080", "Connection timed out"
    )

    def mock_request(*args: Any, **kwargs: Any) -> NoReturn:
        raise original_exc

    monkeypatch.setattr(rest_client.pool_manager, "request", mock_request)

    with pytest.raises(RestTimeoutError) as exc_info:
        rest_client.request(**request_params)

    exc = exc_info.value
    assert exc.status == 0
    assert "ConnectTimeoutError" in exc.reason
    assert "method=GET" in exc.reason
    assert "url=http://localhost:8080/api/test" in exc.reason
    assert "timeout=30" in exc.reason
    assert exc.__cause__ is original_exc


def test_translation__read_timeout_error_raises_rest_timeout_error(
    rest_client: Any,
    request_params: dict[str, Any],
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    original_exc = urllib3.exceptions.ReadTimeoutError(
        cast(Any, None), "http://localhost:8080", "Read timed out"
    )

    def mock_request(*args: Any, **kwargs: Any) -> NoReturn:
        raise original_exc

    monkeypatch.setattr(rest_client.pool_manager, "request", mock_request)

    with pytest.raises(RestTimeoutError) as exc_info:
        rest_client.request(**request_params)

    exc = exc_info.value
    assert exc.status == 0
    assert "ReadTimeoutError" in exc.reason
    assert "method=GET" in exc.reason
    assert "url=http://localhost:8080/api/test" in exc.reason
    assert "timeout=30" in exc.reason
    assert exc.__cause__ is original_exc


def test_translation__max_retry_error_raises_rest_connection_error(
    rest_client: Any,
    request_params: dict[str, Any],
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    original_exc = urllib3.exceptions.MaxRetryError(
        cast(Any, None), "http://localhost:8080", Exception("Max retries exceeded")
    )

    def mock_request(*args: Any, **kwargs: Any) -> NoReturn:
        raise original_exc

    monkeypatch.setattr(rest_client.pool_manager, "request", mock_request)

    with pytest.raises(RestConnectionError) as exc_info:
        rest_client.request(**request_params)

    exc = exc_info.value
    assert exc.status == 0
    assert "MaxRetryError" in exc.reason
    assert "method=GET" in exc.reason
    assert "url=http://localhost:8080/api/test" in exc.reason
    assert "timeout=30" in exc.reason
    assert exc.__cause__ is original_exc


def test_translation__new_connection_error_raises_rest_connection_error(
    rest_client: Any,
    request_params: dict[str, Any],
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    original_exc = urllib3.exceptions.NewConnectionError(
        cast(Any, None), "Failed to establish a new connection"
    )

    def mock_request(*args: Any, **kwargs: Any) -> NoReturn:
        raise original_exc

    monkeypatch.setattr(rest_client.pool_manager, "request", mock_request)

    with pytest.raises(RestConnectionError) as exc_info:
        rest_client.request(**request_params)

    exc = exc_info.value
    assert exc.status == 0
    assert "NewConnectionError" in exc.reason
    assert "method=GET" in exc.reason
    assert "url=http://localhost:8080/api/test" in exc.reason
    assert "timeout=30" in exc.reason
    assert exc.__cause__ is original_exc


def test_translation__protocol_error_raises_rest_protocol_error(
    rest_client: Any,
    request_params: dict[str, Any],
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    original_exc = urllib3.exceptions.ProtocolError("Connection aborted")

    def mock_request(*args: Any, **kwargs: Any) -> NoReturn:
        raise original_exc

    monkeypatch.setattr(rest_client.pool_manager, "request", mock_request)

    with pytest.raises(RestProtocolError) as exc_info:
        rest_client.request(**request_params)

    exc = exc_info.value
    assert exc.status == 0
    assert "ProtocolError" in exc.reason
    assert "method=GET" in exc.reason
    assert "url=http://localhost:8080/api/test" in exc.reason
    assert "timeout=30" in exc.reason
    assert exc.__cause__ is original_exc


# --- Backward compatibility tests ---


def test_backward_compat__catching_api_exception_catches_tls_error(
    rest_client: Any,
    request_params: dict[str, Any],
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    def mock_request(*args: Any, **kwargs: Any) -> NoReturn:
        raise urllib3.exceptions.SSLError("SSL failed")

    monkeypatch.setattr(rest_client.pool_manager, "request", mock_request)

    with pytest.raises(ApiException) as exc_info:
        rest_client.request(**request_params)

    assert isinstance(exc_info.value, RestTLSError)


def test_backward_compat__catching_api_exception_catches_timeout_error(
    rest_client: Any,
    request_params: dict[str, Any],
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    def mock_request(*args: Any, **kwargs: Any) -> NoReturn:
        raise urllib3.exceptions.ConnectTimeoutError(None, "url", "timeout")

    monkeypatch.setattr(rest_client.pool_manager, "request", mock_request)

    with pytest.raises(ApiException) as exc_info:
        rest_client.request(**request_params)

    assert isinstance(exc_info.value, RestTimeoutError)


def test_backward_compat__catching_api_exception_catches_connection_error(
    rest_client: Any,
    request_params: dict[str, Any],
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    def mock_request(*args: Any, **kwargs: Any) -> NoReturn:
        raise urllib3.exceptions.NewConnectionError(
            cast(Any, None), "connection failed"
        )

    monkeypatch.setattr(rest_client.pool_manager, "request", mock_request)

    with pytest.raises(ApiException) as exc_info:
        rest_client.request(**request_params)

    assert isinstance(exc_info.value, RestConnectionError)


def test_backward_compat__catching_api_exception_catches_protocol_error(
    rest_client: Any,
    request_params: dict[str, Any],
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    def mock_request(*args: Any, **kwargs: Any) -> NoReturn:
        raise urllib3.exceptions.ProtocolError("protocol error")

    monkeypatch.setattr(rest_client.pool_manager, "request", mock_request)

    with pytest.raises(ApiException) as exc_info:
        rest_client.request(**request_params)

    assert isinstance(exc_info.value, RestProtocolError)


def test_backward_compat__catching_rest_transport_error_catches_all_subtypes(
    rest_client: Any,
    request_params: dict[str, Any],
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    exceptions_to_test = [
        (urllib3.exceptions.SSLError("ssl"), RestTLSError),
        (
            urllib3.exceptions.ConnectTimeoutError(None, "url", "msg"),
            RestTimeoutError,
        ),
        (
            urllib3.exceptions.ReadTimeoutError(cast(Any, None), "url", "msg"),
            RestTimeoutError,
        ),
        (
            urllib3.exceptions.MaxRetryError(cast(Any, None), "url", Exception("msg")),
            RestConnectionError,
        ),
        (
            urllib3.exceptions.NewConnectionError(cast(Any, None), "msg"),
            RestConnectionError,
        ),
        (urllib3.exceptions.ProtocolError("msg"), RestProtocolError),
    ]

    for urllib3_exc, expected_type in exceptions_to_test:

        def mock_request(
            *args: Any, _exc: Exception = urllib3_exc, **kwargs: Any
        ) -> NoReturn:
            raise _exc

        monkeypatch.setattr(rest_client.pool_manager, "request", mock_request)

        with pytest.raises(RestTransportError) as exc_info:
            rest_client.request(**request_params)

        assert isinstance(
            exc_info.value, expected_type
        ), f"Expected {expected_type.__name__} for {type(urllib3_exc).__name__}"


# --- Diagnostics tests ---


def test_diagnostics__reason_includes_all_diagnostic_fields(
    rest_client: Any,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    def mock_request(*args: Any, **kwargs: Any) -> NoReturn:
        raise urllib3.exceptions.SSLError("test error message")

    monkeypatch.setattr(rest_client.pool_manager, "request", mock_request)

    with pytest.raises(RestTLSError) as exc_info:
        rest_client.request(
            method="POST",
            url="https://example.com/api/v1/resource",
            headers={},
            _request_timeout=(5, 30),
        )

    reason = exc_info.value.reason
    assert "SSLError" in reason
    assert "test error message" in reason
    assert "method=POST" in reason
    assert "url=https://example.com/api/v1/resource" in reason
    assert "timeout=(5, 30)" in reason


def test_diagnostics__reason_handles_none_timeout(
    rest_client: Any,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    def mock_request(*args: Any, **kwargs: Any) -> NoReturn:
        raise urllib3.exceptions.NewConnectionError(
            cast(Any, None), "connection refused"
        )

    monkeypatch.setattr(rest_client.pool_manager, "request", mock_request)

    with pytest.raises(RestConnectionError) as exc_info:
        rest_client.request(
            method="GET",
            url="http://localhost/test",
            headers={},
            _request_timeout=None,
        )

    reason = exc_info.value.reason
    assert "timeout=None" in reason


# --- http_method attribute tests ---


def test_http_method__tls_error_has_http_method(
    rest_client: Any,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    def mock_request(*args: Any, **kwargs: Any) -> NoReturn:
        raise urllib3.exceptions.SSLError("SSL failed")

    monkeypatch.setattr(rest_client.pool_manager, "request", mock_request)

    with pytest.raises(RestTLSError) as exc_info:
        rest_client.request(
            method="POST",
            url="https://example.com/api",
            headers={},
        )

    assert exc_info.value.http_method == "POST"


def test_http_method__timeout_error_has_http_method(
    rest_client: Any,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    def mock_request(*args: Any, **kwargs: Any) -> NoReturn:
        raise urllib3.exceptions.ConnectTimeoutError(None, "url", "timeout")

    monkeypatch.setattr(rest_client.pool_manager, "request", mock_request)

    with pytest.raises(RestTimeoutError) as exc_info:
        rest_client.request(
            method="GET",
            url="http://localhost/test",
            headers={},
        )

    assert exc_info.value.http_method == "GET"


def test_http_method__connection_error_has_http_method(
    rest_client: Any,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    def mock_request(*args: Any, **kwargs: Any) -> NoReturn:
        raise urllib3.exceptions.NewConnectionError(
            cast(Any, None), "connection failed"
        )

    monkeypatch.setattr(rest_client.pool_manager, "request", mock_request)

    with pytest.raises(RestConnectionError) as exc_info:
        rest_client.request(
            method="DELETE",
            url="http://localhost/test",
            headers={},
        )

    assert exc_info.value.http_method == "DELETE"


def test_http_method__protocol_error_has_http_method(
    rest_client: Any,
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    def mock_request(*args: Any, **kwargs: Any) -> NoReturn:
        raise urllib3.exceptions.ProtocolError("protocol error")

    monkeypatch.setattr(rest_client.pool_manager, "request", mock_request)

    with pytest.raises(RestProtocolError) as exc_info:
        rest_client.request(
            method="PUT",
            url="http://localhost/test",
            headers={},
        )

    assert exc_info.value.http_method == "PUT"
