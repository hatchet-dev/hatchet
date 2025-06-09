from enum import Enum
from typing import TypeVar, overload

from google.protobuf.internal.enum_type_wrapper import EnumTypeWrapper

TProtoEnumValue = TypeVar("TProtoEnumValue", bound=int)

TProtoEnum = TypeVar("TProtoEnum", bound=EnumTypeWrapper)
TPythonEnum = TypeVar("TPythonEnum", bound=Enum)


def convert_python_enum_to_proto(
    value: TPythonEnum | None, proto_enum: TProtoEnum
) -> int | None:
    if value is None:
        return None

    names = [item.name for item in proto_enum.DESCRIPTOR.values]

    for name in names:
        if name == value.name:
            return proto_enum.Value(value.name)

    raise ValueError(f"Value must be one of {names}. Got: {value}")


@overload
def convert_proto_enum_to_python(
    value: TProtoEnumValue, python_enum_class: type[TPythonEnum], proto_enum: TProtoEnum
) -> TPythonEnum: ...


@overload
def convert_proto_enum_to_python(
    value: None, python_enum_class: type[TPythonEnum], proto_enum: TProtoEnum
) -> None: ...


def convert_proto_enum_to_python(
    value: TProtoEnumValue | None,
    python_enum_class: type[TPythonEnum],
    proto_enum: TProtoEnum,
) -> TPythonEnum | None:
    if value is None:
        return None

    return python_enum_class[proto_enum.Name(value)]
