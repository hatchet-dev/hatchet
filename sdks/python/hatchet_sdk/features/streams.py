import asyncio
import random
import threading
import time
import uuid
from collections.abc import AsyncGenerator, AsyncIterator, Callable, Generator
from concurrent.futures import Future, ThreadPoolExecutor
from contextlib import asynccontextmanager
from dataclasses import dataclass, field
from datetime import datetime, timedelta, timezone
from typing import TypeVar, cast

import grpc
from pydantic import BaseModel

from hatchet_sdk.clients.rest.api.streams_api import StreamsApi
from hatchet_sdk.clients.rest.exceptions import (
    ApiException,
    RestTransportError,
    TooManyRequestsException,
)
from hatchet_sdk.clients.v1.api_client import BaseRestClient
from hatchet_sdk.config import ClientConfig
from hatchet_sdk.connection import new_conn
from hatchet_sdk.contracts.v1 import streams_pb2 as streams_protos
from hatchet_sdk.contracts.v1.streams_pb2_grpc import V1StreamsStub
from hatchet_sdk.utils.api_auth import create_authorization_header

T = TypeVar("T")

## the server's limit for a payload sent over gRPC (MaxStreamMessagePayloadBytes)
MAX_INLINE_PAYLOAD_BYTES = 4 * 1024 * 1024 - 5 * 1024

## the server's limit for an uploaded payload (MaxStreamUploadedPayloadBytes)
MAX_UPLOADED_PAYLOAD_BYTES = 64 * 1024 * 1024

## the most publishes one PublishBatch request carries; the engine sets no limit of its own
MAX_PUBLISH_BATCH_ENTRIES = 500

## keeps a PublishBatch request under gRPC's 4 MiB limit with the headroom of a single max-size publish
MAX_PUBLISH_BATCH_BYTES = 4 * 1024 * 1024 - 5 * 1024

## an entry's tag and length prefixes, rounded up
PUBLISH_ENTRY_OVERHEAD_BYTES = 16

## an upload's ref, which a batch carries in place of its payload, rounded up
PAYLOAD_REF_BYTES = 256


@dataclass(frozen=True)
class StreamsConfig:
    payload_request_attempts: int = 3
    resubscribe_min_delay_seconds: float = 0.25
    resubscribe_max_delay_seconds: float = 10.0


## the server stored nothing, so the seq is unused
_REJECTED_BEFORE_STORING = {
    grpc.StatusCode.INVALID_ARGUMENT,
    grpc.StatusCode.RESOURCE_EXHAUSTED,
    grpc.StatusCode.UNAUTHENTICATED,
    grpc.StatusCode.PERMISSION_DENIED,
    ## an engine without the RPC
    grpc.StatusCode.UNIMPLEMENTED,
}

## retrying a subscribe that failed with these can't succeed
_PERMANENT_SUBSCRIBE_ERRORS = {
    grpc.StatusCode.INVALID_ARGUMENT,
    grpc.StatusCode.PERMISSION_DENIED,
    grpc.StatusCode.UNAUTHENTICATED,
    grpc.StatusCode.OUT_OF_RANGE,
    grpc.StatusCode.FAILED_PRECONDITION,
    grpc.StatusCode.NOT_FOUND,
}


class StreamEvent(BaseModel):
    payload: bytes
    """The message payload."""

    cursor: str
    """Resumes `events` right after this event."""

    created_at: datetime | None = None
    """When the message was stored."""


class StreamTopicMetadata(BaseModel):
    """What a durable stream topic retains; counts and the latest message cover the tenant's retention."""

    namespace: str
    topic: str
    tenant_id: str
    message_count: int

    latest_cursor: str | None = None
    """Resumes `events` after the newest retained message; None when none is retained."""

    last_published_at: datetime | None = None
    """When the newest retained message was stored; None when none is retained."""


def _grpc_code(err: BaseException) -> grpc.StatusCode | None:
    if isinstance(err, grpc.RpcError):
        code = err.code()
        if isinstance(code, grpc.StatusCode):
            return code

    return None


def _is_temporary_http_error(err: BaseException) -> bool:
    """A payload request that got no response, a 5xx or a 429 may succeed later."""
    if isinstance(err, (RestTransportError, TooManyRequestsException)):
        return True

    return (
        isinstance(err, ApiException) and err.status is not None and err.status >= 500
    )


def _should_resubscribe(err: BaseException, attempt: int) -> bool:
    if isinstance(err, ApiException):
        return _is_temporary_http_error(err)

    code = _grpc_code(err)

    if code is None or code in _PERMANENT_SUBSCRIBE_ERRORS:
        return False

    ## proxies answer 404 while an engine restarts, but on a first attempt it means no streams support
    if code == grpc.StatusCode.UNIMPLEMENTED:
        return attempt > 0

    return True


def _to_event(entry: streams_protos.StreamEntry, payload: bytes) -> StreamEvent:
    return StreamEvent(
        payload=payload,
        cursor=entry.cursor,
        created_at=(
            entry.created_at.ToDatetime(tzinfo=timezone.utc)
            if entry.HasField("created_at")
            else None
        ),
    )


def _iterate_in_thread(agen: AsyncGenerator[T, None]) -> Generator[T, None, None]:
    """Iterates an async generator from sync code on an event loop of its own."""
    loop = asyncio.new_event_loop()
    thread = threading.Thread(target=loop.run_forever, daemon=True)
    thread.start()

    reading: asyncio.Task[T] | None = None

    async def next_item() -> T:
        nonlocal reading
        reading = asyncio.current_task()
        return await agen.__anext__()

    async def close() -> None:
        ## interrupted mid-read: a running generator can't be closed until the read unwinds
        if reading is not None and not reading.done():
            reading.cancel()
            await asyncio.wait([reading])
        await agen.aclose()

    try:
        while True:
            try:
                item = asyncio.run_coroutine_threadsafe(next_item(), loop).result()
            except StopAsyncIteration:
                return
            yield item
    finally:
        asyncio.run_coroutine_threadsafe(close(), loop).result()
        loop.call_soon_threadsafe(loop.stop)
        thread.join()
        loop.close()


def _subscribe_request(
    namespace: str, topic: str, cursor: str | None
) -> streams_protos.SubscribeStreamRequest:
    if cursor is None:
        return streams_protos.SubscribeStreamRequest(namespace=namespace, topic=topic)

    return streams_protos.SubscribeStreamRequest(
        namespace=namespace, topic=topic, cursor=cursor
    )


@dataclass
class _Producer:
    producer_id: str = field(default_factory=lambda: str(uuid.uuid4()))
    seq: int = 0


@dataclass
class _QueuedPublish:
    payload: bytes
    send_by: float
    ## set instead of an inline payload for one too large for a gRPC message
    uploaded: "Future[str] | None"
    result: "Future[None]" = field(default_factory=Future)

    @property
    def request_bytes(self) -> int:
        return PAYLOAD_REF_BYTES if self.uploaded else len(self.payload)


@dataclass
class _PublishQueue:
    items: list[_QueuedPublish] = field(default_factory=list)
    wake: threading.Event = field(default_factory=threading.Event)


def _fitting_publishes(items: list[_QueuedPublish]) -> int:
    """Always at least one, so a max-size payload still goes out on its own."""
    total = 0
    count = 0

    while count < len(items) and count < MAX_PUBLISH_BATCH_ENTRIES:
        nxt = items[count].request_bytes + PUBLISH_ENTRY_OVERHEAD_BYTES
        if count > 0 and total + nxt > MAX_PUBLISH_BATCH_BYTES:
            break
        total += nxt
        count += 1

    return count


class StreamsClient(BaseRestClient):
    """
    The streams client publishes to and reads from durable topics. Unlike `runs.subscribe_to_stream`, topics are stored, independent of any run, and readers can resume from a cursor.
    """

    def __init__(
        self, config: ClientConfig, streams_config: StreamsConfig | None = None
    ) -> None:
        super().__init__(config)

        self._streams_config = streams_config or StreamsConfig()

        self._lock = threading.Lock()
        self._stub: V1StreamsStub | None = None
        self._upload_pool: ThreadPoolExecutor | None = None

        ## per (namespace, topic); seq only advances after a publish succeeds
        self._producers: dict[tuple[str, str], _Producer] = {}

        ## per (namespace, topic); one request per topic is in flight at a time so two never share a seq
        self._queues: dict[tuple[str, str], _PublishQueue] = {}

        ## set once the engine answers PublishBatch as unimplemented: it predates batching
        self._publish_batch_unsupported = False

    def _get_or_create_client(self) -> V1StreamsStub:
        with self._lock:
            if self._stub is None:
                self._stub = V1StreamsStub(new_conn(self.client_config, False))
            return self._stub

    ## an aio channel is bound to its event loop, and sync `events` runs each read on a loop of its own
    @asynccontextmanager
    async def _open_aio_client(self) -> AsyncIterator[V1StreamsStub]:
        channel = new_conn(self.client_config, True)
        try:
            yield V1StreamsStub(channel)
        finally:
            await channel.close()

    def _metadata(self) -> tuple[tuple[str, str]]:
        return create_authorization_header(self.client_config.token)

    def _resubscribe_delay(self, failures: int) -> float:
        delay = min(
            self._streams_config.resubscribe_min_delay_seconds
            * 2.0 ** min(failures, 16),
            self._streams_config.resubscribe_max_delay_seconds,
        )
        return delay * (0.5 + random.random() / 2)

    def _with_retries(self, request: Callable[[], T]) -> T:
        """Retries a 5xx or network failure; a 4xx won't succeed on retry."""
        attempt = 1
        while True:
            try:
                return request()
            except ApiException as e:
                temporary = isinstance(e, RestTransportError) or (
                    e.status is not None and e.status >= 500
                )
                if (
                    not temporary
                    or attempt >= self._streams_config.payload_request_attempts
                ):
                    raise
            attempt += 1

    def _upload_payload(self, payload: bytes) -> str:
        with self.client() as client:
            api = StreamsApi(client)
            return self._with_retries(
                lambda: api.v1_stream_payload_upload(
                    tenant=self.client_config.tenant_id,
                    payload=("payload", payload),
                ).ref
            )

    def _fetch_payload(self, payload_ref: str) -> bytes:
        with self.client() as client:
            api = StreamsApi(client)
            return bytes(
                self._with_retries(
                    lambda: api.v1_stream_payload_get(
                        tenant=self.client_config.tenant_id, ref=payload_ref
                    )
                )
            )

    def _enqueue_publish(
        self,
        topic: str,
        message: bytes | str,
        namespace: str,
        batch_timeout: timedelta | None,
    ) -> "Future[None]":
        payload = message.encode() if isinstance(message, str) else bytes(message)

        ## the engine would reject the whole request it shares, failing other publishes too
        if len(payload) == 0:
            raise ValueError("payload is empty; a message must carry at least one byte")

        if len(payload) > MAX_UPLOADED_PAYLOAD_BYTES:
            raise ValueError(
                f"payload exceeds maximum size of {MAX_UPLOADED_PAYLOAD_BYTES} bytes"
            )

        timeout_seconds = batch_timeout.total_seconds() if batch_timeout else 0.0

        if timeout_seconds < 0:
            raise ValueError(f"batch_timeout must be 0 or more; got {batch_timeout}")

        uploaded: Future[str] | None = None

        ## uploaded before taking its place in the topic's order, so uploads overlap
        if len(payload) > MAX_INLINE_PAYLOAD_BYTES:
            with self._lock:
                if self._upload_pool is None:
                    self._upload_pool = ThreadPoolExecutor(
                        thread_name_prefix="hatchet-stream-upload"
                    )
                pool = self._upload_pool
            uploaded = pool.submit(self._upload_payload, payload)

        publish = _QueuedPublish(
            payload=payload,
            send_by=time.monotonic() + timeout_seconds,
            uploaded=uploaded,
        )
        key = (namespace, topic)

        with self._lock:
            queue = self._queues.get(key)
            draining = queue is not None
            if queue is None:
                queue = _PublishQueue()
                self._queues[key] = queue
            queue.items.append(publish)
            ## a new publish may fill the request or have a shorter timeout
            queue.wake.set()

        if not draining:
            threading.Thread(
                target=self._drain_publishes,
                args=(key, queue),
                name="hatchet-stream-publish",
                daemon=True,
            ).start()

        return publish.result

    def _next_batch(
        self, key: tuple[str, str], queue: _PublishQueue
    ) -> list[_QueuedPublish] | None:
        """The next request's publishes, or None once the queue is empty and removed."""
        while True:
            with self._lock:
                if not queue.items:
                    ## removed under the lock publish takes, so a later publish starts a new drain
                    del self._queues[key]
                    return None

                count = _fitting_publishes(queue.items)
                full = count < len(queue.items) or count == MAX_PUBLISH_BATCH_ENTRIES
                wait = min(p.send_by for p in queue.items) - time.monotonic()

                if full or wait <= 0:
                    batch = queue.items[:count]
                    del queue.items[:count]
                    return batch

                queue.wake.clear()

            queue.wake.wait(wait)

    def _drain_publishes(self, key: tuple[str, str], queue: _PublishQueue) -> None:
        while (batch := self._next_batch(key, queue)) is not None:
            sent: list[_QueuedPublish] = []
            entries: list[streams_protos.StreamPublishEntry] = []

            for p in batch:
                if p.uploaded is None:
                    entries.append(streams_protos.StreamPublishEntry(payload=p.payload))
                    sent.append(p)
                    continue

                ## a failed upload fails only its own publish, as the rest never depended on it
                try:
                    ref = p.uploaded.result()
                except BaseException as e:
                    p.result.set_exception(e)
                    continue

                entries.append(streams_protos.StreamPublishEntry(payload_ref=ref))
                sent.append(p)

            if entries:
                self._send_publishes(key, sent, entries)

    def _send_publishes(
        self,
        key: tuple[str, str],
        sent: list[_QueuedPublish],
        entries: list[streams_protos.StreamPublishEntry],
    ) -> None:
        """Sends entries as one request, or one request per entry to an engine that predates PublishBatch, and settles each publish."""
        if not self._publish_batch_unsupported:
            try:
                self._publish_ordered(key, entries)
            except Exception as e:
                if _grpc_code(e) != grpc.StatusCode.UNIMPLEMENTED:
                    for p in sent:
                        p.result.set_exception(e)
                    return
                ## nothing was stored, so the batch goes out again below
                self._publish_batch_unsupported = True
            else:
                for p in sent:
                    p.result.set_result(None)
                return

        ## settled one by one: a failure partway leaves the publishes before it stored
        for p, entry in zip(sent, entries, strict=True):
            try:
                self._publish_ordered(key, [entry])
            except Exception as e:  # noqa: PERF203
                p.result.set_exception(e)
            else:
                p.result.set_result(None)

    def _publish_ordered(
        self,
        key: tuple[str, str],
        entries: list[streams_protos.StreamPublishEntry],
        retry_gap: bool = True,
    ) -> None:
        namespace, topic = key

        with self._lock:
            producer = self._producers.setdefault(key, _Producer())

        try:
            if self._publish_batch_unsupported and len(entries) == 1:
                self._get_or_create_client().Publish(
                    streams_protos.PublishStreamMessageRequest(
                        namespace=namespace,
                        topic=topic,
                        payload=entries[0].payload,
                        payload_ref=entries[0].payload_ref,
                        producer_id=producer.producer_id,
                        producer_seq=producer.seq,
                    ),
                    metadata=self._metadata(),
                )
            else:
                self._get_or_create_client().PublishBatch(
                    streams_protos.PublishStreamMessagesRequest(
                        namespace=namespace,
                        topic=topic,
                        producer_id=producer.producer_id,
                        first_producer_seq=producer.seq,
                        entries=entries,
                    ),
                    metadata=self._metadata(),
                )
        except Exception as e:
            code = _grpc_code(e)

            ## it may still land, and a different payload under its seq would be dropped as a duplicate
            if code not in _REJECTED_BEFORE_STORING:
                with self._lock:
                    self._producers[key] = _Producer()

            ## a gap stored nothing (e.g. the watermark passed cursor retention), so resend as the new producer
            if retry_gap and code == grpc.StatusCode.FAILED_PRECONDITION:
                return self._publish_ordered(key, entries, retry_gap=False)

            raise

        producer.seq += len(entries)
        return None

    def publish(
        self,
        topic: str,
        message: bytes | str,
        namespace: str = "",
        batch_timeout: timedelta | None = None,
    ) -> None:
        """
        Publish a message to a topic, returning once it is stored. A client's messages to a topic are delivered in the order `publish` was called. Publishes to a topic share requests: everything queued while one is in flight goes out in the next, and `batch_timeout` holds a message up to that long for more to join it.

        :param topic: The topic to publish to.
        :param message: The message payload. A string is encoded as UTF-8.
        :param namespace: The topic's namespace; the empty namespace by default.
        :param batch_timeout: How long the message may wait for later publishes to the same topic to share its request, bounding the latency batching adds. By default, it is sent as soon as no other request to the topic is in flight.

        :return: None
        """
        self._enqueue_publish(topic, message, namespace, batch_timeout).result()

    async def aio_publish(
        self,
        topic: str,
        message: bytes | str,
        namespace: str = "",
        batch_timeout: timedelta | None = None,
    ) -> None:
        """
        Publish a message to a topic, returning once it is stored. A client's messages to a topic are delivered in the order `aio_publish` was called. Publishes to a topic share requests: everything queued while one is in flight goes out in the next, and `batch_timeout` holds a message up to that long for more to join it.

        :param topic: The topic to publish to.
        :param message: The message payload. A string is encoded as UTF-8.
        :param namespace: The topic's namespace; the empty namespace by default.
        :param batch_timeout: How long the message may wait for later publishes to the same topic to share its request, bounding the latency batching adds. By default, it is sent as soon as no other request to the topic is in flight.

        :return: None
        """
        await asyncio.wrap_future(
            self._enqueue_publish(topic, message, namespace, batch_timeout)
        )

    def topic_metadata(self, topic: str, namespace: str = "") -> StreamTopicMetadata:
        """
        Describe a topic's retained messages. Raises a `NOT_FOUND` gRPC error for a topic nothing was published to.

        :param topic: The topic to describe.
        :param namespace: The topic's namespace; the empty namespace by default.

        :return: The topic's metadata.
        """
        res = cast(
            streams_protos.StreamTopicMetadata,
            self._get_or_create_client().GetTopicMetadata(
                streams_protos.GetStreamTopicMetadataRequest(
                    namespace=namespace, topic=topic
                ),
                metadata=self._metadata(),
            ),
        )

        return StreamTopicMetadata(
            namespace=res.namespace,
            topic=res.topic,
            tenant_id=res.tenant_id,
            message_count=res.message_count,
            latest_cursor=(
                res.latest_cursor if res.HasField("latest_cursor") else None
            ),
            last_published_at=(
                res.last_published_at.ToDatetime(tzinfo=timezone.utc)
                if res.HasField("last_published_at")
                else None
            ),
        )

    async def aio_topic_metadata(
        self, topic: str, namespace: str = ""
    ) -> StreamTopicMetadata:
        """
        Describe a topic's retained messages. Raises a `NOT_FOUND` gRPC error for a topic nothing was published to.

        :param topic: The topic to describe.
        :param namespace: The topic's namespace; the empty namespace by default.

        :return: The topic's metadata.
        """
        return await asyncio.to_thread(self.topic_metadata, topic, namespace)

    def reset(self, topic: str, namespace: str = "", cursor: str | None = None) -> None:
        """
        Make `events` calls without a cursor start after `cursor`, or the topic's latest message. Readers already iterating, and any call with a cursor, are unaffected.

        :param topic: The topic to reset.
        :param namespace: The topic's namespace; the empty namespace by default.
        :param cursor: A cursor read off one of the topic's events to reset to, instead of its latest message.

        :return: None
        """
        request = streams_protos.ResetStreamTopicRequest(
            namespace=namespace, topic=topic, cursor=cursor
        )

        self._get_or_create_client().Reset(request, metadata=self._metadata())

    async def aio_reset(
        self, topic: str, namespace: str = "", cursor: str | None = None
    ) -> None:
        """
        Make `events` calls without a cursor start after `cursor`, or the topic's latest message. Readers already iterating, and any call with a cursor, are unaffected.

        :param topic: The topic to reset.
        :param namespace: The topic's namespace; the empty namespace by default.
        :param cursor: A cursor read off one of the topic's events to reset to, instead of its latest message.

        :return: None
        """
        await asyncio.to_thread(self.reset, topic, namespace, cursor)

    def events(
        self, topic: str, namespace: str = "", cursor: str | None = None
    ) -> Generator[StreamEvent, None, None]:
        """
        Yield a topic's messages after `cursor`, or from the oldest retained (or, once the topic is reset, after the reset). If the connection drops, it resubscribes from the last message yielded, so nothing is skipped or repeated. Ends when the server hangs up an idle subscription or the caller stops iterating; raises on errors a retry can't fix, such as an expired cursor (`OUT_OF_RANGE`) or a tenant without durable streams (`PERMISSION_DENIED`). Runs `aio_events` on an event loop of its own.

        :param topic: The topic to read from.
        :param namespace: The topic's namespace; the empty namespace by default.
        :param cursor: A cursor read off a previous event, to resume right after it.

        :return: An iterator of the topic's events.
        """
        return _iterate_in_thread(self.aio_events(topic, namespace, cursor))

    async def aio_events(
        self, topic: str, namespace: str = "", cursor: str | None = None
    ) -> AsyncGenerator[StreamEvent, None]:
        """
        Yield a topic's messages after `cursor`, or from the oldest retained (or, once the topic is reset, after the reset). If the connection drops, it resubscribes from the last message yielded, so nothing is skipped or repeated. Ends when the server hangs up an idle subscription or the caller stops iterating; raises on errors a retry can't fix, such as an expired cursor.

        :param topic: The topic to read from.
        :param namespace: The topic's namespace; the empty namespace by default.
        :param cursor: A cursor read off a previous event, to resume right after it.

        :return: An async iterator of the topic's events.

        :raises Exception: An error a resubscribe can't fix, such as an expired cursor (`OUT_OF_RANGE`) or a tenant without durable streams (`PERMISSION_DENIED`).
        """
        failures = 0
        attempt = 0

        async with self._open_aio_client() as stub:
            while True:
                call = cast(
                    grpc.aio.UnaryStreamCall[
                        streams_protos.SubscribeStreamRequest,
                        streams_protos.StreamMessage,
                    ],
                    stub.Subscribe(
                        _subscribe_request(namespace, topic, cursor),
                        metadata=self._metadata(),
                    ),
                )

                try:
                    async for msg in call:
                        if msg.hangup:
                            return

                        for entry in msg.entries:
                            payload = (
                                await asyncio.to_thread(
                                    self._fetch_payload, entry.payload_ref
                                )
                                if entry.payload_ref
                                else entry.payload
                            )
                            cursor = entry.cursor
                            # reset only on delivery, so a payload fetch failing on every reconnect still backs off
                            failures = 0
                            yield _to_event(entry, payload)
                    # ending without a hangup means the engine went away, e.g. it is shutting down
                except Exception as e:
                    if not _should_resubscribe(e, attempt):
                        raise
                finally:
                    call.cancel()

                await asyncio.sleep(self._resubscribe_delay(failures))
                failures += 1
                attempt += 1
