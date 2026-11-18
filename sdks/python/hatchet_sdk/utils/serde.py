from typing import TypeVar, cast, overload

T = TypeVar("T")
K = TypeVar("K")
R = TypeVar("R")


@overload
def remove_null_unicode_character(
    data: dict[K, T], replacement: str = ""
) -> dict[K, T]: ...


@overload
def remove_null_unicode_character(data: list[T], replacement: str = "") -> list[T]: ...


@overload
def remove_null_unicode_character(
    data: tuple[T, ...], replacement: str = ""
) -> tuple[T, ...]: ...


@overload
def remove_null_unicode_character(data: R, replacement: str = "") -> R: ...


def remove_null_unicode_character(data: object, replacement: str = "") -> object:
    """
    Recursively traverse a dictionary (a task's output) and remove the unicode escape sequence \\u0000 which will cause unexpected behavior in Hatchet.

    Needed as Hatchet does not support \\u0000 in task outputs

    :param data: The task output (a JSON-serializable dictionary or mapping)
    :param replacement: The string to replace \\u0000 with.

    :return: The same dictionary with all \\u0000 characters removed from strings, and nested dictionaries/lists processed recursively.
    """
    if isinstance(data, str):
        return data.replace("\u0000", replacement)

    if isinstance(data, dict):
        mapping = cast("dict[object, object]", data)
        return {
            remove_null_unicode_character(
                key, replacement
            ): remove_null_unicode_character(value, replacement)
            for key, value in mapping.items()
        }

    if isinstance(data, list):
        items = cast("list[object]", data)
        return [remove_null_unicode_character(item, replacement) for item in items]

    if isinstance(data, tuple):
        tuple_items = cast("tuple[object, ...]", data)
        return tuple(
            remove_null_unicode_character(item, replacement) for item in tuple_items
        )

    return data
