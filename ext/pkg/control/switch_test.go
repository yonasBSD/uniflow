package control

import (
	"context"
	"testing"
	"time"

	"github.com/siyul-park/uniflow/ext/pkg/language/text"
	"github.com/siyul-park/uniflow/pkg/node"
	"github.com/siyul-park/uniflow/pkg/packet"
	"github.com/siyul-park/uniflow/pkg/port"
	"github.com/siyul-park/uniflow/pkg/process"
	"github.com/siyul-park/uniflow/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestNewSwitchNode(t *testing.T) {
	n := NewSwitchNode()
	assert.NotNil(t, n)
	assert.NoError(t, n.Close())
}

func TestSwitchNode_SendAndReceive(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
	defer cancel()

	n := NewSwitchNode()
	defer n.Close()

	n.Match(func(_ any) (bool, error) { return true, nil }, node.PortWithIndex(node.PortOut, 0))

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
		assert.Equal(t, inPayload, outPck.Payload())
		outReader0.Receive(outPck)
	case <-ctx.Done():
		assert.Fail(t, ctx.Err().Error())
	}

	select {
	case backPck := <-inWriter.Receive():
		assert.NotNil(t, backPck)
	case <-ctx.Done():
		assert.Fail(t, "timeout")
	}
}

func TestSwitchNodeCodec_Decode(t *testing.T) {
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
	assert.NoError(t, err)
	assert.NotNil(t, n)
	assert.NoError(t, n.Close())
}

func BenchmarkSwitchNode_SendAndReceive(b *testing.B) {
	n := NewSwitchNode()
	defer n.Close()

	n.Match(func(_ any) (bool, error) { return true, nil }, node.PortWithIndex(node.PortOut, 0))

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
