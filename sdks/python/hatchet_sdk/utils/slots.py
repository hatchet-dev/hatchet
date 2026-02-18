from typing import Any

from hatchet_sdk.runnables.workflow import BaseWorkflow
from hatchet_sdk.worker.slot_types import SlotType


def normalize_slot_config(
    slot_config: dict[SlotType | str, int],
) -> dict[str, int]:
    return {
        (key.value if isinstance(key, SlotType) else key): value
        for key, value in slot_config.items()
    }


def has_slot_config(
    slot_config: dict[SlotType | str, int], slot_type: SlotType
) -> bool:
    return slot_type in slot_config or slot_type.value in slot_config


def ensure_slot_config(
    slot_config: dict[SlotType | str, int], slot_type: SlotType, default_value: int
) -> dict[SlotType | str, int]:
    if has_slot_config(slot_config, slot_type):
        return slot_config
    return {**slot_config, slot_type: default_value}


def required_slot_types_from_workflows(
    workflows: list[BaseWorkflow[Any]] | None,
) -> set[str]:
    required: set[str] = set()
    if not workflows:
        return required

    for workflow in workflows:
        for task in workflow.tasks:
            if task.is_durable:
                required.add(SlotType.DURABLE.value)
            for key in task.slot_requests:
                required.add(key.value if isinstance(key, SlotType) else key)

    return required


def resolve_worker_slot_config(
    slot_config: dict[SlotType | str, int] | None,
    slots: int | None,
    durable_slots: int | None,
    workflows: list[BaseWorkflow[Any]] | None,
) -> dict[SlotType | str, int]:
    resolved_config: dict[SlotType | str, int]

    if slot_config is not None:
        resolved_config = slot_config
    else:
        legacy_config: dict[SlotType | str, int] = {
            key: value
            for key, value in (
                (SlotType.DEFAULT, slots),
                (SlotType.DURABLE, durable_slots),
            )
            if value is not None
        }
        resolved_config = legacy_config if legacy_config else {}

    required_slot_types = required_slot_types_from_workflows(workflows)

    # Apply defaults for well-known slot types
    if SlotType.DEFAULT.value in required_slot_types:
        resolved_config = ensure_slot_config(resolved_config, SlotType.DEFAULT, 100)
    if SlotType.DURABLE.value in required_slot_types:
        resolved_config = ensure_slot_config(resolved_config, SlotType.DURABLE, 1000)

    # Raise for any required custom slot types that are missing from the config
    configured_keys = {
        key.value if isinstance(key, SlotType) else key for key in resolved_config
    }
    missing = required_slot_types - configured_keys
    if missing:
        formatted = ", ".join(sorted(missing))
        raise ValueError(
            f"Worker is missing slot config for required slot type(s): {formatted}. "
            "Please provide a slot_config entry for each custom slot type used by your workflows."
        )

    if not resolved_config:
        resolved_config = {SlotType.DEFAULT: 100}

    return resolved_config
