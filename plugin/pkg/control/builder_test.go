package control

import (
	"testing"

	"github.com/siyul-park/uniflow/pkg/scheme"
	"github.com/stretchr/testify/assert"
)

func TestAddToScheme(t *testing.T) {
	s := scheme.New()

	err := AddToScheme()(s)
	assert.NoError(t, err)

	_, ok := s.KnownType(KindSnippet)
	assert.True(t, ok)

	_, ok = s.Codec(KindSnippet)
	assert.True(t, ok)
}