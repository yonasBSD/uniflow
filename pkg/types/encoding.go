package types

import (
	"reflect"
	"unsafe"

	"github.com/pkg/errors"
	"github.com/siyul-park/uniflow/pkg/encoding"
)

type Marshaler interface {
	Marshal() (Value, error)
}

type Unmarshaler interface {
	Unmarshal(Value) error
}

var (
	Encoder = encoding.NewEncodeAssembler[any, Value]()
	Decoder = encoding.NewDecodeAssembler[Value, any]()
)

func init() {
	Encoder.Add(newPointerEncoder(Encoder))
	Encoder.Add(newMapEncoder(Encoder))
	Encoder.Add(newSliceEncoder(Encoder))
	Encoder.Add(newUintegerEncoder())
	Encoder.Add(NewIntegerEncoder())
	Encoder.Add(newFloatEncoder())
	Encoder.Add(newBooleanEncoder())
	Encoder.Add(newBinaryEncoder())
	Encoder.Add(newStringEncoder())
	Encoder.Add(newErrorEncoder())
	Encoder.Add(newExpandedEncoder())
	Encoder.Add(newShortcutEncoder())

	Decoder.Add(newPointerDecoder(Decoder))
	Decoder.Add(newMapDecoder(Decoder))
	Decoder.Add(newSliceDecoder(Decoder))
	Decoder.Add(newUintegerDecoder())
	Decoder.Add(NewIntegerDecoder())
	Decoder.Add(newFloatDecoder())
	Decoder.Add(newBooleanDecoder())
	Decoder.Add(newBinaryDecoder())
	Decoder.Add(newStringDecoder())
	Decoder.Add(newErrorDecoder())
	Decoder.Add(newExpandedDecoder())
	Decoder.Add(newShortcutDecoder())
}

func newShortcutEncoder() encoding.EncodeCompiler[any, Value] {
	typeValue := reflect.TypeOf((*Value)(nil)).Elem()

	return encoding.EncodeCompilerFunc[any, Value](func(typ reflect.Type) (encoding.Encoder[any, Value], error) {
		if typ != nil && typ.ConvertibleTo(typeValue) {
			return encoding.EncodeFunc[any, Value](func(source any) (Value, error) {
				s := source.(Value)
				return s, nil
			}), nil
		}
		return nil, errors.WithStack(encoding.ErrUnsupportedType)
	})
}

func newShortcutDecoder() encoding.DecodeCompiler[Value] {
	typeValue := reflect.TypeOf((*Value)(nil)).Elem()

	return encoding.DecodeCompilerFunc[Value](func(typ reflect.Type) (encoding.Decoder[Value, unsafe.Pointer], error) {
		if typ != nil && typ.Kind() == reflect.Pointer && typ.Elem().ConvertibleTo(typeValue) {
			return encoding.DecodeFunc[Value, unsafe.Pointer](func(source Value, target unsafe.Pointer) error {
				*(*Value)(target) = source
				return nil
			}), nil
		}
		return nil, errors.WithStack(encoding.ErrUnsupportedType)
	})
}

func newExpandedEncoder() encoding.EncodeCompiler[any, Value] {
	typeMarshaler := reflect.TypeOf((*Marshaler)(nil)).Elem()

	return encoding.EncodeCompilerFunc[any, Value](func(typ reflect.Type) (encoding.Encoder[any, Value], error) {
		if typ != nil && typ.Kind() == reflect.Pointer && typ.ConvertibleTo(typeMarshaler) {
			return encoding.EncodeFunc[any, Value](func(source any) (Value, error) {
				s := source.(Marshaler)
				return s.Marshal()
			}), nil
		}
		return nil, errors.WithStack(encoding.ErrUnsupportedType)
	})
}

func newExpandedDecoder() encoding.DecodeCompiler[Value] {
	typeUnmarshaler := reflect.TypeOf((*Unmarshaler)(nil)).Elem()

	return encoding.DecodeCompilerFunc[Value](func(typ reflect.Type) (encoding.Decoder[Value, unsafe.Pointer], error) {
		if typ != nil && typ.ConvertibleTo(typeUnmarshaler) {
			return encoding.DecodeFunc[Value, unsafe.Pointer](func(source Value, target unsafe.Pointer) error {
				t := reflect.NewAt(typ.Elem(), target).Interface().(Unmarshaler)
				return t.Unmarshal(source)
			}), nil
		}
		return nil, errors.WithStack(encoding.ErrUnsupportedType)
	})
}

func newPointerEncoder(encoder *encoding.EncodeAssembler[any, Value]) encoding.EncodeCompiler[any, Value] {
	return encoding.EncodeCompilerFunc[any, Value](func(typ reflect.Type) (encoding.Encoder[any, Value], error) {
		if typ == nil {
			return encoding.EncodeFunc[any, Value](func(source any) (Value, error) {
				return nil, nil
			}), nil
		} else if typ.Kind() == reflect.Pointer {
			enc, err := encoder.Compile(typ.Elem())
			if err != nil {
				return nil, err
			}

			return encoding.EncodeFunc[any, Value](func(source any) (Value, error) {
				if source == nil {
					return nil, nil
				}
				s := reflect.ValueOf(source)
				return enc.Encode(s.Elem().Interface())
			}), nil
		}
		return nil, errors.WithStack(encoding.ErrUnsupportedType)
	})
}

func newPointerDecoder(decoder *encoding.DecodeAssembler[Value, any]) encoding.DecodeCompiler[Value] {
	return encoding.DecodeCompilerFunc[Value](func(typ reflect.Type) (encoding.Decoder[Value, unsafe.Pointer], error) {
		if typ != nil && typ.Kind() == reflect.Pointer && typ.Elem().Kind() == reflect.Pointer {
			dec, err := decoder.Compile(typ.Elem())
			if err != nil {
				return nil, err
			}

			return encoding.DecodeFunc[Value, unsafe.Pointer](func(source Value, target unsafe.Pointer) error {
				t := reflect.NewAt(typ.Elem(), target)
				if t.Elem().IsNil() {
					zero := reflect.New(t.Type().Elem().Elem())
					t.Elem().Set(zero)
				}
				return dec.Decode(source, t.Elem().UnsafePointer())
			}), nil
		}
		return nil, errors.WithStack(encoding.ErrUnsupportedType)
	})
}
