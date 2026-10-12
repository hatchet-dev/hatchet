import json
from datetime import UTC, datetime

from google.protobuf import timestamp_pb2

from hatchet_sdk import Event
from hatchet_sdk.clients.rest.models.api_resource_meta import APIResourceMeta
from hatchet_sdk.clients.rest.models.v1_event import V1Event
from hatchet_sdk.clients.rest.models.v1_event_workflow_run_summary import (
    V1EventWorkflowRunSummary,
)
from hatchet_sdk.contracts.events_pb2 import Event as EventProto

SEEN_AT = datetime(2026, 1, 2, 3, 4, 5, tzinfo=UTC)
PAYLOAD = {"order": "1"}
ADDITIONAL_METADATA = {"source": "test"}


def test_pushed_event_has_parsed_payload_and_metadata() -> None:
    event = Event.from_proto(
        EventProto(
            tenant_id="tenant",
            event_id="event",
            key="user:create",
            payload=json.dumps(PAYLOAD),
            event_timestamp=timestamp_pb2.Timestamp(seconds=int(SEEN_AT.timestamp())),
            additional_metadata=json.dumps(ADDITIONAL_METADATA),
            scope="user:1",
        )
    )

    assert event.event_id == "event"
    assert event.payload == PAYLOAD
    assert event.additional_metadata == ADDITIONAL_METADATA
    assert event.scope == "user:1"
    assert event.seen_at == SEEN_AT
    assert event.triggered_runs is None
    assert event.workflow_run_summary is None


def test_pushed_event_without_metadata_has_empty_metadata() -> None:
    event = Event.from_proto(
        EventProto(tenant_id="tenant", event_id="event", key="user:create")
    )

    assert event.payload == {}
    assert event.additional_metadata == {}
    assert event.scope is None


def test_event_read_from_the_api_matches_the_pushed_shape() -> None:
    summary = V1EventWorkflowRunSummary(
        running=1, queued=0, succeeded=2, failed=0, cancelled=0
    )

    event = Event.from_rest_model(
        V1Event(
            metadata=APIResourceMeta(id="event", createdAt=SEEN_AT, updatedAt=SEEN_AT),
            key="user:create",
            tenantId="tenant",
            workflowRunSummary=summary,
            additionalMetadata=ADDITIONAL_METADATA,
            payload=PAYLOAD,
            seenAt=SEEN_AT,
            triggeredRuns=[],
        )
    )

    assert event.event_id == "event"
    assert event.tenant_id == "tenant"
    assert event.payload == PAYLOAD
    assert event.additional_metadata == ADDITIONAL_METADATA
    assert event.seen_at == SEEN_AT
    assert event.workflow_run_summary == summary
    assert event.triggered_runs == []
    assert event.payloads_restricted is False
