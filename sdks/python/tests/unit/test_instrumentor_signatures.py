import importlib
import inspect
import types
import typing
from collections.abc import Callable
from types import ModuleType
from typing import Any, Union, cast

import pytest

from hatchet_sdk.opentelemetry.instrumentor import HatchetInstrumentor

WRAPPED_PAIRS = [
    (
        "worker.runner.runner.Runner.handle_start_step_run",
        "_wrap_handle_start_step_run",
    ),
    ("worker.runner.runner.Runner.handle_cancel_action", "_wrap_handle_cancel_action"),
    ("clients.events.EventClient.push", "_wrap_push_event"),
    ("clients.events.EventClient.bulk_push", "_wrap_bulk_push_event"),
    ("clients.admin.AdminClient.run_workflow", "_wrap_run_workflow"),
    ("clients.admin.AdminClient.aio_run_workflow", "_wrap_async_run_workflow"),
    ("clients.admin.AdminClient.schedule_workflow", "_wrap_schedule_workflow"),
    ("clients.admin.AdminClient.run_workflows", "_wrap_run_workflows"),
    ("clients.admin.AdminClient.aio_run_workflows", "_wrap_async_run_workflows"),
    ("context.context.DurableContext.aio_wait_for", "_wrap_aio_wait_for"),
    (
        "context.context.DurableContext._spawn_children_no_wait",
        "_wrap_spawn_children_no_wait",
    ),
]


def _resolve_method(dotted_path: str) -> Callable[..., Any]:
    parts = dotted_path.split(".")
    mod: ModuleType | None = None
    split = 0
    for i in range(1, len(parts) + 1):
        try:
            mod = importlib.import_module("hatchet_sdk." + ".".join(parts[:i]))
            split = i
        except ModuleNotFoundError:
            break

    assert mod is not None
    result = cast(Callable[..., Any], mod)

    for attr in parts[split:]:
        result = getattr(result, attr)

    return result


def _get_sig(func: Callable[..., Any]) -> inspect.Signature:
    try:
        import annotationlib  # type: ignore[import-not-found, unused-ignore]

        return inspect.signature(func, annotation_format=annotationlib.Format.STRING)  # type: ignore[call-arg, unused-ignore]
    except Exception:
        return inspect.signature(func)


def _get_param_types(func: Callable[..., Any]) -> list[Any] | None:
    try:
        hints = typing.get_type_hints(func)
    except Exception:
        return None

    sig = _get_sig(func)
    return [hints[name] for name in sig.parameters if name not in ("self", "return")]


def _get_param_count(func: Callable[..., Any]) -> int:
    sig = _get_sig(func)
    return len([n for n in sig.parameters if n != "self"])


def _get_tuple_type_args(annotation: Any) -> list[Any] | None:
    type_args: tuple[Any, ...] | None = getattr(annotation, "__args__", None)

    if type_args is None:
        return None

    if len(type_args) == 2 and type_args[1] is Ellipsis:
        return None

    return list(type_args)


def _flatten_union(t: Any) -> set[Any]:
    origin = getattr(t, "__origin__", None)
    if isinstance(t, types.UnionType) or origin is types.UnionType or origin is Union:
        result: set[Any] = set()
        for a in t.__args__:
            result |= _flatten_union(a)
        return result
    return {t}


def _get_kwargs_value_types(wrapper: Callable[..., Any]) -> set[Any] | None:
    sig = inspect.signature(wrapper)
    if "kwargs" not in sig.parameters:
        return None

    annotation = sig.parameters["kwargs"].annotation
    dict_args: tuple[Any, ...] | None = getattr(annotation, "__args__", None)
    if dict_args is None or len(dict_args) < 2:
        return None

    return _flatten_union(dict_args[1])


@pytest.mark.parametrize(
    "dotted_path,wrapper_name",
    WRAPPED_PAIRS,
    ids=[p[1] for p in WRAPPED_PAIRS],
)
def test_wrapper_args_match_wrapped_signature(
    dotted_path: str, wrapper_name: str
) -> None:
    wrapped_method = _resolve_method(dotted_path)
    wrapper = getattr(HatchetInstrumentor, wrapper_name)
    wrapper_arg_types = _get_tuple_type_args(
        inspect.signature(wrapper).parameters["args"].annotation
    )

    if wrapper_arg_types is None:
        return

    wrapped_count = _get_param_count(wrapped_method)
    assert len(wrapper_arg_types) == wrapped_count, (
        f"{wrapper_name}: args tuple has {len(wrapper_arg_types)} elements "
        f"but {dotted_path} has {wrapped_count}"
    )

    wrapped_types = _get_param_types(wrapped_method)
    if wrapped_types is None:
        return

    for i, (wrapper_type, wrapped_type) in enumerate(
        zip(wrapper_arg_types, wrapped_types)
    ):
        assert wrapper_type == wrapped_type, (
            f"{wrapper_name}: arg {i} type mismatch: "
            f"wrapper has {wrapper_type!r}, wrapped has {wrapped_type!r}"
        )

    # The kwargs value type union must cover every type in the wrapped params
    kwargs_types = _get_kwargs_value_types(wrapper)
    if kwargs_types is None:
        return

    all_wrapped_types = set()
    for t in wrapped_types:
        all_wrapped_types |= _flatten_union(t)

    missing = all_wrapped_types - kwargs_types
    assert not missing, (
        f"{wrapper_name}: kwargs is missing types {missing} "
        f"that appear in {dotted_path}'s params"
    )
