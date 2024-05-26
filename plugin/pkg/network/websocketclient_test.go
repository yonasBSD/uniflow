package network

import (
	"context"
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/go-faker/faker/v4"
	"github.com/gorilla/websocket"
	"github.com/phayes/freeport"
	"github.com/siyul-park/uniflow/pkg/node"
	"github.com/siyul-park/uniflow/pkg/packet"
	"github.com/siyul-park/uniflow/pkg/port"
	"github.com/siyul-park/uniflow/pkg/primitive"
	"github.com/siyul-park/uniflow/pkg/process"
	"github.com/stretchr/testify/assert"
)

func TestNewWebSocketClient(t *testing.T) {
	n := NewWebSocketClientNode(&url.URL{})
	assert.NotNil(t, n)
	assert.NoError(t, n.Close())
}

func TestWebSocketClientNode_Port(t *testing.T) {
	n := NewWebSocketClientNode(&url.URL{})
	defer n.Close()

	assert.NotNil(t, n.In(node.PortIO))
	assert.NotNil(t, n.In(node.PortIn))
	assert.NotNil(t, n.Out(node.PortOut))
	assert.NotNil(t, n.Out(node.PortErr))
}

func TestWebSocketClientNode_SendAndReceive(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
	defer cancel()

	p, err := freeport.GetFreePort()
	assert.NoError(t, err)

	u, _ := url.Parse(fmt.Sprintf("ws://localhost:%d", p))

	http := NewHTTPServerNode(fmt.Sprintf(":%d", p))
	defer http.Close()

	ws := NewWebSocketUpgradeNode()
	defer ws.Close()

	client := NewWebSocketClientNode(u)
	defer client.Close()

	http.Out(node.PortOut).Link(ws.In(node.PortIO))
	ws.Out(node.PortOut).Link(ws.In(node.PortIn))

	assert.NoError(t, http.Listen())

	io := port.NewOut()
	io.Link(client.In(node.PortIO))

	in := port.NewOut()
	in.Link(client.In(node.PortIn))

	out := port.NewIn()
	client.Out(node.PortOut).Link(out)

	proc := process.New()
	defer proc.Exit(nil)

	ioWriter := io.Open(proc)
	inWriter := in.Open(proc)

	done := make(chan struct{})
	out.AddInitHook(port.InitHookFunc(func(proc *process.Process) {
		outReader := out.Open(proc)

		for {
			_, ok := <-outReader.Read()
			if !ok {
				return
			}

			outReader.Receive(packet.None)
			select {
			case <-done:
			default:
				close(done)
			}
		}
	}))

	var inPayload primitive.Value
	inPck := packet.New(inPayload)

	ioWriter.Write(inPck)

	select {
	case <-ioWriter.Receive():
	case <-ctx.Done():
		assert.Fail(t, ctx.Err().Error())
	}

	inPayload, _ = primitive.MarshalText(&WebSocketPayload{
		Type: websocket.TextMessage,
		Data: primitive.NewString(faker.UUIDHyphenated()),
	})
	inPck = packet.New(inPayload)

	inWriter.Write(inPck)

	select {
	case <-done:
	case <-ctx.Done():
		assert.Fail(t, ctx.Err().Error())
	}

	inPayload, _ = primitive.MarshalText(&WebSocketPayload{
		Type: websocket.CloseMessage,
	})
	inPck = packet.New(inPayload)

	inWriter.Write(inPck)

	select {
	case <-inWriter.Receive():
	case <-ctx.Done():
		assert.Fail(t, ctx.Err().Error())
	}
}

func TestWebSocketClientNodeCodec_Decode(t *testing.T) {
	codec := NewWebSocketClientNodeCodec()

	spec := &WebSocketClientNodeSpec{
		URL: "ws://localhost:8080/",
	}
	n, err := codec.Decode(spec)
	assert.NoError(t, err)
	assert.NotNil(t, n)
	assert.NoError(t, n.Close())
}

func BenchmarkWebSocketClientNode_SendAndReceive(b *testing.B) {
	p, _ := freeport.GetFreePort()

	u, _ := url.Parse(fmt.Sprintf("ws://localhost:%d", p))

	http := NewHTTPServerNode(fmt.Sprintf(":%d", p))
	defer http.Close()

	ws := NewWebSocketUpgradeNode()
	defer ws.Close()

	client := NewWebSocketClientNode(u)
	defer client.Close()

	http.Out(node.PortOut).Link(ws.In(node.PortIO))
	ws.Out(node.PortOut).Link(ws.In(node.PortIn))

	http.Listen()

	io := port.NewOut()
	io.Link(client.In(node.PortIO))

	in := port.NewOut()
	in.Link(client.In(node.PortIn))

	out := port.NewIn()
	client.Out(node.PortOut).Link(out)

	proc := process.New()
	defer proc.Exit(nil)

	ioWriter := io.Open(proc)
	inWriter := in.Open(proc)
	outReader := out.Open(proc)

	var inPayload primitive.Value
	inPck := packet.New(inPayload)

	ioWriter.Write(inPck)

	inPayload, _ = primitive.MarshalText(&WebSocketPayload{
		Type: websocket.TextMessage,
		Data: primitive.NewString(faker.UUIDHyphenated()),
	})
	inPck = packet.New(inPayload)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		inWriter.Write(inPck)
		outPck := <-outReader.Read()
		outReader.Receive(outPck)
	}

	inPayload, _ = primitive.MarshalText(&WebSocketPayload{
		Type: websocket.CloseMessage,
	})
	inPck = packet.New(inPayload)

	inWriter.Write(inPck)
}
