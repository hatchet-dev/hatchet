from unittest.mock import MagicMock

from hatchet_sdk.clients.admin import MAX_BULK_WORKFLOW_RUN_BATCH_SIZE, AdminClient
from hatchet_sdk.contracts.v1.shared import trigger_pb2 as trigger_protos


def make_client(grpc_max_send_message_length: int = 4 * 1024 * 1024) -> AdminClient:
    config = MagicMock()
    config.grpc_max_send_message_length = grpc_max_send_message_length

    return AdminClient(
        config=config,
        workflow_run_listener=MagicMock(),
        workflow_run_event_listener=MagicMock(),
    )


def make_request(
    name: str = "wf", input_size_bytes: int = 0
) -> trigger_protos.TriggerWorkflowRequest:
    return trigger_protos.TriggerWorkflowRequest(
        name=name,
        input="x" * input_size_bytes,
    )


def test_empty_input_yields_nothing() -> None:
    client = make_client()
    chunks = list(client.chunk_workflow_runs([]))

    assert chunks == []


def test_single_item_yields_one_chunk() -> None:
    client = make_client()
    reqs = [make_request()]
    chunks = list(client.chunk_workflow_runs(reqs))

    assert len(chunks) == 1
    assert len(chunks[0]) == 1


def test_items_within_size_and_count_limits_yield_one_chunk() -> None:
    client = make_client()
    reqs = [make_request() for _ in range(10)]
    chunks = list(client.chunk_workflow_runs(reqs))

    assert len(chunks) == 1
    assert sum(len(c) for c in chunks) == 10


def test_splits_on_count_limit() -> None:
    client = make_client()
    reqs = [make_request() for _ in range(MAX_BULK_WORKFLOW_RUN_BATCH_SIZE + 1)]
    chunks = list(client.chunk_workflow_runs(reqs))

    assert len(chunks) == 2
    assert len(chunks[0]) == MAX_BULK_WORKFLOW_RUN_BATCH_SIZE
    assert len(chunks[1]) == 1


def test_splits_on_byte_size_limit() -> None:
    max_bytes = 1000
    client = make_client(grpc_max_send_message_length=max_bytes)

    req = make_request(input_size_bytes=600)
    single_size = req.ByteSize()

    assert (
        single_size < max_bytes
    ), "single request must fit within the limit for this test to be valid"

    reqs = [make_request(input_size_bytes=600) for _ in range(3)]
    chunks = list(client.chunk_workflow_runs(reqs))

    assert len(chunks) > 1
    assert sum(len(c) for c in chunks) == 3


def test_all_items_present_across_chunks() -> None:
    client = make_client()
    reqs = [
        make_request(name=f"wf-{i}")
        for i in range(MAX_BULK_WORKFLOW_RUN_BATCH_SIZE * 3 + 7)
    ]
    chunks = list(client.chunk_workflow_runs(reqs))

    assert sum(len(c) for c in chunks) == len(reqs)


def test_no_chunk_exceeds_byte_limit() -> None:
    max_bytes = 500
    client = make_client(grpc_max_send_message_length=max_bytes)
    reqs = [make_request(input_size_bytes=100) for _ in range(20)]
    chunks = list(client.chunk_workflow_runs(reqs))

    for chunk in chunks:
        total = sum(r.ByteSize() for r in chunk)
        assert total <= max_bytes


def test_no_chunk_exceeds_count_limit() -> None:
    client = make_client()
    reqs = [make_request() for _ in range(MAX_BULK_WORKFLOW_RUN_BATCH_SIZE * 5)]
    chunks = list(client.chunk_workflow_runs(reqs))

    for chunk in chunks:
        assert len(chunk) <= MAX_BULK_WORKFLOW_RUN_BATCH_SIZE
