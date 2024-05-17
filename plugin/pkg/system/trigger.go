package system

import (
	"sync"

	"github.com/siyul-park/uniflow/pkg/event"
	"github.com/siyul-park/uniflow/pkg/node"
	"github.com/siyul-park/uniflow/pkg/packet"
	"github.com/siyul-park/uniflow/pkg/port"
	"github.com/siyul-park/uniflow/pkg/primitive"
	"github.com/siyul-park/uniflow/pkg/process"
	"github.com/siyul-park/uniflow/pkg/scheme"
)

// TriggerNode represents a node that triggers events.
type TriggerNode struct {
	consumer *event.Consumer
	done     chan struct{}
	inPort   *port.InPort
	outPort  *port.OutPort
	errPort  *port.OutPort
	mu       sync.RWMutex
}

// TriggerNodeSpec holds the specifications for creating a TriggerNode.
type TriggerNodeSpec struct {
	scheme.SpecMeta `map:",inline"`
	Topic           string `map:"topic"`
}

const KindTrigger = "trigger"

const (
	TopicLoad   = "load"
	TopicUnload = "unload"
)

// NewTriggerNode creates a new TriggerNode instance.
func NewTriggerNode(consumer *event.Consumer) *TriggerNode {
	n := &TriggerNode{
		consumer: consumer,
		done:     make(chan struct{}),
		inPort:   port.NewIn(),
		outPort:  port.NewOut(),
		errPort:  port.NewOut(),
	}

	n.inPort.AddHandler(port.HandlerFunc(n.forward))

	return n
}

// In returns the input port with the specified name.
func (n *TriggerNode) In(name string) *port.InPort {
	n.mu.RLock()
	defer n.mu.RUnlock()

	switch name {
	case node.PortIn:
		return n.inPort
	default:
	}

	return nil
}

// Out returns the output port with the specified name.
func (n *TriggerNode) Out(name string) *port.OutPort {
	n.mu.RLock()
	defer n.mu.RUnlock()

	switch name {
	case node.PortOut:
		return n.outPort
	case node.PortErr:
		return n.errPort
	default:
	}

	return nil
}

// Listen listens for incoming events and triggers processing.
func (n *TriggerNode) Listen() {
	n.mu.Lock()
	defer n.mu.Unlock()

	select {
	case <-n.done:
		n.done = make(chan struct{})
	default:
	}

	done := n.done
	go func() {
		for {
			var e *event.Event
			var ok bool
			select {
			case e, ok = <-n.consumer.Consume():
			case <-done:
			}
			if !ok {
				return
			}

			proc := process.New()

			outWriter := n.outPort.Open(proc)
			errWriter := n.errPort.Open(proc)

			port.Discard(outWriter)
			port.Discard(errWriter)

			if outPayload, err := primitive.MarshalText(e.Data()); err != nil {
				errPck := packet.WithError(err, nil)
				errWriter.Write(errPck)
			} else {
				outPck := packet.New(outPayload)
				outWriter.Write(outPck)
			}

			go func() {
				proc.Close()
				e.Close()
			}()
		}
	}()
}

// Shutdown shuts down the trigger node.
func (n *TriggerNode) Shutdown() {
	n.mu.Lock()
	defer n.mu.Unlock()

	select {
	case <-n.done:
	default:
		close(n.done)
	}
}

// Close closes all ports associated with the node.
func (n *TriggerNode) Close() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	select {
	case <-n.done:
	default:
		close(n.done)
	}

	n.consumer.Close()
	n.inPort.Close()
	n.outPort.Close()
	n.errPort.Close()

	for e := range n.consumer.Consume() {
		e.Close()
	}

	return nil
}

func (n *TriggerNode) forward(proc *process.Process) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	inReader := n.inPort.Open(proc)

	for {
		inPck, ok := <-inReader.Read()
		if !ok {
			return
		}

		proc.Stack().Clear(inPck)
	}
}

// NewTriggerNodeCodec creates a new codec for TriggerNodeSpec.
func NewTriggerNodeCodec(broker *event.Broker) scheme.Codec {
	return scheme.CodecWithType(func(spec *TriggerNodeSpec) (node.Node, error) {
		c := broker.Consumer(spec.Topic)
		n := NewTriggerNode(c)
		return n, nil
	})
}