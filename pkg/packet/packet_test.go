package packet

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/siyul-park/uniflow/pkg/types"
)

func TestNew(t *testing.T) {
	pck := New(nil)
	require.NotZero(t, pck.ID())
	require.NotNil(t, pck)
}

func TestJoin(t *testing.T) {
	t.Run("None", func(t *testing.T) {
		res := Join(None, None)
		require.Equal(t, None, res)
	})

	t.Run("Zero", func(t *testing.T) {
		res := Join()
		require.Equal(t, None, res)
	})

	t.Run("One", func(t *testing.T) {
		pck := New(nil)
		res := Join(pck)
		require.Equal(t, pck, res)
	})

	t.Run("Many", func(t *testing.T) {
		pck1 := New(nil)
		pck2 := New(nil)
		res := Join(pck1, pck2)
		require.Equal(t, types.NewSlice(nil, nil), res.Payload())
	})
}
