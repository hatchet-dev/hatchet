from google.protobuf.message import Message

from hatchet_sdk.exceptions import PayloadTooLargeError

try:
    import grpc
except ImportError:  # pragma: no cover
    grpc = None  # type: ignore[assignment]


def raise_if_payload_too_large(e: Exception, message: Message) -> None:
    """Re-raises `e` as a `PayloadTooLargeError` (reporting the exact serialized size of
    `message`) if `e` is a gRPC RESOURCE_EXHAUSTED error caused by an oversized outgoing
    message. Otherwise this is a no-op, leaving the original error to propagate normally.
    """
    code = getattr(e, "code", None)
    details = getattr(e, "details", None)

    if code is None or details is None:
        return

    if grpc is not None and code() != grpc.StatusCode.RESOURCE_EXHAUSTED:
        return

    detail_str = details() or ""
    lowered = detail_str.lower()
    if "larger than" not in lowered and "too large" not in lowered:
        return

    raise PayloadTooLargeError(message.ByteSize(), detail_str) from e
