package node

import (
	"context"
	"testing"
	"time"

	"github.com/go-faker/faker/v4"
	"github.com/siyul-park/uniflow/object"
	"github.com/siyul-park/uniflow/packet"
	"github.com/siyul-park/uniflow/port"
	"github.com/siyul-park/uniflow/process"
	"github.com/stretchr/testify/assert"
)

func TestNewOneToOneNode(t *testing.T) {
	n := NewOneToOneNode(nil)
	assert.NotNil(t, n)
	assert.NoError(t, n.Close())
}

func TestOneToOneNode_Port(t *testing.T) {
	n := NewOneToOneNode(nil)
	defer n.Close()

	assert.NotNil(t, n.In(PortIn))
	assert.NotNil(t, n.Out(PortOut))
	assert.NotNil(t, n.Out(PortErr))
}

func TestOneToOneNode_SendAndReceive(t *testing.T) {
	t.Run("SingleInputToNoOutput", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
		defer cancel()

		n := NewOneToOneNode(func(_ *process.Process, inPck *packet.Packet) (*packet.Packet, *packet.Packet) {
			return inPck, nil
		})
		defer n.Close()

		in := port.NewOut()
		in.Link(n.In(PortIn))

		proc := process.New()
		defer proc.Exit(nil)

		inWriter := in.Open(proc)

		inPayload := object.NewString(faker.UUIDHyphenated())
		inPck := packet.New(inPayload)

		inWriter.Write(inPck)

		select {
		case <-inWriter.Receive():
		case <-ctx.Done():
			assert.Fail(t, "timeout")
		}
	})

	t.Run("SingleInputToSingleOutput", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
		defer cancel()

		n := NewOneToOneNode(func(_ *process.Process, inPck *packet.Packet) (*packet.Packet, *packet.Packet) {
			return inPck, nil
		})
		defer n.Close()

		in := port.NewOut()
		in.Link(n.In(PortIn))

		out := port.NewIn()
		n.Out(PortOut).Link(out)

		proc := process.New()
		defer proc.Exit(nil)

		inWriter := in.Open(proc)
		outReader := out.Open(proc)

		inPayload := object.NewString(faker.UUIDHyphenated())
		inPck := packet.New(inPayload)

		inWriter.Write(inPck)

		select {
		case outPck := <-outReader.Read():
			assert.Equal(t, inPayload.Interface(), outPck.Payload().Interface())
			outReader.Receive(outPck)
		case <-ctx.Done():
			assert.Fail(t, "timeout")
		}

		select {
		case backPck := <-inWriter.Receive():
			assert.NotNil(t, backPck)
		case <-ctx.Done():
			assert.Fail(t, "timeout")
		}
	})

	t.Run("SingleInputToSingleError", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
		defer cancel()

		n := NewOneToOneNode(func(_ *process.Process, inPck *packet.Packet) (*packet.Packet, *packet.Packet) {
			return nil, packet.New(object.NewString(faker.UUIDHyphenated()))
		})
		defer n.Close()

		in := port.NewOut()
		in.Link(n.In(PortIn))

		err := port.NewIn()
		n.Out(PortErr).Link(err)

		proc := process.New()
		defer proc.Exit(nil)

		inWriter := in.Open(proc)
		errReader := err.Open(proc)

		inPayload := object.NewString(faker.UUIDHyphenated())
		inPck := packet.New(inPayload)

		inWriter.Write(inPck)

		select {
		case outPck := <-errReader.Read():
			assert.NotNil(t, outPck)
			errReader.Receive(outPck)
		case <-ctx.Done():
			assert.Fail(t, "timeout")
		}

		select {
		case backPck := <-inWriter.Receive():
			assert.NotNil(t, backPck)
		case <-ctx.Done():
			assert.Fail(t, "timeout")
		}
	})
}

func BenchmarkOneToOneNode_SendAndReceive(b *testing.B) {
	n := NewOneToOneNode(func(_ *process.Process, inPck *packet.Packet) (*packet.Packet, *packet.Packet) {
		return inPck, nil
	})
	defer n.Close()

	in := port.NewOut()
	in.Link(n.In(PortIn))

	out := port.NewIn()
	n.Out(PortOut).Link(out)

	proc := process.New()
	defer proc.Exit(nil)

	inWriter := in.Open(proc)
	outReader := out.Open(proc)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		inPayload := object.NewString(faker.UUIDHyphenated())
		inPck := packet.New(inPayload)

		inWriter.Write(inPck)

		outPck := <-outReader.Read()
		outReader.Receive(outPck)

		<-inWriter.Receive()
	}
}