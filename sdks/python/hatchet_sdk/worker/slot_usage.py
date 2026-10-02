from ctypes import c_int
from multiprocessing.context import BaseContext

SharedSlotUsageByPool = dict[str, c_int]


def create_shared_slot_usage_by_pool(
    ctx: BaseContext, slot_config: dict[str, int]
) -> SharedSlotUsageByPool:
    return {pool: ctx.RawValue(c_int, 0) for pool in slot_config}


def add_slot_requests(
    used_slots_by_pool: dict[str, int], slot_requests: dict[str, int]
) -> dict[str, int]:
    return {
        **used_slots_by_pool,
        **{
            pool: used_slots_by_pool.get(pool, 0) + units
            for pool, units in slot_requests.items()
        },
    }


def subtract_slot_requests(
    used_slots_by_pool: dict[str, int], slot_requests: dict[str, int]
) -> dict[str, int]:
    return add_slot_requests(
        used_slots_by_pool,
        {pool: -units for pool, units in slot_requests.items()},
    )


def publish_slot_usage(
    shared_slot_usage_by_pool: SharedSlotUsageByPool,
    used_slots_by_pool: dict[str, int],
) -> None:
    for pool, shared_used_slots in shared_slot_usage_by_pool.items():
        shared_used_slots.value = used_slots_by_pool.get(pool, 0)


def read_slot_usage(
    shared_slot_usage_by_pool: SharedSlotUsageByPool,
) -> dict[str, int]:
    return {
        pool: shared_used_slots.value
        for pool, shared_used_slots in shared_slot_usage_by_pool.items()
    }
