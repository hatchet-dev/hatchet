from hatchet_sdk.runnables.workflow import BaseWorkflow
from hatchet_sdk.worker.slot_types import SlotType


def normalize_slot_capacities(
    slot_capacities: dict[SlotType | str, int],
) -> dict[str, int]:
    normalized: dict[str, int] = {}
    for key, value in slot_capacities.items():
        normalized_key = key.value if isinstance(key, SlotType) else key
        normalized[normalized_key] = value
    return normalized


def has_slot_capacity(
    slot_capacities: dict[SlotType | str, int], slot_type: SlotType
) -> bool:
    return slot_type in slot_capacities or slot_type.value in slot_capacities


def ensure_slot_capacity(
    slot_capacities: dict[SlotType | str, int], slot_type: SlotType, default_value: int
) -> None:
    if not has_slot_capacity(slot_capacities, slot_type):
        slot_capacities[slot_type] = default_value


def required_slot_types_from_workflows(
    workflows: list[BaseWorkflow] | None,
) -> set[SlotType]:
    required: set[SlotType] = set()
    if not workflows:
        return required

    for workflow in workflows:
        for task in workflow.tasks:
            if task.is_durable:
                required.add(SlotType.DURABLE)
            for key in task.slot_requirements.keys():
                if key == SlotType.DEFAULT or key == SlotType.DEFAULT.value:
                    required.add(SlotType.DEFAULT)
                if key == SlotType.DURABLE or key == SlotType.DURABLE.value:
                    required.add(SlotType.DURABLE)

    return required


def resolve_worker_slot_capacities(
    slot_capacities: dict[SlotType | str, int] | None,
    slots: int | None,
    durable_slots: int | None,
    workflows: list[BaseWorkflow] | None,
) -> dict[SlotType | str, int]:
    resolved_capacities = slot_capacities

    if resolved_capacities is None:
        legacy_capacities = {
            key: value
            for key, value in (
                (SlotType.DEFAULT, slots),
                (SlotType.DURABLE, durable_slots),
            )
            if value is not None
        }
        resolved_capacities = legacy_capacities or {}

    required_slot_types = required_slot_types_from_workflows(workflows)
    if SlotType.DEFAULT in required_slot_types:
        ensure_slot_capacity(resolved_capacities, SlotType.DEFAULT, 100)
    if SlotType.DURABLE in required_slot_types:
        ensure_slot_capacity(resolved_capacities, SlotType.DURABLE, 1000)

    if not resolved_capacities:
        resolved_capacities[SlotType.DEFAULT] = 100

    return resolved_capacities
