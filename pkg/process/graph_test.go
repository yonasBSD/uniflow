package process

import (
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
)

func TestGraph_Add(t *testing.T) {
	g := newGraph()

	v1 := ulid.Make()
	v2 := ulid.Make()

	g.Add(v1, v2)

	assert.True(t, g.Has(v1, v2))
	assert.False(t, g.Has(v2, v1))

	g.Add(v1, v2)

	assert.True(t, g.Has(v1, v2))
	assert.False(t, g.Has(v2, v1))
}

func TestGraph_Delete(t *testing.T) {
	g := newGraph()

	v1 := ulid.Make()
	v2 := ulid.Make()

	g.Add(v1, v2)

	g.Delete(v1, v2)

	assert.False(t, g.Has(v1, v2))
	assert.False(t, g.Has(v2, v1))

	g.Delete(v1, v2)

	assert.False(t, g.Has(v1, v2))
	assert.False(t, g.Has(v2, v1))
}

func TestGraph_Upwards(t *testing.T) {
	g := newGraph()

	v1 := ulid.Make()
	v2 := ulid.Make()

	g.Add(v1, v2)

	var trace []ulid.ULID
	g.Upwards(v2, func(v ulid.ULID) bool {
		trace = append(trace, v)
		return true
	})
	assert.Equal(t, []ulid.ULID{v2, v1}, trace)
}
