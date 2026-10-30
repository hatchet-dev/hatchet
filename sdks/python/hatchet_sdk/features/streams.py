import asyncio
import random
import threading
import time
import uuid
from collections.abc import AsyncGenerator, AsyncIterator, Callable
from concurrent.futures import ThreadPoolExecutor
from contextlib import asynccontextmanager, suppress
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
    """Resumes `subscribe` right after this event."""

    created_at: datetime | None = None
    """When the message was stored."""


class StreamTopicMetadata(BaseModel):
    """What a durable stream topic retains; counts and the latest message cover the tenant's retention."""

    namespace: str
    topic: str
    tenant_id: str
    message_count: int

    latest_cursor: str | None = None
    """Resumes `subscribe` after the newest retained message; None when none is retained."""

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
    uploaded: "asyncio.Future[str] | None"
    result: "asyncio.Future[None]" = field(
        default_factory=lambda: asyncio.get_running_loop().create_future()
    )

    @property
    def request_bytes(self) -> int:
        return PAYLOAD_REF_BYTES if self.uploaded else len(self.payload)

    def settle(self, err: BaseException | None = None) -> None:
        """Skips a publish whose caller stopped waiting, which cancelled its result."""
        if self.result.done():
            return

        if err is None:
            self.result.set_result(None)
        elif isinstance(err, asyncio.CancelledError):
            self.result.cancel()
        else:
            self.result.set_exception(err)

    def drop_upload(self) -> None:
        if self.uploaded is None:
            return

        ## a finished upload's error is retrieved so asyncio doesn't log it as unhandled
        if self.uploaded.done():
            self.uploaded.exception()
        else:
            self.uploaded.cancel()


@dataclass
class _PublishQueue:
    """One topic's publishes on one event loop; one request is in flight at a time so two never share a seq."""

    client: "StreamsClient"
    stub: V1StreamsStub
    namespace: str
    topic: str

    items: list[_QueuedPublish] = field(default_factory=list)
    wake: asyncio.Event = field(default_factory=asyncio.Event)

    ## seq only advances after a publish succeeds
    producer: _Producer = field(default_factory=_Producer)

    ## held here because asyncio keeps only weak references to tasks
    drain: "asyncio.Task[None] | None" = None

    async def publish(self, publish: _QueuedPublish) -> None:
        self.items.append(publish)
        ## a new publish may fill the request or have a shorter timeout
        self.wake.set()

        if self.drain is None:
            self.drain = asyncio.create_task(self._drain())

        await publish.result

    def _fitting_publishes(self) -> int:
        """Always at least one, so a max-size payload still goes out on its own."""
        total = 0
        count = 0

        while count < len(self.items) and count < MAX_PUBLISH_BATCH_ENTRIES:
            nxt = self.items[count].request_bytes + PUBLISH_ENTRY_OVERHEAD_BYTES
            if count > 0 and total + nxt > MAX_PUBLISH_BATCH_BYTES:
                break
            total += nxt
            count += 1

        return count

    async def _next_batch(self) -> list[_QueuedPublish] | None:
        """The next request's publishes, or None once the queue is empty."""
        while True:
            ## a publish cancelled before its request goes out is dropped, so it never takes a seq
            for p in self.items:
                if p.result.cancelled():
                    p.drop_upload()
            self.items = [p for p in self.items if not p.result.cancelled()]

            if not self.items:
                return None

            count = self._fitting_publishes()
            full = count < len(self.items) or count == MAX_PUBLISH_BATCH_ENTRIES
            wait = min(p.send_by for p in self.items) - time.monotonic()

            if full or wait <= 0:
                batch = self.items[:count]
                del self.items[:count]
                return batch

            self.wake.clear()

            with suppress(asyncio.TimeoutError):
                await asyncio.wait_for(self.wake.wait(), wait)

    async def _drain(self) -> None:
        batch: list[_QueuedPublish] | None = None

        try:
            while (batch := await self._next_batch()) is not None:
                await self._send_batch(batch)
        except BaseException as e:
            ## e.g. cancelled as its loop shuts down; no caller is left waiting on a publish it held
            for p in [*(batch or []), *self.items]:
                p.settle(e)
            self.items.clear()
            ## a request cut off mid-flight may still land under the current seq
            self.producer = _Producer()
            raise
        finally:
            ## cleared with no await since the last check, so a later publish starts a new drain
            self.drain = None

    async def _send_batch(self, batch: list[_QueuedPublish]) -> None:
        sent: list[_QueuedPublish] = []
        entries: list[streams_protos.StreamPublishEntry] = []

        for p in batch:
            if p.uploaded is None:
                entries.append(streams_protos.StreamPublishEntry(payload=p.payload))
                sent.append(p)
                continue

            ## a failed upload fails only its own publish, as the rest never depended on it
            try:
                ref = await p.uploaded
            except Exception as e:
                p.settle(e)
                continue

            entries.append(streams_protos.StreamPublishEntry(payload_ref=ref))
            sent.append(p)

        if entries:
            await self._send_entries(sent, entries)

    async def _send_entries(
        self,
        sent: list[_QueuedPublish],
        entries: list[streams_protos.StreamPublishEntry],
    ) -> None:
        """Sends entries as one request, or one request per entry to an engine that predates PublishBatch, and settles each publish."""
        if not self.client._publish_batch_unsupported:
            try:
                await self._publish_ordered(entries)
            except Exception as e:
                if _grpc_code(e) != grpc.StatusCode.UNIMPLEMENTED:
                    for p in sent:
                        p.settle(e)
                    return
                ## nothing was stored, so the batch goes out again below
                self.client._publish_batch_unsupported = True
            else:
                for p in sent:
                    p.settle()
                return

        ## settled one by one: a failure partway leaves the publishes before it stored
        for p, entry in zip(sent, entries, strict=True):
            try:
                await self._publish_ordered([entry])
            except Exception as e:  # noqa: PERF203
                p.settle(e)
            else:
                p.settle()

    async def _publish_ordered(
        self,
        entries: list[streams_protos.StreamPublishEntry],
        retry_gap: bool = True,
    ) -> None:
        producer = self.producer

        try:
            if self.client._publish_batch_unsupported and len(entries) == 1:
                await self.stub.Publish(  # type: ignore[misc]
                    streams_protos.PublishStreamMessageRequest(
                        namespace=self.namespace,
                        topic=self.topic,
                        payload=entries[0].payload,
                        payload_ref=entries[0].payload_ref,
                        producer_id=producer.producer_id,
                        producer_seq=producer.seq,
                    ),
                    metadata=self.client._metadata(),
                )
            else:
                await self.stub.PublishBatch(  # type: ignore[misc]
                    streams_protos.PublishStreamMessagesRequest(
                        namespace=self.namespace,
                        topic=self.topic,
                        producer_id=producer.producer_id,
                        first_producer_seq=producer.seq,
                        entries=entries,
                    ),
                    metadata=self.client._metadata(),
                )
        except Exception as e:
            code = _grpc_code(e)

            ## it may still land, and a different payload under its seq would be dropped as a duplicate
            if code not in _REJECTED_BEFORE_STORING:
                self.producer = _Producer()

            ## a gap stored nothing (e.g. the watermark passed cursor retention), so resend as the new producer
            if retry_gap and code == grpc.StatusCode.FAILED_PRECONDITION:
                return await self._publish_ordered(entries, retry_gap=False)

            raise

        producer.seq += len(entries)
        return None


@dataclass
class _LoopPublisher:
    """Publish state for one event loop, as an aio channel and asyncio primitives only work on the loop that made them."""

    channel: grpc.aio.Channel
    stub: V1StreamsStub

    ## per (namespace, topic)
    queues: dict[tuple[str, str], _PublishQueue] = field(default_factory=dict)

    ## held here because a loop keeps only weak references to its async generators
    shutdown_hook: AsyncGenerator[None, None] | None = None


def _finish_shutdown_hook(publisher: _LoopPublisher) -> None:
    """Runs a closed loop's hook by hand; left unfinished, asyncio would schedule its cleanup on that closed loop when it is collected."""
    if publisher.shutdown_hook is None:
        return

    with suppress(StopIteration):
        publisher.shutdown_hook.aclose().send(None)


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
        self._publishers: dict[asyncio.AbstractEventLoop, _LoopPublisher] = {}

        ## set once the engine answers PublishBatch as unimplemented: it predates batching
        self._publish_batch_unsupported = False

    def _get_or_create_client(self) -> V1StreamsStub:
        with self._lock:
            if self._stub is None:
                self._stub = V1StreamsStub(new_conn(self.client_config, False))
            return self._stub

    async def _loop_publisher(self) -> _LoopPublisher:
        loop = asyncio.get_running_loop()

        with self._lock:
            ## a loop closed without shutting down its async generators never ran its hook
            abandoned = [
                self._publishers.pop(lp)
                for lp in list(self._publishers)
                if lp.is_closed()
            ]

            publisher = self._publishers.get(loop)
            created = publisher is None
            if publisher is None:
                channel = new_conn(self.client_config, True)
                publisher = _LoopPublisher(channel=channel, stub=V1StreamsStub(channel))
                self._publishers[loop] = publisher

        for closed in abandoned:
            _finish_shutdown_hook(closed)

        if created:
            publisher.shutdown_hook = self._close_at_shutdown(loop, publisher)
            ## the first iteration registers it with the loop, and reaches the yield without suspending
            await anext(publisher.shutdown_hook)

        return publisher

    async def _close_at_shutdown(
        self, loop: asyncio.AbstractEventLoop, publisher: _LoopPublisher
    ) -> AsyncGenerator[None, None]:
        """A loop's shutdown hook: `asyncio.run` and other runners close every async generator before closing the loop, after cancelling its tasks."""
        try:
            yield
        finally:
            with self._lock:
                if self._publishers.get(loop) is publisher:
                    del self._publishers[loop]

            ## finished by hand once the loop is closed, when nothing can be awaited
            if not loop.is_closed():
                await publisher.channel.close()

    ## its own pool, so large uploads don't hold the default executor's threads that `to_thread` callers share
    def _upload_executor(self) -> ThreadPoolExecutor:
        with self._lock:
            if self._upload_pool is None:
                self._upload_pool = ThreadPoolExecutor(
                    thread_name_prefix="hatchet-stream-upload"
                )
            return self._upload_pool

    ## an aio channel is bound to its event loop, so each read opens its own
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

    async def publish(
        self,
        topic: str,
        message: bytes | str,
        namespace: str = "",
        batch_timeout: timedelta | None = None,
    ) -> None:
        """
        Publish a message to a topic, returning once it is stored. A client's messages to a topic are delivered in the order `publish` was called. Publishes to a topic share requests: everything queued while one is in flight goes out in the next, and `batch_timeout` holds a message up to that long for more to join it. Cancelling a publish before its request goes out drops the message; once the request is sent, it may still be stored.

        :param topic: The topic to publish to.
        :param message: The message payload. A string is encoded as UTF-8.
        :param namespace: The topic's namespace; the empty namespace by default.
        :param batch_timeout: How long the message may wait for later publishes to the same topic to share its request, bounding the latency batching adds. By default, it is sent as soon as no other request to the topic is in flight.

        :return: None

        :raises ValueError: The payload is empty or over 64 MiB, or `batch_timeout` is negative.
        """
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

        publisher = await self._loop_publisher()

        ## uploaded before taking its place in the topic's order, so uploads overlap
        uploaded = (
            asyncio.get_running_loop().run_in_executor(
                self._upload_executor(), self._upload_payload, payload
            )
            if len(payload) > MAX_INLINE_PAYLOAD_BYTES
            else None
        )

        queue = publisher.queues.get((namespace, topic))
        if queue is None:
            queue = _PublishQueue(self, publisher.stub, namespace, topic)
            publisher.queues[(namespace, topic)] = queue

        await queue.publish(
            _QueuedPublish(
                payload=payload,
                send_by=time.monotonic() + timeout_seconds,
                uploaded=uploaded,
            )
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

    async def aio_reset(
        self, topic: str, namespace: str = "", cursor: str | None = None
    ) -> None:
        """
        Make `subscribe` calls without a cursor start after `cursor`, or the topic's latest message. Readers already iterating, and any call with a cursor, are unaffected.

        :param topic: The topic to reset.
        :param namespace: The topic's namespace; the empty namespace by default.
        :param cursor: A cursor read off one of the topic's events to reset to, instead of its latest message.

        :return: None
        """
        request = streams_protos.ResetStreamTopicRequest(
            namespace=namespace, topic=topic, cursor=cursor
        )

        async with self._open_aio_client() as stub:
            await stub.Reset(request, metadata=self._metadata())  # type: ignore[misc]

    async def subscribe(
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
