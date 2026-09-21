import multiprocessing

from hatchet_sdk.worker.slot_usage import (
    SharedSlotUsageByPool,
    add_slot_requests,
    create_shared_slot_usage_by_pool,
    publish_slot_usage,
    read_slot_usage,
    subtract_slot_requests,
)

_CTX = multiprocessing.get_context("spawn")


def _publish_from_child_process(
    shared_slot_usage_by_pool: SharedSlotUsageByPool,
) -> None:
    publish_slot_usage(shared_slot_usage_by_pool, {"default": 3, "gpu": 2})


def test_add_slot_requests_adds_units_to_every_requested_pool() -> None:
    assert add_slot_requests(
        {"default": 2, "durable": 1}, {"default": 1, "gpu": 2}
    ) == {"default": 3, "durable": 1, "gpu": 2}


def test_add_slot_requests_does_not_mutate_its_input() -> None:
    used_slots_by_pool = {"default": 2}

    add_slot_requests(used_slots_by_pool, {"default": 1})

    assert used_slots_by_pool == {"default": 2}


def test_subtract_slot_requests_undoes_add_slot_requests() -> None:
    used_slots_by_pool = {"default": 2, "durable": 1}
    slot_requests = {"default": 1, "gpu": 2}

    assert subtract_slot_requests(
        add_slot_requests(used_slots_by_pool, slot_requests), slot_requests
    ) == {"default": 2, "durable": 1, "gpu": 0}


def test_publish_resets_pools_with_no_running_tasks_to_zero() -> None:
    shared = create_shared_slot_usage_by_pool(_CTX, {"default": 10, "gpu": 4})

    publish_slot_usage(shared, {"default": 3, "gpu": 2})
    publish_slot_usage(shared, {"default": 1})

    assert read_slot_usage(shared) == {"default": 1, "gpu": 0}


def test_publish_ignores_pools_missing_from_the_worker_slot_config() -> None:
    shared = create_shared_slot_usage_by_pool(_CTX, {"default": 10})

    publish_slot_usage(shared, {"default": 1, "unknown": 5})

    assert read_slot_usage(shared) == {"default": 1}


def test_slot_usage_published_in_another_process_is_visible_to_the_reader() -> None:
    shared = create_shared_slot_usage_by_pool(_CTX, {"default": 10, "gpu": 4})

    process = _CTX.Process(target=_publish_from_child_process, args=(shared,))
    process.start()
    process.join(timeout=30)

    assert read_slot_usage(shared) == {"default": 3, "gpu": 2}
