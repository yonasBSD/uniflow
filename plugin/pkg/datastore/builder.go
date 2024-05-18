package datastore

import (
	"github.com/siyul-park/uniflow/pkg/scheme"
)

// AddToScheme returns a function that adds node types and codecs to the provided scheme.
func AddToScheme() func(*scheme.Scheme) error {
	return func(s *scheme.Scheme) error {
		s.AddKnownType(KindRDB, &RDBNodeSpec{})
		s.AddCodec(KindRDB, NewRDBNodeCodec())

		s.AddKnownType(KindSQL, &SQLNodeSpec{})
		s.AddCodec(KindSQL, NewSQLNodeCodec())

		s.AddKnownType(KindWrite, &WriteNodeSpec{})
		s.AddCodec(KindWrite, NewWriteNodeCodec())

		return nil
	}
}
