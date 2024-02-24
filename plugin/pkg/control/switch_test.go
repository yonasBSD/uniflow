package control

import (
	"context"
	"testing"
	"time"

	"github.com/siyul-park/uniflow/pkg/node"
	"github.com/siyul-park/uniflow/pkg/packet"
	"github.com/siyul-park/uniflow/pkg/port"
	"github.com/siyul-park/uniflow/pkg/primitive"
	"github.com/siyul-park/uniflow/pkg/process"
	"github.com/siyul-park/uniflow/plugin/internal/language"
	"github.com/stretchr/testify/assert"
)

func TestNewSwitchNode(t *testing.T) {
	n := NewSwitchNode(language.JSONata)
	assert.NotNil(t, n)

	assert.NoError(t, n.Close())
}

func TestSwitchNode_Add(t *testing.T) {
	t.Run(language.Typescript, func(t *testing.T) {
		n := NewSwitchNode(language.Typescript)
		defer n.Close()

		err := n.Add("$.foo === \"bar\"", node.MultiPort(node.PortOut, 0))
		assert.NoError(t, err)
	})

	t.Run(language.Javascript, func(t *testing.T) {
		n := NewSwitchNode(language.Javascript)
		defer n.Close()

		err := n.Add("$.foo === \"bar\"", node.MultiPort(node.PortOut, 0))
		assert.NoError(t, err)
	})

	t.Run(language.JSONata, func(t *testing.T) {
		n := NewSwitchNode(language.JSONata)
		defer n.Close()

		err := n.Add("$.foo = \"bar\"", node.MultiPort(node.PortOut, 0))
		assert.NoError(t, err)
	})
}

func TestSwitchNode_SendAndReceive(t *testing.T) {
	t.Run(language.Typescript, func(t *testing.T) {
		n := NewSwitchNode(language.Typescript)
		defer n.Close()

		_ = n.Add("$.foo === \"bar\"", node.MultiPort(node.PortOut, 0))

		in := port.NewOut()
		in.Link(n.In(node.PortIn))

		out0 := port.NewIn()
		n.Out(node.MultiPort(node.PortOut, 0)).Link(out0)

		proc := process.New()
		defer proc.Exit(nil)

		inWriter := in.Open(proc)
		outReader0 := out0.Open(proc)

		inPayload := primitive.NewMap(primitive.NewString("foo"), primitive.NewString("bar"))
		inPck := packet.New(inPayload)

		inWriter.Write(inPck)

		ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
		defer cancel()

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
	})

	t.Run(language.Javascript, func(t *testing.T) {
		n := NewSwitchNode(language.Javascript)
		defer n.Close()

		_ = n.Add("$.foo === \"bar\"", node.MultiPort(node.PortOut, 0))

		in := port.NewOut()
		in.Link(n.In(node.PortIn))

		out0 := port.NewIn()
		n.Out(node.MultiPort(node.PortOut, 0)).Link(out0)

		proc := process.New()
		defer proc.Exit(nil)

		inWriter := in.Open(proc)
		outReader0 := out0.Open(proc)

		inPayload := primitive.NewMap(primitive.NewString("foo"), primitive.NewString("bar"))
		inPck := packet.New(inPayload)

		inWriter.Write(inPck)

		ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
		defer cancel()

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
	})

	t.Run(language.JSONata, func(t *testing.T) {
		n := NewSwitchNode(language.JSONata)
		defer n.Close()

		_ = n.Add("$.foo = \"bar\"", node.MultiPort(node.PortOut, 0))

		in := port.NewOut()
		in.Link(n.In(node.PortIn))

		out0 := port.NewIn()
		n.Out(node.MultiPort(node.PortOut, 0)).Link(out0)

		proc := process.New()
		defer proc.Exit(nil)

		inWriter := in.Open(proc)
		outReader0 := out0.Open(proc)

		inPayload := primitive.NewMap(primitive.NewString("foo"), primitive.NewString("bar"))
		inPck := packet.New(inPayload)

		inWriter.Write(inPck)

		ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
		defer cancel()

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
	})
}

func TestSwitchNodeCodec_Decode(t *testing.T) {
	codec := NewSwitchNodeCodec()

	spec := &SwitchNodeSpec{
		Lang: language.JSONata,
		Match: []Condition{
			{
				When: "$.foo = \"bar\"",
				Port: node.MultiPort(node.PortOut, 0),
			},
		},
	}

	n, err := codec.Decode(spec)
	assert.NoError(t, err)
	assert.NotNil(t, n)

	assert.NoError(t, n.Close())
}

func BenchmarkSwitchNode_SendAndReceive(b *testing.B) {
	b.Run(language.Typescript, func(b *testing.B) {
		n := NewSwitchNode(language.Typescript)
		defer n.Close()

		_ = n.Add("$.foo === \"bar\"", node.MultiPort(node.PortOut, 0))

		in := port.NewOut()
		in.Link(n.In(node.PortIn))

		out0 := port.NewIn()
		n.Out(node.MultiPort(node.PortOut, 0)).Link(out0)

		proc := process.New()
		defer proc.Exit(nil)

		inWriter := in.Open(proc)
		outReader0 := out0.Open(proc)

		inPayload := primitive.NewMap(primitive.NewString("foo"), primitive.NewString("bar"))
		inPck := packet.New(inPayload)

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			inWriter.Write(inPck)
			outPck := <-outReader0.Read()
			outReader0.Receive(outPck)
			<-inWriter.Receive()
		}
	})

	b.Run(language.Javascript, func(b *testing.B) {
		n := NewSwitchNode(language.Javascript)
		defer n.Close()

		_ = n.Add("$.foo === \"bar\"", node.MultiPort(node.PortOut, 0))

		in := port.NewOut()
		in.Link(n.In(node.PortIn))

		out0 := port.NewIn()
		n.Out(node.MultiPort(node.PortOut, 0)).Link(out0)

		proc := process.New()
		defer proc.Exit(nil)

		inWriter := in.Open(proc)
		outReader0 := out0.Open(proc)

		inPayload := primitive.NewMap(primitive.NewString("foo"), primitive.NewString("bar"))
		inPck := packet.New(inPayload)

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			inWriter.Write(inPck)
			outPck := <-outReader0.Read()
			outReader0.Receive(outPck)
			<-inWriter.Receive()
		}
	})

	b.Run(language.JSONata, func(b *testing.B) {
		n := NewSwitchNode(language.Javascript)
		defer n.Close()

		_ = n.Add("$.foo === \"bar\"", node.MultiPort(node.PortOut, 0))

		in := port.NewOut()
		in.Link(n.In(node.PortIn))

		out0 := port.NewIn()
		n.Out(node.MultiPort(node.PortOut, 0)).Link(out0)

		proc := process.New()
		defer proc.Exit(nil)

		inWriter := in.Open(proc)
		outReader0 := out0.Open(proc)

		inPayload := primitive.NewMap(primitive.NewString("foo"), primitive.NewString("bar"))
		inPck := packet.New(inPayload)

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			inWriter.Write(inPck)
			outPck := <-outReader0.Read()
			outReader0.Receive(outPck)
			<-inWriter.Receive()
		}
	})
}
