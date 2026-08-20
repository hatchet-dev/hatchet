import asyncio
from collections.abc import Generator
from datetime import datetime, timezone
from subprocess import Popen
from uuid import uuid4

import pytest

from examples.migration_guides.temporal import (
    ApprovalInput,
    ApprovalOutput,
    ChargeOutput,
    EmailOutput,
    FanOutInput,
    FulfillOutput,
    ModelOutput,
    OnboardingInput,
    OrderInput,
    PromptInput,
    ReportInput,
    ReportOutput,
    SyncInput,
    approval_flow,
    call_model,
    charge_order,
    charge_order_with_logging,
    charge_order_with_retries,
    create_schedules,
    grant_approval,
    onboarding_flow,
    order_workflow,
    process_items,
    process_order,
    send_followup_email,
    send_welcome_email,
    sync_customer,
    trigger_process_order,
    wait_a_day,
    weekly_report,
)
from examples.test_utils import wait_for_running_status
from hatchet_sdk import Hatchet
from tests.worker_fixture import get_free_port, hatchet_worker


@pytest.fixture(scope="module")
def temporal_guide_worker() -> Generator[Popen[bytes], None, None]:
    with hatchet_worker(
        ["poetry", "run", "python", "-m", "examples.migration_guides.temporal"],
        get_free_port(),
    ) as proc:
        yield proc


requires_worker = pytest.mark.usefixtures("temporal_guide_worker")


@requires_worker
@pytest.mark.asyncio(loop_scope="session")
async def test_charge_order_task() -> None:
    result = await charge_order.aio_run(OrderInput(order_id="123"))

    assert result == ChargeOutput(charged=True, charge_id="ch_123")


@requires_worker
@pytest.mark.asyncio(loop_scope="session")
async def test_process_order_durable_task() -> None:
    result = await process_order.aio_run(OrderInput(order_id="123"))

    assert result == FulfillOutput(order_id="123", tracking_number="TRACK-123")


@requires_worker
@pytest.mark.asyncio(loop_scope="session")
async def test_trigger_process_order() -> None:
    result = await trigger_process_order("456")

    assert result == FulfillOutput(order_id="456", tracking_number="TRACK-456")


@requires_worker
@pytest.mark.asyncio(loop_scope="session")
async def test_charge_order_with_retries() -> None:
    # The task raises on its first two attempts, so a successful result is only
    # reachable if Hatchet retried it.
    result = await charge_order_with_retries.aio_run(OrderInput(order_id="789"))

    assert result == ChargeOutput(charged=True, charge_id="ch_789")


def test_retry_and_timeout_settings_reach_the_engine() -> None:
    (task,) = charge_order_with_retries.to_proto().tasks

    assert task.retries == 10
    assert task.backoff_factor == 2.0
    assert task.backoff_max_seconds == 10
    assert task.timeout == "30s"
    assert task.schedule_timeout == "10m"


@requires_worker
@pytest.mark.asyncio(loop_scope="session")
async def test_durable_event_wait(hatchet: Hatchet) -> None:
    correlation_id = str(uuid4())

    ref = await approval_flow.aio_run(
        ApprovalInput(order_id="123", correlation_id=correlation_id),
        wait_for_result=False,
    )

    # The durable wait has to be established before the event is pushed, otherwise
    # there is nothing for the CEL filter to match against.
    await wait_for_running_status(hatchet, ref.workflow_run_id)
    await asyncio.sleep(2)

    await grant_approval(correlation_id)

    assert await ref.aio_result() == ApprovalOutput(approved=True)


@requires_worker
@pytest.mark.asyncio(loop_scope="session")
async def test_fan_out_children() -> None:
    result = await process_items.aio_run(FanOutInput(item_ids=["a", "b", "c"]))

    assert [item.item_id for item in result.results] == ["a", "b", "c"]
    assert [item.result for item in result.results] == [
        "processed-a",
        "processed-b",
        "processed-c",
    ]


@requires_worker
@pytest.mark.asyncio(loop_scope="session")
async def test_dag_workflow() -> None:
    result = await order_workflow.aio_run(OrderInput(order_id="123"))

    assert result["validate"]["valid"] is True
    assert result["charge"]["charge_id"] == "ch_123"

    # `fulfill` builds its tracking number out of `charge`'s output, which it reads
    # off the context rather than through a local variable.
    assert result["fulfill"]["tracking_number"] == "TRACK-ch_123"


def test_dag_workflow_declares_a_chain() -> None:
    assert [
        (task.readable_id, list(task.parents))
        for task in order_workflow.to_proto().tasks
    ] == [
        ("validate", []),
        ("charge", ["validate"]),
        ("fulfill", ["charge"]),
    ]


@requires_worker
@pytest.mark.asyncio(loop_scope="session")
async def test_cron_declared_task_is_runnable() -> None:
    result = await weekly_report.aio_run(ReportInput(kind="weekly"))

    assert result == ReportOutput(kind="weekly", rows=7)


def test_cron_declaration_reaches_the_engine() -> None:
    assert weekly_report.to_proto().cron_triggers == ["0 9 * * 1"]


@requires_worker
@pytest.mark.asyncio(loop_scope="session")
async def test_runtime_schedules(hatchet: Hatchet) -> None:
    customer_id = f"acme-{uuid4().hex[:8]}"

    cron_id, scheduled_id = await create_schedules(customer_id)

    cron = await hatchet.cron.aio_get(cron_id)
    assert cron.cron == "0 9 * * 1"
    assert cron.name == f"weekly-report-{customer_id}"

    scheduled = await hatchet.scheduled.aio_get(scheduled_id)
    assert scheduled.trigger_at > datetime.now(tz=timezone.utc)

    await hatchet.cron.aio_delete(cron_id)
    await hatchet.scheduled.aio_delete(scheduled_id)


@requires_worker
@pytest.mark.asyncio(loop_scope="session")
async def test_concurrency_and_rate_limits() -> None:
    sync_result = await sync_customer.aio_run(SyncInput(customer_id="acme"))
    assert sync_result["sync"]["records_synced"] == 3

    model_result = await call_model.aio_run(PromptInput(prompt="hello"))
    assert model_result == ModelOutput(completion="echo: hello")


def test_concurrency_and_rate_limit_settings_reach_the_engine() -> None:
    concurrency = sync_customer.to_proto().concurrency

    assert concurrency.expression == "input.customer_id"
    assert concurrency.max_runs == 1

    (task,) = call_model.to_proto().tasks
    (rate_limit,) = task.rate_limits

    assert rate_limit.key == "openai"
    assert rate_limit.units == 1


@requires_worker
@pytest.mark.asyncio(loop_scope="session")
async def test_context_logging() -> None:
    # The log line itself lands in Hatchet's log sink, which is not asserted here.
    result = await charge_order_with_logging.aio_run(OrderInput(order_id="123"))

    assert result == ChargeOutput(charged=True, charge_id="ch_123")


@requires_worker
@pytest.mark.asyncio(loop_scope="session")
async def test_onboarding_emails() -> None:
    # The three-day sleep between these two tasks cannot be exercised in CI, so the
    # tasks on either side of it are covered here and the flow itself is checked for
    # shape in `test_durable_sleeps_are_covered_by_their_execution_timeouts`.
    user = OnboardingInput(user_id="user-1")

    assert await send_welcome_email.aio_run(user) == EmailOutput(delivered=True)
    assert await send_followup_email.aio_run(user) == EmailOutput(delivered=True)


def test_durable_sleeps_are_covered_by_their_execution_timeouts() -> None:
    # Running either of these to completion would take days. A durable task's
    # execution timeout has to outlast its sleeps, which is what is checked here.
    (onboarding,) = onboarding_flow.to_proto().tasks
    assert onboarding.timeout == "168h"

    (daily,) = wait_a_day.to_proto().tasks
    assert daily.timeout == "48h"
