from collections.abc import Iterator
from typing import Any

import griffe


def _load_by_path(
    path: str, loader: griffe.GriffeLoader
) -> griffe.Object | griffe.Alias | None:
    try:
        return loader.modules_collection.get_member(path)
    except KeyError:
        pass

    if not path.startswith("hatchet_sdk."):
        return None

    ## the docs load each subpackage of hatchet_sdk as its own top-level package, and
    ## subpackages without an `__init__.py` (such as `runnables`) only exist under that
    ## top-level name, so `hatchet_sdk.runnables.types` is found as `runnables.types`
    for candidate_path in (path, path.removeprefix("hatchet_sdk.")):
        try:
            return loader.load(candidate_path)
        except (ImportError, KeyError):
            continue

    return None


def _resolve_name(
    name: griffe.ExprName, loader: griffe.GriffeLoader
) -> griffe.Object | griffe.Alias | None:
    target = _load_by_path(name.canonical_path, loader)

    if isinstance(target, griffe.Alias):
        try:
            return target.final_target
        except (griffe.AliasResolutionError, griffe.CyclicAliasError):
            return None

    return target


def _literal_alias_value(
    name: griffe.ExprName, loader: griffe.GriffeLoader
) -> griffe.Expr | None:
    target = _resolve_name(name, loader)

    if (
        isinstance(target, griffe.Attribute)
        and isinstance(target.value, griffe.Expr)
        and _is_literal_type(target.value, loader)
    ):
        return target.value

    return None


def _is_literal_type(expression: griffe.Expr, loader: griffe.GriffeLoader) -> bool:
    if isinstance(expression, griffe.ExprSubscript):
        return str(expression.left) == "Literal"

    if isinstance(expression, griffe.ExprBinOp) and expression.operator == "|":
        return all(
            isinstance(side, griffe.Expr) and _is_literal_type(side, loader)
            for side in (expression.left, expression.right)
        )

    if isinstance(expression, griffe.ExprName):
        return _literal_alias_value(expression, loader) is not None

    return False


def _expand_literal_aliases(
    annotation: str | griffe.Expr | None, loader: griffe.GriffeLoader
) -> str | griffe.Expr | None:
    if not isinstance(annotation, griffe.Expr):
        return annotation

    parts: list[str] = []
    expanded_any_alias = False

    for part in annotation.iterate(flat=True):
        alias_value = (
            _literal_alias_value(part, loader)
            if isinstance(part, griffe.ExprName)
            else None
        )

        if alias_value is None:
            parts.append(str(part))
        else:
            parts.append(str(_expand_literal_aliases(alias_value, loader)))
            expanded_any_alias = True

    return "".join(parts) if expanded_any_alias else annotation


def _documented_objects(obj: griffe.Object) -> Iterator[griffe.Object]:
    for member in obj.members.values():
        if member.is_alias or not isinstance(member, griffe.Object):
            continue

        yield member
        yield from _documented_objects(member)


def _expand_package_annotations(
    pkg: griffe.Module, loader: griffe.GriffeLoader
) -> None:
    for obj in _documented_objects(pkg):
        if isinstance(obj, griffe.Function):
            for parameter in obj.parameters:
                parameter.annotation = _expand_literal_aliases(
                    parameter.annotation, loader
                )
            obj.returns = _expand_literal_aliases(obj.returns, loader)
        elif isinstance(obj, griffe.Attribute):
            obj.annotation = _expand_literal_aliases(obj.annotation, loader)


class ExpandLiteralAliasAnnotations(griffe.Extension):
    """Show `Literal` type aliases such as `RunStatus` as the strings they allow, since the alias name alone tells a reader nothing."""

    def __init__(self) -> None:
        super().__init__()
        self._is_expanding = False
        self._packages_to_expand: list[griffe.Module] = []

    def on_package(
        self,
        *,
        pkg: griffe.Module,
        loader: griffe.GriffeLoader,
        **kwargs: Any,  # noqa: ARG002, ANN401
    ) -> None:
        self._packages_to_expand.append(pkg)

        ## resolving an alias can load another package, which calls back into this hook,
        ## so packages loaded mid-expansion are queued instead of expanded recursively
        if self._is_expanding:
            return

        self._is_expanding = True
        try:
            while self._packages_to_expand:
                _expand_package_annotations(self._packages_to_expand.pop(), loader)
        finally:
            self._is_expanding = False
