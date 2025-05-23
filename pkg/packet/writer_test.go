package packet

import (
	"testing"

	"github.com/go-faker/faker/v4"
	"github.com/stretchr/testify/require"

	"github.com/siyul-park/uniflow/pkg/types"
)

func TestSend(t *testing.T) {
	w := NewWriter()
	defer w.Close()

	r := NewReader()
	defer r.Close()

	w.Link(r)

	go func() {
		for {
			inPck, ok := <-r.Read()
			if !ok {
				return
			}
			r.Receive(inPck)
		}
	}()

	outPck := New(types.NewString(faker.UUIDHyphenated()))

	backPck := Send(w, outPck)
	require.Equal(t, outPck.Payload(), backPck.Payload())
}

func TestSendOrFallback(t *testing.T) {
	t.Run("Call", func(t *testing.T) {
		w := NewWriter()
		defer w.Close()

		r := NewReader()
		defer r.Close()

		w.Link(r)

		go func() {
			for {
				inPck, ok := <-r.Read()
				if !ok {
					return
				}
				r.Receive(inPck)
			}
		}()

		outPck := New(types.NewString(faker.UUIDHyphenated()))

		backPck := SendOrFallback(w, outPck, None)
		require.Equal(t, outPck.Payload(), backPck.Payload())
	})

	t.Run("Return", func(t *testing.T) {
		w := NewWriter()
		defer w.Close()

		outPck := New(nil)

		backPck := SendOrFallback(w, outPck, None)
		require.Equal(t, None, backPck)
	})
}

func TestWriter_AddHook(t *testing.T) {
	w := NewWriter()
	defer w.Close()

	r := NewReader()
	defer r.Close()

	count := 0
	w.AddInboundHook(HookFunc(func(_ *Packet) {
		count += 1
	}))
	w.AddOutboundHook(HookFunc(func(_ *Packet) {
		count += 1
	}))

	w.Link(r)

	out := New(nil)

	w.Write(out)
	require.Equal(t, 1, count)

	in := <-r.Read()

	r.Receive(in)

	back, ok := <-w.Receive()
	require.True(t, ok)
	require.Equal(t, in, back)
	require.Equal(t, 2, count)
}

func TestWriter_Link(t *testing.T) {
	w := NewWriter()
	defer w.Close()

	r := NewReader()
	defer r.Close()

	ok := w.Link(r)
	require.True(t, ok)
	require.Len(t, w.Links(), 1)

	ok = w.Link(r)
	require.False(t, ok)
}

func TestWriter_Unlink(t *testing.T) {
	w := NewWriter()
	defer w.Close()

	r := NewReader()
	defer r.Close()

	w.Link(r)

	pck1 := New(types.NewString(faker.UUIDHyphenated()))

	w.Write(pck1)

	pck2, ok := <-r.Read()
	require.True(t, ok)
	require.Equal(t, pck1.Payload(), pck2.Payload())

	ok = w.Unlink(r)
	require.True(t, ok)
	require.Len(t, w.Links(), 0)

	pck3, ok := <-w.Receive()
	require.True(t, ok)
	require.Equal(t, ErrDroppedPacket, pck3.Payload())

	ok = r.Receive(None)
	require.False(t, ok)

	ok = w.Unlink(r)
	require.False(t, ok)
}

func TestWriter_Write(t *testing.T) {
	w := NewWriter()
	defer w.Close()

	r := NewReader()
	defer r.Close()

	w.Link(r)

	pck1 := New(types.NewString(faker.UUIDHyphenated()))
	pck2 := New(types.NewString(faker.UUIDHyphenated()))

	count := w.Write(pck1)
	require.Equal(t, 1, count)

	count = w.Write(pck2)
	require.Equal(t, 1, count)

	pck3, ok := <-r.Read()
	require.True(t, ok)
	require.Equal(t, pck1.Payload(), pck3.Payload())

	pck4, ok := <-r.Read()
	require.True(t, ok)
	require.Equal(t, pck2.Payload(), pck4.Payload())
}

func BenchmarkWriter_Write(b *testing.B) {
	w := NewWriter()
	defer w.Close()

	r := NewReader()
	defer r.Close()

	w.Link(r)

	b.RunParallel(func(p *testing.PB) {
		out := New(nil)

		for p.Next() {
			count := w.Write(out)
			require.Equal(b, 1, count)

			_, ok := <-r.Read()
			require.True(b, ok)
		}
	})
}
