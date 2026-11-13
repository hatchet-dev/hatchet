import importlib
from collections.abc import Iterator
from types import UnionType
from typing import Any, Literal, Union, get_args, get_origin

import griffe


def _import_runtime_value(canonical_path: str) -> object | None:
    ## the docs load each subpackage of hatchet_sdk as its own top-level package, so a
    ## name defined in `features/cron.py` has the canonical path `features.cron.X`
    full_path = (
        canonical_path
        if canonical_path.startswith("hatchet_sdk.")
        else f"hatchet_sdk.{canonical_path}"
    )
    module_path, _, attribute_name = full_path.rpartition(".")

    try:
        return getattr(importlib.import_module(module_path), attribute_name, None)
    except ImportError:
        return None


def _render_literal_type(value: object) -> str | None:
    if get_origin(value) is Literal:
        return "Literal[" + ", ".join(repr(arg) for arg in get_args(value)) + "]"

    if get_origin(value) in (Union, UnionType):
        rendered_members = [_render_literal_type(arg) for arg in get_args(value)]
        if all(rendered_members):
            return " | ".join(str(member) for member in rendered_members)

    return None


def _expand_literal_aliases(
    annotation: str | griffe.Expr | None,
) -> str | griffe.Expr | None:
    if not isinstance(annotation, griffe.Expr):
        return annotation

    parts: list[str] = []
    expanded_any_alias = False

    for part in annotation.iterate(flat=True):
        rendered_alias = (
            _render_literal_type(_import_runtime_value(part.canonical_path))
            if isinstance(part, griffe.ExprName)
            else None
        )

        if rendered_alias is None:
            parts.append(str(part))
        else:
            parts.append(rendered_alias)
            expanded_any_alias = True

    return "".join(parts) if expanded_any_alias else annotation


def _documented_objects(obj: griffe.Object) -> Iterator[griffe.Object]:
    for member in obj.members.values():
        if member.is_alias or not isinstance(member, griffe.Object):
            continue

        yield member
        yield from _documented_objects(member)


class ExpandLiteralAliasAnnotations(griffe.Extension):
    """Show `Literal` type aliases such as `RunStatus` as the strings they allow, since the alias name alone tells a reader nothing."""

    def on_package(
        self,
        *,
        pkg: griffe.Module,
        **kwargs: Any,  # noqa: ARG002, ANN401
    ) -> None:
        for obj in _documented_objects(pkg):
            if isinstance(obj, griffe.Function):
                for parameter in obj.parameters:
                    parameter.annotation = _expand_literal_aliases(parameter.annotation)
                obj.returns = _expand_literal_aliases(obj.returns)
            elif isinstance(obj, griffe.Attribute):
                obj.annotation = _expand_literal_aliases(obj.annotation)
