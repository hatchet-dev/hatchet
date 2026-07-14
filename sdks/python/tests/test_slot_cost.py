"""Unit tests for the public slot_cost task parameter."""

import pytest

from hatchet_sdk import Context, DurableContext, EmptyModel, Hatchet


def dummy(input: EmptyModel, ctx: Context) -> None:
    return None


async def dummy_durable(input: EmptyModel, ctx: DurableContext) -> None:
    return None


def test_slot_cost_maps_to_default_pool(hatchet: Hatchet) -> None:
    wf = hatchet.workflow(name="slot-cost-wf")

    plain = wf.task(name="plain")(dummy)
    heavy = wf.task(name="heavy", slot_cost=5)(dummy)

    assert plain.to_proto("svc").slot_requests == {"default": 1}
    assert heavy.to_proto("svc").slot_requests == {"default": 5}


def test_standalone_task_slot_cost(hatchet: Hatchet) -> None:
    standalone = hatchet.task(name="standalone-heavy", slot_cost=5)(dummy)

    assert standalone._task.to_proto("svc").slot_requests == {"default": 5}


@pytest.mark.parametrize("bad", [0, -1])
def test_non_positive_slot_cost_is_rejected(hatchet: Hatchet, bad: int) -> None:
    wf = hatchet.workflow(name="slot-cost-invalid-wf")

    with pytest.raises(ValueError):
        wf.task(name="bad", slot_cost=bad)(dummy)


def test_durable_task_keeps_durable_slot(hatchet: Hatchet) -> None:
    wf = hatchet.workflow(name="slot-cost-durable-wf")
    t = wf.durable_task()(dummy_durable)

    assert t.to_proto("svc").slot_requests == {"durable": 1}
