import asyncio
from warnings import warn

from hatchet_sdk.clients.dispatcher.dispatcher import DispatcherClient
from hatchet_sdk.types.labels import WorkerLabel


class WorkerContext:
    def __init__(self, labels: list[WorkerLabel], client: DispatcherClient):
        self._worker_id: str | None = None
        self._labels = {
            label.key: label.value for label in labels if label.key is not None
        }
        self._client = client

    @property
    def client(self) -> DispatcherClient:
        warn(
            "The client property is internal and should not be used directly. It will be removed in v2.0.0.",
            DeprecationWarning,
            stacklevel=2,
        )
        return self._client

    def labels(self) -> dict[str, str | int]:
        return self._labels

    def upsert_labels(self, labels: dict[str, str | int]) -> None:
        warn(
            "The upsert_labels method deprecated. It will be removed in v2.0.0. Use `Context.upsert_labels` instead.",
            DeprecationWarning,
            stacklevel=2,
        )

        self._client.upsert_worker_labels(
            self._worker_id,
            [WorkerLabel(key=key, value=value) for key, value in labels.items()],
        )
        self._labels.update(labels)

    async def async_upsert_labels(self, labels: dict[str, str | int]) -> None:
        await asyncio.to_thread(self.upsert_labels, labels)

    def id(self) -> str | None:
        return self._worker_id
