package datastore

import (
	"testing"

	"github.com/siyul-park/uniflow/pkg/scheme"
	"github.com/stretchr/testify/assert"
)

func TestAddToScheme(t *testing.T) {
	s := scheme.New()

	err := AddToScheme()(s)
	assert.NoError(t, err)

	testCase := []string{KindRDB, KindSQL, KindWrite}

	for _, tc := range testCase {
		t.Run(tc, func(t *testing.T) {
			_, ok := s.KnownType(tc)
			assert.True(t, ok)

			_, ok = s.Codec(tc)
			assert.True(t, ok)
		})
	}
}
