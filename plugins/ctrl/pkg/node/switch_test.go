package node

import (
	"context"
	"testing"
	"time"

	"github.com/siyul-park/uniflow/pkg/language/text"
	"github.com/siyul-park/uniflow/pkg/node"
	"github.com/siyul-park/uniflow/pkg/packet"
	"github.com/siyul-park/uniflow/pkg/port"
	"github.com/siyul-park/uniflow/pkg/process"
	"github.com/siyul-park/uniflow/pkg/types"
	"github.com/stretchr/testify/require"
)

func TestSwitchNodeCodec_Compile(t *testing.T) {
	codec := NewSwitchNodeCodec(text.NewCompiler())

	spec := &SwitchNodeSpec{
		Matches: []Condition{
			{
				When: "",
				Port: node.PortWithIndex(node.PortOut, 0),
			},
		},
	}

	n, err := codec.Compile(spec)
	require.NoError(t, err)
	require.NotNil(t, n)
	require.NoError(t, n.Close())
}

func TestNewSwitchNode(t *testing.T) {
	n := NewSwitchNode()
	require.NotNil(t, n)
	require.NoError(t, n.Close())
}

func TestSwitchNode_SendAndReceive(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
	defer cancel()

	n := NewSwitchNode()
	defer n.Close()

	n.Match(node.PortWithIndex(node.PortOut, 0), func(_ context.Context, _ any) (bool, error) { return true, nil })

	in := port.NewOut()
	in.Link(n.In(node.PortIn))

	out0 := port.NewIn()
	n.Out(node.PortWithIndex(node.PortOut, 0)).Link(out0)

	proc := process.New()
	defer proc.Exit(nil)

	inWriter := in.Open(proc)
	outReader0 := out0.Open(proc)

	inPayload := types.NewMap(types.NewString("foo"), types.NewString("bar"))
	inPck := packet.New(inPayload)

	inWriter.Write(inPck)

	select {
	case outPck := <-outReader0.Read():
		require.Equal(t, inPayload, outPck.Payload())
		outReader0.Receive(outPck)
	case <-ctx.Done():
		require.Fail(t, ctx.Err().Error())
	}

	select {
	case backPck := <-inWriter.Receive():
		require.NotNil(t, backPck)
	case <-ctx.Done():
		require.Fail(t, ctx.Err().Error())
	}
}

func BenchmarkSwitchNode_SendAndReceive(b *testing.B) {
	n := NewSwitchNode()
	defer n.Close()

	n.Match(node.PortWithIndex(node.PortOut, 0), func(_ context.Context, _ any) (bool, error) { return true, nil })

	in := port.NewOut()
	in.Link(n.In(node.PortIn))

	out0 := port.NewIn()
	n.Out(node.PortWithIndex(node.PortOut, 0)).Link(out0)

	proc := process.New()
	defer proc.Exit(nil)

	inWriter := in.Open(proc)
	outReader0 := out0.Open(proc)

	inPayload := types.NewMap(types.NewString("foo"), types.NewString("bar"))
	inPck := packet.New(inPayload)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		inWriter.Write(inPck)

		outPck := <-outReader0.Read()
		outReader0.Receive(outPck)

		<-inWriter.Receive()
	}
}
