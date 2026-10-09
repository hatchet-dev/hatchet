from enum import Enum
from typing import TYPE_CHECKING, TypeVar, cast, overload

if TYPE_CHECKING:
    from google.protobuf.internal.enum_type_wrapper import EnumTypeWrapper

TProtoEnumValue = TypeVar("TProtoEnumValue", bound=int)
TPythonEnum = TypeVar("TPythonEnum", bound=Enum)


def convert_python_enum_to_proto(
    value: TPythonEnum | None, proto_enum: type[TProtoEnumValue]
) -> TProtoEnumValue | None:
    if value is None:
        return None

    return convert_python_literal_to_proto(value.name, proto_enum)


def convert_python_literal_to_proto(
    value: str | None, proto_enum: type[TProtoEnumValue]
) -> TProtoEnumValue | None:
    if value is None:
        return None

    # The stubs present proto enums as `int` subclasses, but at runtime they are
    # `EnumTypeWrapper` instances, which is where `DESCRIPTOR` and `Value` live.
    wrapper = cast("EnumTypeWrapper", proto_enum)
    names = [item.name for item in wrapper.DESCRIPTOR.values]

    for name in names:
        if name == value:
            return cast("TProtoEnumValue", wrapper.Value(value))

    raise ValueError(f"Value must be one of {names}. Got: {value}")


@overload
def convert_proto_enum_to_python(
    value: TProtoEnumValue,
    python_enum_class: type[TPythonEnum],
    proto_enum: type[TProtoEnumValue],
) -> TPythonEnum: ...


@overload
def convert_proto_enum_to_python(
    value: None,
    python_enum_class: type[TPythonEnum],
    proto_enum: type[TProtoEnumValue],
) -> None: ...


def convert_proto_enum_to_python(
    value: TProtoEnumValue | None,
    python_enum_class: type[TPythonEnum],
    proto_enum: type[TProtoEnumValue],
) -> TPythonEnum | None:
    if value is None:
        return None

    wrapper = cast("EnumTypeWrapper", proto_enum)

    return python_enum_class[wrapper.Name(value)]
