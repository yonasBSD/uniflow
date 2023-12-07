package controllx

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-faker/faker/v4"
	"github.com/siyul-park/uniflow/pkg/node"
	"github.com/siyul-park/uniflow/pkg/packet"
	"github.com/siyul-park/uniflow/pkg/port"
	"github.com/siyul-park/uniflow/pkg/primitive"
	"github.com/siyul-park/uniflow/pkg/process"
	"github.com/stretchr/testify/assert"
)

func TestNewSnippetNode(t *testing.T) {
	n, err := NewSnippetNode(LangJSON, "{}")
	assert.NoError(t, err)
	assert.NotNil(t, n)

	_ = n.Close()
}

func TestSnippetNode_Send(t *testing.T) {
	t.Run(LangTypescript, func(t *testing.T) {
		n, _ := NewSnippetNode(LangTypescript, `
function main(inPayload: any): any {
	return inPayload;
}
		`)
		defer func() { _ = n.Close() }()

		io := port.New()
		ioPort, _ := n.Port(node.PortIO)
		ioPort.Link(io)

		proc := process.New()
		defer proc.Exit(nil)

		ioStream := io.Open(proc)

		inPayload := primitive.NewString(faker.UUIDHyphenated())
		inPck := packet.New(inPayload)

		ioStream.Send(inPck)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		select {
		case outPck := <-ioStream.Receive():
			assert.Equal(t, inPayload, outPck.Payload())
		case <-ctx.Done():
			assert.Fail(t, "timeout")
		}
	})

	t.Run(LangJavascript, func(t *testing.T) {
		n, _ := NewSnippetNode(LangTypescript, `
function main(inPayload) {
	return inPayload;
}
		`)
		defer func() { _ = n.Close() }()

		io := port.New()
		ioPort, _ := n.Port(node.PortIO)
		ioPort.Link(io)

		proc := process.New()
		defer proc.Exit(nil)

		ioStream := io.Open(proc)

		inPayload := primitive.NewString(faker.UUIDHyphenated())
		inPck := packet.New(inPayload)

		ioStream.Send(inPck)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		select {
		case outPck := <-ioStream.Receive():
			assert.Equal(t, inPayload, outPck.Payload())
		case <-ctx.Done():
			assert.Fail(t, "timeout")
		}
	})

	t.Run(LangJSON, func(t *testing.T) {
		data := faker.UUIDHyphenated()

		n, _ := NewSnippetNode(LangJSON, fmt.Sprintf("\"%s\"", data))
		defer func() { _ = n.Close() }()

		io := port.New()
		ioPort, _ := n.Port(node.PortIO)
		ioPort.Link(io)

		proc := process.New()
		defer proc.Exit(nil)

		ioStream := io.Open(proc)

		inPayload := primitive.NewString(faker.UUIDHyphenated())
		inPck := packet.New(inPayload)

		ioStream.Send(inPck)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		select {
		case outPck := <-ioStream.Receive():
			assert.Equal(t, data, outPck.Payload().Interface())
		case <-ctx.Done():
			assert.Fail(t, "timeout")
		}
	})

	t.Run(LangJSONata, func(t *testing.T) {
		n, _ := NewSnippetNode(LangJSONata, "$")
		defer func() { _ = n.Close() }()

		io := port.New()
		ioPort, _ := n.Port(node.PortIO)
		ioPort.Link(io)

		proc := process.New()
		defer proc.Exit(nil)

		ioStream := io.Open(proc)

		inPayload := primitive.NewString(faker.UUIDHyphenated())
		inPck := packet.New(inPayload)

		ioStream.Send(inPck)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		select {
		case outPck := <-ioStream.Receive():
			assert.Equal(t, inPayload, outPck.Payload())
		case <-ctx.Done():
			assert.Fail(t, "timeout")
		}
	})
}

func BenchmarkSnippetNode_Send(b *testing.B) {
	b.Run(LangTypescript, func(b *testing.B) {
		n, _ := NewSnippetNode(LangTypescript, `
function main(inPayload: any): any {
	return inPayload;
}
		`)
		defer func() { _ = n.Close() }()

		io := port.New()
		ioPort, _ := n.Port(node.PortIO)
		ioPort.Link(io)

		proc := process.New()
		defer proc.Exit(nil)

		ioStream := io.Open(proc)

		inPayload := primitive.NewString(faker.UUIDHyphenated())
		inPck := packet.New(inPayload)

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			ioStream.Send(inPck)
			<-ioStream.Receive()
		}
	})

	b.Run(LangJavascript, func(b *testing.B) {
		n, _ := NewSnippetNode(LangJavascript, `
function main(inPayload) {
	return inPayload;
}
		`)
		defer func() { _ = n.Close() }()

		io := port.New()
		ioPort, _ := n.Port(node.PortIO)
		ioPort.Link(io)

		proc := process.New()
		defer proc.Exit(nil)

		ioStream := io.Open(proc)

		inPayload := primitive.NewString(faker.UUIDHyphenated())
		inPck := packet.New(inPayload)

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			ioStream.Send(inPck)
			<-ioStream.Receive()
		}
	})

	b.Run(LangJSON, func(b *testing.B) {
		n, _ := NewSnippetNode(LangJSON, fmt.Sprintf("\"%s\"", faker.UUIDHyphenated()))
		defer func() { _ = n.Close() }()

		io := port.New()
		ioPort, _ := n.Port(node.PortIO)
		ioPort.Link(io)

		proc := process.New()
		defer proc.Exit(nil)

		ioStream := io.Open(proc)

		inPayload := primitive.NewString(faker.UUIDHyphenated())
		inPck := packet.New(inPayload)

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			ioStream.Send(inPck)
			<-ioStream.Receive()
		}
	})

	b.Run(LangJSONata, func(b *testing.B) {
		n, _ := NewSnippetNode(LangJSONata, "$")
		defer func() { _ = n.Close() }()

		io := port.New()
		ioPort, _ := n.Port(node.PortIO)
		ioPort.Link(io)

		proc := process.New()
		defer proc.Exit(nil)

		ioStream := io.Open(proc)

		inPayload := primitive.NewString(faker.UUIDHyphenated())
		inPck := packet.New(inPayload)

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			ioStream.Send(inPck)
			<-ioStream.Receive()
		}
	})
}
