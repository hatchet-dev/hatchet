from typing import Any


def flatten(xs: dict[str, Any], parent_key: str, separator: str) -> dict[str, Any]:
    if not xs:
        return {}

    items: list[tuple[str, Any]] = []

    for k, v in xs.items():
        new_key = parent_key + separator + k if parent_key else k

        if isinstance(v, dict):
            items.extend(flatten(v, new_key, separator).items())
        else:
            items.append((new_key, v))

    return dict(items)
