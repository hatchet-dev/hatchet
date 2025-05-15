from typing import overload


@overload
def apply_namespace(resource_name: str, namespace: str | None) -> str: ...


@overload
def apply_namespace(resource_name: None, namespace: str | None) -> None: ...


def apply_namespace(resource_name: str | None, namespace: str | None) -> str | None:
    if resource_name is None:
        return None

    if not namespace:
        return resource_name

    if resource_name.startswith(namespace):
        return resource_name

    return namespace + "_" + resource_name
