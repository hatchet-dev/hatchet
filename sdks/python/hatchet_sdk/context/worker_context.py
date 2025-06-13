from hatchet_sdk.clients.dispatcher.dispatcher import DispatcherClient


class WorkerContext:
    def __init__(self, labels: dict[str, str | int], client: DispatcherClient):
        self._worker_id: str | None = None
        self._registered_workflow_names: list[str] = []
        self._labels: dict[str, str | int] = {}

        self._labels = labels
        self.client = client

    def labels(self) -> dict[str, str | int]:
        return self._labels

    def upsert_labels(self, labels: dict[str, str | int]) -> None:
        self.client.upsert_worker_labels(self._worker_id, labels)
        self._labels.update(labels)

    async def async_upsert_labels(self, labels: dict[str, str | int]) -> None:
        await self.client.async_upsert_worker_labels(self._worker_id, labels)
        self._labels.update(labels)

    def id(self) -> str | None:
        return self._worker_id
