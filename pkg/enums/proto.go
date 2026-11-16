package enums

import (
	"fmt"

	"google.golang.org/protobuf/reflect/protoreflect"
)

// ProtoEnum is any protoc-gen-go enum type.
type ProtoEnum interface {
	~int32
	protoreflect.Enum
}

type validator interface {
	Valid() bool
}

// FromProto converts a proto enum to the string enum with the same value name, rejecting names T's Valid method (if any) refuses.
func FromProto[T ~string, P ProtoEnum](p P) (T, error) {
	ev := p.Descriptor().Values().ByNumber(p.Number())
	if ev == nil {
		return "", fmt.Errorf("unknown %s value %d", p.Descriptor().FullName(), p.Number())
	}

	v := T(ev.Name())

	if vv, ok := any(v).(validator); ok && !vv.Valid() {
		return "", fmt.Errorf("%s value %s has no %T equivalent", p.Descriptor().FullName(), ev.Name(), v)
	}

	return v, nil
}

// FromProtoPtr is FromProto for optional proto fields; nil maps to nil.
func FromProtoPtr[T ~string, P ProtoEnum](p *P) (*T, error) {
	if p == nil {
		return nil, nil
	}

	v, err := FromProto[T](*p)
	if err != nil {
		return nil, err
	}

	return &v, nil
}

// ToProto converts a string enum to the proto enum with the same value name.
func ToProto[P ProtoEnum, T ~string](v T) (P, error) {
	var zero P

	ev := zero.Descriptor().Values().ByName(protoreflect.Name(v))
	if ev == nil {
		return zero, fmt.Errorf("%q has no %s equivalent", v, zero.Descriptor().FullName())
	}

	return P(ev.Number()), nil
}

// ToProtoPtr is ToProto for optional fields; nil maps to nil.
func ToProtoPtr[P ProtoEnum, T ~string](v *T) (*P, error) {
	if v == nil {
		return nil, nil
	}

	p, err := ToProto[P](*v)
	if err != nil {
		return nil, err
	}

	return &p, nil
}
