package object

import (
	"fmt"
	"reflect"
	"strings"
	"unsafe"

	"github.com/benbjohnson/immutable"
	"github.com/iancoleman/strcase"
	"github.com/pkg/errors"
	"github.com/siyul-park/uniflow/pkg/encoding"
)

// Map represents a map structure.
type Map struct {
	value *immutable.SortedMap[Object, Object]
}

// mapTag represents the tag for map fields.
type mapTag struct {
	alias     string
	ignore    bool
	omitempty bool
	inline    bool
}

type comparer struct{}

const tagMap = "map"

var _ Object = (*Map)(nil)
var _ immutable.Comparer[Object] = (*comparer)(nil)

// NewMap creates a new Map with key-value pairs.
func NewMap(pairs ...Object) *Map {
	builder := immutable.NewSortedMapBuilder[Object, Object](&comparer{})
	for i := 0; i < len(pairs)/2; i++ {
		k, v := pairs[i*2], pairs[i*2+1]
		builder.Set(k, v)
	}
	return &Map{value: builder.Map()}
}

// Get retrieves the value for a given key.
func (m *Map) Get(key Object) (Object, bool) {
	return m.value.Get(key)
}

// GetOr returns the value for a given key or a default value if the key is not found.
func (m *Map) GetOr(key, value Object) Object {
	if v, ok := m.Get(key); ok {
		return v
	}
	return value
}

// Set adds or updates a key-value pair in the map.
func (m *Map) Set(key, value Object) *Map {
	return &Map{value: m.value.Set(key, value)}
}

// Delete removes a key and its corresponding value from the map.
func (m *Map) Delete(key Object) *Map {
	return &Map{value: m.value.Delete(key)}
}

// Keys returns all keys in the map.
func (m *Map) Keys() []Object {
	var keys []Object
	itr := m.value.Iterator()

	for !itr.Done() {
		k, _, _ := itr.Next()
		keys = append(keys, k)
	}
	return keys
}

// Values returns all values in the map.
func (m *Map) Values() []Object {
	var values []Object
	itr := m.value.Iterator()

	for !itr.Done() {
		_, v, _ := itr.Next()
		values = append(values, v)
	}
	return values
}

// Pairs returns all keys and values in the map.
func (m *Map) Pairs() []Object {
	var pairs []Object
	itr := m.value.Iterator()

	for !itr.Done() {
		k, v, _ := itr.Next()
		pairs = append(pairs, k, v)
	}
	return pairs
}

// Len returns the number of key-value pairs in the map.
func (m *Map) Len() int {
	return m.value.Len()
}

// Map converts the Map to a raw Go map.
func (m *Map) Map() map[any]any {
	result := make(map[any]any, m.value.Len())

	itr := m.value.Iterator()
	for !itr.Done() {
		k, v, _ := itr.Next()

		if k != nil {
			result[k.Interface()] = v.Interface()
		}
	}

	return result
}

// Merge merges the contents of the other Map into the current Map.
// If there are any overlapping keys, the values from the other Map will overwrite the values in the current Map.
func (m *Map) Merge(other *Map) *Map {
	return NewMap(append(m.Pairs(), other.Pairs()...)...)
}

// Kind returns the kind of the Map.
func (m *Map) Kind() Kind {
	return KindMap
}

// Compare compares two maps.
func (m *Map) Compare(v Object) int {
	if r, ok := v.(*Map); ok {
		keys1, keys2 := m.Keys(), r.Keys()

		if len(keys1) < len(keys2) {
			return -1
		} else if len(keys1) > len(keys2) {
			return 1
		}

		for i, k1 := range keys1 {
			k2 := keys2[i]
			if diff := Compare(k1, k2); diff != 0 {
				return diff
			}

			v1, ok1 := m.Get(k1)
			v2, ok2 := r.Get(k2)

			if diff := Compare(NewBool(ok1), NewBool(ok2)); diff != 0 {
				return diff
			}
			if diff := Compare(v1, v2); diff != 0 {
				return diff
			}
		}

		return 0
	}

	if KindOf(m) > KindOf(v) {
		return 1
	}
	return -1
}

// Interface converts the Map to an interface{}.
func (m *Map) Interface() any {
	keys := make([]any, m.value.Len())
	values := make([]any, m.value.Len())

	itr := m.value.Iterator()
	for i := 0; !itr.Done(); i++ {
		k, v, _ := itr.Next()

		if k != nil {
			keys[i] = k.Interface()
		}
		if v != nil {
			values[i] = v.Interface()
		}
	}

	keyType := getCommonType(keys)
	valueType := getCommonType(values)

	t := reflect.MakeMapWithSize(reflect.MapOf(keyType, valueType), len(keys))
	for i, key := range keys {
		value := values[i]
		t.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(value))
	}

	return t.Interface()
}

func (*comparer) Compare(a Object, b Object) int {
	return Compare(a, b)
}

func newMapEncoder(encoder *encoding.Assembler[*Object, any]) encoding.Compiler[*Object] {
	return encoding.CompilerFunc[*Object](func(typ reflect.Type) (encoding.Encoder[*Object, unsafe.Pointer], error) {
		if typ.Kind() == reflect.Pointer {
			if typ.Elem().Kind() == reflect.Map {
				keyType := typ.Elem().Key()
				valueType := typ.Elem().Elem()

				keyEncoder, _ := encoder.Compile(keyType)
				valueEncoder, _ := encoder.Compile(valueType)

				return encoding.EncodeFunc[*Object, unsafe.Pointer](func(source *Object, target unsafe.Pointer) error {
					t := reflect.NewAt(typ.Elem(), target).Elem()

					pairs := make([]Object, 0, t.Len()*2)
					for _, k := range t.MapKeys() {
						v := t.MapIndex(k)

						k = reflect.ValueOf(k.Interface())
						v = reflect.ValueOf(v.Interface())

						var key Object
						if keyEncoder != nil && k.Type() == keyType {
							kPtr := reflect.New(keyType)
							kPtr.Elem().Set(k)

							if err := keyEncoder.Encode(&key, kPtr.UnsafePointer()); err != nil {
								return err
							}
						} else if err := encoder.Encode(&key, k.Interface()); err != nil {
							return err
						}
						pairs = append(pairs, key)

						var val Object
						if valueEncoder != nil && k.Type() == keyType {
							vPtr := reflect.New(valueType)
							vPtr.Elem().Set(v)

							if err := valueEncoder.Encode(&val, vPtr.UnsafePointer()); err != nil {
								return err
							}
						} else if err := encoder.Encode(&val, v.Interface()); err != nil {
							return err
						}
						pairs = append(pairs, val)
					}
					*source = NewMap(pairs...)
					return nil
				}), nil
			} else if typ.Elem().Kind() == reflect.Struct {
				var encoders []encoding.Encoder[*[]Object, unsafe.Pointer]

				for i := 0; i < typ.Elem().NumField(); i++ {
					field := typ.Elem().Field(i)
					tag := getMapTag(field)

					if !field.IsExported() || tag.ignore {
						continue
					}

					child, err := encoder.Compile(field.Type)
					if err != nil {
						return nil, err
					}

					offset := field.Offset
					alias := NewString(tag.alias)

					var enc encoding.Encoder[*[]Object, unsafe.Pointer]
					if tag.inline {
						enc = encoding.EncodeFunc[*[]Object, unsafe.Pointer](func(source *[]Object, target unsafe.Pointer) error {
							var s Object
							if err := child.Encode(&s, unsafe.Pointer(uintptr(target)+offset)); err != nil {
								return err
							} else if t, ok := s.(*Map); !ok {
								return errors.WithStack(encoding.ErrInvalidValue)
							} else {
								*source = append(*source, t.Pairs()...)
							}
							return nil
						})
					} else {
						enc = encoding.EncodeFunc[*[]Object, unsafe.Pointer](func(source *[]Object, target unsafe.Pointer) error {
							t := unsafe.Pointer(uintptr(target) + offset)
							if tag.omitempty {
								if t := reflect.NewAt(field.Type, t).Elem(); t.IsZero() {
									return nil
								}
							}

							var s Object
							if err := child.Encode(&s, t); err != nil {
								return err
							} else {
								*source = append(*source, alias, s)
							}
							return nil
						})
					}

					encoders = append(encoders, enc)
				}

				return encoding.EncodeFunc[*Object, unsafe.Pointer](func(source *Object, target unsafe.Pointer) error {
					pairs := make([]Object, 0, len(encoders)*2)
					for _, enc := range encoders {
						if err := enc.Encode(&pairs, target); err != nil {
							return err
						}
					}
					*source = NewMap(pairs...)
					return nil
				}), nil
			}
		}
		return nil, errors.WithStack(encoding.ErrUnsupportedValue)
	})
}

func newMapDecoder(decoder *encoding.Assembler[Object, any]) encoding.Compiler[Object] {
	return encoding.CompilerFunc[Object](func(typ reflect.Type) (encoding.Encoder[Object, unsafe.Pointer], error) {
		if typ.Kind() == reflect.Pointer {
			if typ.Elem().Kind() == reflect.Map {
				keyType := typ.Elem().Key()
				valueType := typ.Elem().Elem()

				keyDecoder, err := decoder.Compile(keyType)
				if err != nil {
					return nil, err
				}
				valueDecoder, err := decoder.Compile(valueType)
				if err != nil {
					return nil, err
				}

				return encoding.EncodeFunc[Object, unsafe.Pointer](func(source Object, target unsafe.Pointer) error {
					if s, ok := source.(*Map); ok {
						t := reflect.NewAt(typ.Elem(), target).Elem()
						if t.IsNil() {
							t.Set(reflect.MakeMapWithSize(t.Type(), s.Len()))
						}

						for _, key := range s.Keys() {
							value, _ := s.Get(key)

							k := reflect.New(keyType)
							v := reflect.New(valueType)

							if err := keyDecoder.Encode(key, k.UnsafePointer()); err != nil {
								return err
							} else if err := valueDecoder.Encode(value, v.UnsafePointer()); err != nil {
								return err
							} else {
								t.SetMapIndex(k.Elem(), v.Elem())
							}
						}
						return nil
					}
					return errors.WithStack(encoding.ErrUnsupportedValue)
				}), nil
			} else if typ.Elem().Kind() == reflect.Struct {
				var decoders []encoding.Encoder[*Map, unsafe.Pointer]
				for i := 0; i < typ.Elem().NumField(); i++ {
					field := typ.Elem().Field(i)
					tag := getMapTag(field)

					if !field.IsExported() || tag.ignore {
						continue
					}

					child, err := decoder.Compile(field.Type)
					if err != nil {
						return nil, err
					}

					offset := field.Offset
					alias := NewString(tag.alias)

					var dec encoding.Encoder[*Map, unsafe.Pointer]
					if tag.inline {
						dec = encoding.EncodeFunc[*Map, unsafe.Pointer](func(source *Map, target unsafe.Pointer) error {
							return child.Encode(source, unsafe.Pointer(uintptr(target)+offset))
						})
					} else {
						dec = encoding.EncodeFunc[*Map, unsafe.Pointer](func(source *Map, target unsafe.Pointer) error {
							value, ok := source.Get(alias)
							if !ok {
								if !tag.omitempty {
									return errors.WithMessage(encoding.ErrInvalidValue, fmt.Sprintf("key(%v) is zero value", field.Name))
								}
								return nil
							}
							return child.Encode(value, unsafe.Pointer(uintptr(target)+offset))
						})
					}

					decoders = append(decoders, dec)
				}

				return encoding.EncodeFunc[Object, unsafe.Pointer](func(source Object, target unsafe.Pointer) error {
					if s, ok := source.(*Map); ok {
						for _, dec := range decoders {
							if err := dec.Encode(s, target); err != nil {
								return err
							}
						}
						return nil
					}
					return errors.WithStack(encoding.ErrUnsupportedValue)
				}), nil
			} else if typ.Elem().Kind() == reflect.Interface {
				return encoding.EncodeFunc[Object, unsafe.Pointer](func(source Object, target unsafe.Pointer) error {
					if s, ok := source.(*Map); ok {
						*(*any)(target) = s.Interface()
						return nil
					}
					return errors.WithStack(encoding.ErrUnsupportedValue)
				}), nil
			}
		}
		return nil, errors.WithStack(encoding.ErrUnsupportedValue)
	})
}

func getMapTag(f reflect.StructField) mapTag {
	key := strcase.ToSnake(f.Name)
	rawTag := f.Tag.Get(tagMap)

	if rawTag != "" {
		if rawTag == "-" {
			return mapTag{ignore: true}
		}

		if index := strings.Index(rawTag, ","); index != -1 {
			tag := mapTag{}
			tag.alias = key
			if rawTag[:index] != "" {
				tag.alias = rawTag[:index]
			}

			if rawTag[index+1:] == "omitempty" {
				tag.omitempty = true
			} else if rawTag[index+1:] == "inline" {
				tag.alias = ""
				tag.inline = true
			}
			return tag
		} else {
			return mapTag{alias: rawTag}
		}
	}

	return mapTag{alias: key}
}