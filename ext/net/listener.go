package net

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"sync"

	"github.com/pkg/errors"
	"github.com/siyul-park/uniflow/node"
	"github.com/siyul-park/uniflow/object"
	"github.com/siyul-park/uniflow/packet"
	"github.com/siyul-park/uniflow/port"
	"github.com/siyul-park/uniflow/process"
	"github.com/siyul-park/uniflow/scheme"
	"github.com/siyul-park/uniflow/spec"
)

// HTTPListenerNode represents a Node for handling HTTP requests.
type HTTPListenerNode struct {
	server   *http.Server
	listener net.Listener
	inPort   *port.InPort
	outPort  *port.OutPort
	errPort  *port.OutPort
	mu       sync.RWMutex
}

// ListenerNodeSpec holds the specifications for creating a ListenerNode.
type ListenerNodeSpec struct {
	spec.Meta `map:",inline"`
	Protocol  string `map:"protocol"`
	Port      int    `map:"port"`
}

const KindListener = "listener"

const KeyHTTPRequest = "http.Request"
const KeyHTTPResponseWriter = "http.ResponseWriter"

var _ node.Node = (*HTTPListenerNode)(nil)
var _ http.Handler = (*HTTPListenerNode)(nil)

// NewHTTPListenerNode creates a new HTTPListenerNode with the specified address.
func NewHTTPListenerNode(address string) *HTTPListenerNode {
	n := &HTTPListenerNode{
		inPort:  port.NewIn(),
		outPort: port.NewOut(),
		errPort: port.NewOut(),
	}

	n.inPort.AddInitHook(port.InitHookFunc(n.forward))

	s := new(http.Server)
	s.Addr = address
	s.Handler = n
	n.server = s

	return n
}

// In returns the input port with the specified name.
func (n *HTTPListenerNode) In(name string) *port.InPort {
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
func (n *HTTPListenerNode) Out(name string) *port.OutPort {
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

// Address returns the listener address if available.
func (n *HTTPListenerNode) Address() net.Addr {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.listener == nil {
		return nil
	}
	return n.listener.Addr()
}

// Listen starts the HTTP server.
func (n *HTTPListenerNode) Listen() error {
	if err := func() error {
		n.mu.Lock()
		defer n.mu.Unlock()

		if n.listener != nil {
			return nil
		}
		if l, err := newTCPKeepAliveListener(n.server.Addr, "tcp"); err != nil {
			return err
		} else {
			n.listener = l
		}
		return nil
	}(); err != nil {
		return err
	}

	go n.server.Serve(n.listener)

	return nil
}

// Shutdown shuts down the HTTPListenerNode by closing the server and its associated listener.
// It locks the mutex to ensure safe concurrent access to the server and listener.
// If an error occurs during the shutdown process, it returns the error.
func (n *HTTPListenerNode) Shutdown() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.listener != nil {
		return nil
	}

	if err := n.server.Close(); err != nil {
		return err
	}
	s := new(http.Server)
	s.Addr = n.server.Addr
	n.server = s

	_ = n.listener.Close()
	n.listener = nil

	return nil
}

// ServeHTTP handles HTTP requests.
func (n *HTTPListenerNode) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	proc := process.New()
	ctx := r.Context()

	go func() {
		<-ctx.Done()
		proc.Wait()
		proc.Exit(ctx.Err())
	}()

	proc.Data().Store(KeyHTTPResponseWriter, w)
	proc.Data().Store(KeyHTTPRequest, r)

	outWriter := n.outPort.Open(proc)
	errWriter := n.errPort.Open(proc)

	var outPck *packet.Packet
	var errPck *packet.Packet
	req, err := n.read(r)
	if err != nil {
		errPck = packet.New(object.NewError(err))
	} else if outPayload, err := object.MarshalText(req); err != nil {
		errPck = packet.New(object.NewError(err))
	} else {
		outPck = packet.New(outPayload)
	}

	var backPck *packet.Packet
	if errPck != nil {
		backPck = packet.Call(errWriter, errPck)
	} else {
		backPck = packet.Call(outWriter, outPck)
		if _, ok := backPck.Payload().(object.Error); ok {
			backPck = packet.CallOrFallback(errWriter, backPck, backPck)
		}
	}

	err = nil
	if backPck != packet.None {
		var res *HTTPPayload
		if _, ok := backPck.Payload().(object.Error); ok {
			res = NewHTTPPayload(http.StatusInternalServerError)
		} else if err := object.Unmarshal(backPck.Payload(), &res); err != nil {
			res.Body = backPck.Payload()
		}

		if res.Status >= 400 && res.Status < 600 {
			err = errors.New(http.StatusText(res.Status))
		}

		if w, ok := proc.Data().LoadAndDelete(KeyHTTPResponseWriter).(http.ResponseWriter); ok {
			n.negotiate(req, res)
			_ = n.write(w, res)
		}
	}

	go func() {
		proc.Wait()
		proc.Exit(err)
	}()
}

// Close closes all ports and stops the HTTP server.
func (n *HTTPListenerNode) Close() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.inPort.Close()
	n.outPort.Close()
	n.errPort.Close()

	return n.server.Close()
}

func (n *HTTPListenerNode) forward(proc *process.Process) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	inReader := n.inPort.Open(proc)

	for {
		inPck, ok := <-inReader.Read()
		if !ok {
			return
		}

		var res *HTTPPayload
		if _, ok := inPck.Payload().(object.Error); ok {
			res = NewHTTPPayload(http.StatusInternalServerError)
		} else if err := object.Unmarshal(inPck.Payload(), &res); err != nil {
			res.Body = inPck.Payload()
		}

		var err error
		if res.Status >= 400 && res.Status < 600 {
			err = errors.New(http.StatusText(res.Status))
		}

		w, ok1 := proc.Data().LoadAndDelete(KeyHTTPResponseWriter).(http.ResponseWriter)
		r, ok2 := proc.Data().Load(KeyHTTPRequest).(*http.Request)
		if ok1 && ok2 {
			req, _ := n.read(r)

			n.negotiate(req, res)
			_ = n.write(w, res)
		}

		if err == nil {
			inReader.Receive(packet.None)
		} else {
			inReader.Receive(packet.New(object.NewError(err)))
		}
	}
}

func (n *HTTPListenerNode) negotiate(req *HTTPPayload, res *HTTPPayload) {
	if res.Header == nil {
		res.Header = http.Header{}
	}
	if res.Header.Get(HeaderContentEncoding) == "" {
		acceptEncoding := req.Header.Get(HeaderAcceptEncoding)
		res.Header.Set(HeaderContentEncoding, Negotiate(acceptEncoding, []string{EncodingIdentity, EncodingGzip, EncodingDeflate, EncodingBr}))
	}
	if res.Header.Get(HeaderContentType) == "" {
		accept := req.Header.Get(HeaderAccept)
		res.Header.Set(HeaderContentType, Negotiate(accept, []string{ApplicationJSON, ApplicationForm, ApplicationOctetStream, TextPlain, MultipartFormData}))
	}
}

func (n *HTTPListenerNode) read(r *http.Request) (*HTTPPayload, error) {
	contentType := r.Header.Get(HeaderContentType)
	contentEncoding := r.Header.Get(HeaderContentEncoding)

	if b, err := io.ReadAll(r.Body); err != nil {
		return nil, err
	} else if b, err := Decompress(b, contentEncoding); err != nil {
		return nil, err
	} else if b, err := UnmarshalMIME(b, &contentType); err != nil {
		return nil, err
	} else {
		r.Header.Set(HeaderContentType, contentType)
		return &HTTPPayload{
			Method: r.Method,
			Scheme: r.URL.Scheme,
			Host:   r.Host,
			Path:   r.URL.Path,
			Query:  r.URL.Query(),
			Proto:  r.Proto,
			Header: r.Header,
			Body:   b,
		}, nil
	}
}

func (n *HTTPListenerNode) write(w http.ResponseWriter, res *HTTPPayload) error {
	if res == nil {
		return nil
	}

	contentType := res.Header.Get(HeaderContentType)
	contentEncoding := res.Header.Get(HeaderContentEncoding)

	b, err := MarshalMIME(res.Body, &contentType)
	if err != nil {
		return err
	}
	b, err = Compress(b, contentEncoding)
	if err != nil {
		return err
	}

	if res.Header == nil {
		res.Header = http.Header{}
	}
	res.Header.Set(HeaderContentType, contentType)
	for key := range w.Header() {
		w.Header().Del(key)
	}
	for key, headers := range res.Header {
		if !IsResponseHeader(key) {
			continue
		}
		for _, header := range headers {
			w.Header().Add(key, header)
		}
	}
	w.Header().Set(HeaderContentLength, strconv.Itoa(len(b)))
	w.Header().Set(HeaderContentType, contentType)

	status := res.Status
	if status == 0 {
		status = http.StatusOK
	}
	w.WriteHeader(status)

	if _, err := w.Write(b); err != nil {
		return err
	}
	return nil
}

// NewListenerNodeCodec creates a new codec for ListenerNodeSpec.
func NewListenerNodeCodec() scheme.Codec {
	return scheme.CodecWithType(func(spec *ListenerNodeSpec) (node.Node, error) {
		switch spec.Protocol {
		case ProtocolHTTP:
			return NewHTTPListenerNode(fmt.Sprintf(":%d", spec.Port)), nil
		}
		return nil, errors.WithStack(ErrInvalidProtocol)
	})
}