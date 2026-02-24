"""Unit tests for the tenacity retry predicate (tenacity_should_retry).

These tests verify which REST exceptions and gRPC status codes are treated as retryable.
"""

import grpc
import pytest

from hatchet_sdk.clients.rest.exceptions import (
    BadRequestException,
    ForbiddenException,
    NotFoundException,
    RestConnectionError,
    RestProtocolError,
    RestTimeoutError,
    RestTLSError,
    RestTransportError,
    ServiceException,
    TooManyRequestsException,
    UnauthorizedException,
)
from hatchet_sdk.clients.rest.tenacity_utils import tenacity_should_retry


@pytest.mark.parametrize(
    ("exc", "expected"),
    [
        pytest.param(
            ServiceException(status=500, reason="Internal Server Error"),
            True,
            id="ServiceException (HTTP 5xx) should be retried",
        ),
        pytest.param(
            NotFoundException(status=404, reason="Not Found"),
            True,
            id="NotFoundException (HTTP 404) should be retried",
        ),
        pytest.param(
            BadRequestException(status=400, reason="Bad Request"),
            False,
            id="BadRequestException (HTTP 400) should not be retried",
        ),
        pytest.param(
            UnauthorizedException(status=401, reason="Unauthorized"),
            False,
            id="UnauthorizedException (HTTP 401) should not be retried",
        ),
        pytest.param(
            ForbiddenException(status=403, reason="Forbidden"),
            False,
            id="ForbiddenException (HTTP 403) should not be retried",
        ),
        pytest.param(
            TooManyRequestsException(status=429, reason="Too Many Requests"),
            False,
            id="TooManyRequestsException (HTTP 429) should not be retried by default",
        ),
    ],
)
def test_rest__exception_retry_behavior(exc: BaseException, expected: bool) -> None:
    """Test that REST exceptions have the expected retry behavior."""
    assert tenacity_should_retry(exc) is expected


@pytest.mark.parametrize(
    ("exc", "expected"),
    [
        pytest.param(
            RestTransportError(status=0, reason="Transport error"),
            False,
            id="RestTransportError (base class) should not be retried",
        ),
        pytest.param(
            RestTimeoutError(status=0, reason="Connection timed out"),
            False,
            id="RestTimeoutError should not be retried",
        ),
        pytest.param(
            RestConnectionError(status=0, reason="Connection refused"),
            False,
            id="RestConnectionError should not be retried",
        ),
        pytest.param(
            RestTLSError(status=0, reason="SSL certificate verify failed"),
            False,
            id="RestTLSError should not be retried",
        ),
        pytest.param(
            RestProtocolError(status=0, reason="Connection aborted"),
            False,
            id="RestProtocolError should not be retried",
        ),
    ],
)
def test_transport__error_retry_behavior(exc: BaseException, expected: bool) -> None:
    """Test that REST transport errors have the expected retry behavior."""
    assert tenacity_should_retry(exc) is expected


@pytest.mark.parametrize(
    ("exc", "expected"),
    [
        pytest.param(
            RuntimeError("Something went wrong"),
            False,
            id="RuntimeError should not be retried",
        ),
        pytest.param(
            ValueError("Invalid value"),
            False,
            id="ValueError should not be retried",
        ),
        pytest.param(
            Exception("Generic error"),
            False,
            id="Generic Exception should not be retried",
        ),
    ],
)
def test_generic__exception_retry_behavior(exc: BaseException, expected: bool) -> None:
    """Test that generic exceptions have the expected retry behavior."""
    assert tenacity_should_retry(exc) is expected


class FakeRpcError(grpc.RpcError):
    """A fake gRPC RpcError for testing without real gRPC infrastructure."""

    def __init__(self, code: grpc.StatusCode) -> None:
        self._code = code
        super().__init__()

    def code(self) -> grpc.StatusCode:
        return self._code


@pytest.mark.parametrize(
    ("status_code", "expected"),
    [
        # Status codes that should be retried (transient/server errors)
        pytest.param(
            grpc.StatusCode.UNAVAILABLE,
            True,
            id="UNAVAILABLE should be retried (transient error)",
        ),
        pytest.param(
            grpc.StatusCode.DEADLINE_EXCEEDED,
            True,
            id="DEADLINE_EXCEEDED should be retried (transient error)",
        ),
        pytest.param(
            grpc.StatusCode.INTERNAL,
            True,
            id="INTERNAL should be retried (server error)",
        ),
        pytest.param(
            grpc.StatusCode.RESOURCE_EXHAUSTED,
            True,
            id="RESOURCE_EXHAUSTED should be retried",
        ),
        pytest.param(
            grpc.StatusCode.ABORTED,
            True,
            id="ABORTED should be retried",
        ),
        pytest.param(
            grpc.StatusCode.UNKNOWN,
            True,
            id="UNKNOWN should be retried",
        ),
        # Status codes that should not be retried (permanent/client errors)
        pytest.param(
            grpc.StatusCode.UNIMPLEMENTED,
            False,
            id="UNIMPLEMENTED should not be retried (permanent error)",
        ),
        pytest.param(
            grpc.StatusCode.NOT_FOUND,
            False,
            id="NOT_FOUND should not be retried (permanent error)",
        ),
        pytest.param(
            grpc.StatusCode.INVALID_ARGUMENT,
            False,
            id="INVALID_ARGUMENT should not be retried (client error)",
        ),
        pytest.param(
            grpc.StatusCode.ALREADY_EXISTS,
            False,
            id="ALREADY_EXISTS should not be retried (permanent error)",
        ),
        pytest.param(
            grpc.StatusCode.UNAUTHENTICATED,
            False,
            id="UNAUTHENTICATED should not be retried (auth error)",
        ),
        pytest.param(
            grpc.StatusCode.PERMISSION_DENIED,
            False,
            id="PERMISSION_DENIED should not be retried (auth error)",
        ),
    ],
)
def test_grpc__status_code_retry_behavior(
    status_code: grpc.StatusCode, expected: bool
) -> None:
    """Test that gRPC status codes have the expected retry behavior."""
    exc = FakeRpcError(status_code)
    assert tenacity_should_retry(exc) is expected
