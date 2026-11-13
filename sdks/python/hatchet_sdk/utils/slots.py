from typing import Any

from hatchet_sdk.runnables.workflow import BaseWorkflow
from hatchet_sdk.types.slot_types import SlotType


def ensure_slot_config(
    slot_config: dict[SlotType | str, int], slot_type: SlotType, default_value: int
) -> dict[SlotType | str, int]:
    if slot_type in slot_config:
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
            if task._is_durable:
                required.add("durable")
            required.update(task._slot_requests)

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
                ("default", slots),
                ("durable", durable_slots),
            )
            if value is not None
        }
        resolved_config = legacy_config if legacy_config else {}

    required_slot_types = required_slot_types_from_workflows(workflows)

    # Apply defaults for well-known slot types
    if "default" in required_slot_types:
        resolved_config = ensure_slot_config(resolved_config, "default", 100)
    if "durable" in required_slot_types:
        resolved_config = ensure_slot_config(resolved_config, "durable", 1000)

    # Raise for any required custom slot types that are missing from the config
    missing = required_slot_types - resolved_config.keys()
    if missing:
        formatted = ", ".join(sorted(missing))
        raise ValueError(
            f"Worker is missing slot config for required slot type(s): {formatted}. "
            "Please provide a slot_config entry for each custom slot type used by your workflows."
        )

    if not resolved_config:
        resolved_config = {"default": 100}

    return resolved_config
