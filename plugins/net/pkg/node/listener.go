package node

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/textproto"
	"sync"

	"github.com/pkg/errors"
	"github.com/siyul-park/uniflow/pkg/node"
	"github.com/siyul-park/uniflow/pkg/packet"
	"github.com/siyul-park/uniflow/pkg/port"
	"github.com/siyul-park/uniflow/pkg/process"
	"github.com/siyul-park/uniflow/pkg/scheme"
	"github.com/siyul-park/uniflow/pkg/spec"
	"github.com/siyul-park/uniflow/pkg/types"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/siyul-park/uniflow/plugins/net/pkg/mime"
)

// ListenNodeSpec defines the specifications for creating a ListenNode.
type ListenNodeSpec struct {
	spec.Meta `json:",inline"`
	Protocol  string `json:"protocol" validate:"required"`
	Host      string `json:"host,omitempty" validate:"omitempty,hostname|ip"`
	Port      int    `json:"port" validate:"required"`
	TLS       TLS    `json:"tls"`
}

type TLS struct {
	Cert []byte `json:"cert,omitempty"`
	Key  []byte `json:"key,omitempty"`
}

// HTTPListenNode represents a Node for handling HTTP requests.
type HTTPListenNode struct {
	server   *http.Server
	listener net.Listener
	outPort  *port.OutPort
	errPort  *port.OutPort
	mu       sync.RWMutex
}

const KindListener = "listener"

const (
	KeyHTTPRequest        = "__http.Request__"
	KeyHTTPResponseWriter = "__http.ResponseWriter__"
)

var (
	_ node.Node    = (*HTTPListenNode)(nil)
	_ http.Handler = (*HTTPListenNode)(nil)
)

// NewListenNodeCodec creates a new codec for ListenNodeSpec.
func NewListenNodeCodec() scheme.Codec {
	return scheme.CodecWithType(func(spec *ListenNodeSpec) (node.Node, error) {
		switch spec.Protocol {
		case ProtocolHTTP:
			n := NewHTTPListenNode(fmt.Sprintf("%s:%d", spec.Host, spec.Port))
			if len(spec.TLS.Cert) > 0 || len(spec.TLS.Key) > 0 {
				if err := n.TLS(spec.TLS.Cert, spec.TLS.Key); err != nil {
					_ = n.Close()
					return nil, err
				}
			}
			return n, nil
		}
		return nil, errors.WithStack(ErrInvalidProtocol)
	})
}

// NewHTTPListenNode creates a new HTTPListenNode with the specified address.
func NewHTTPListenNode(address string) *HTTPListenNode {
	n := &HTTPListenNode{
		outPort: port.NewOut(),
		errPort: port.NewOut(),
	}
	n.server = &http.Server{
		Addr:    address,
		Handler: n,
	}
	return n
}

// In returns the input port with the specified name.
func (n *HTTPListenNode) In(_ string) *port.InPort {
	return nil
}

// Out returns the output port with the specified name.
func (n *HTTPListenNode) Out(name string) *port.OutPort {
	n.mu.RLock()
	defer n.mu.RUnlock()

	switch name {
	case node.PortOut:
		return n.outPort
	case node.PortError:
		return n.errPort
	default:
		return nil
	}
}

// Address returns the listener address if available.
func (n *HTTPListenNode) Address() net.Addr {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.listener == nil {
		return nil
	}
	return n.listener.Addr()
}

// TLS configures the HTTP server to use TLS with the provided certificate and key.
func (n *HTTPListenNode) TLS(cert, key []byte) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	certificate, err := tls.X509KeyPair(cert, key)
	if err != nil {
		return err
	}

	s := n.server
	s.TLSConfig = new(tls.Config)
	s.TLSConfig.Certificates = []tls.Certificate{certificate}
	s.TLSConfig.NextProtos = append(s.TLSConfig.NextProtos, "h2")

	return nil
}

// Listen starts the HTTP server.
func (n *HTTPListenNode) Listen() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.listener != nil {
		return nil
	}

	listener, err := net.Listen("tcp", n.server.Addr)
	if err != nil {
		return err
	}

	if n.server.TLSConfig != nil {
		listener = tls.NewListener(listener, n.server.TLSConfig)
	} else if n.server.Handler == n {
		h2s := &http2.Server{}
		n.server.Handler = h2c.NewHandler(n, h2s)
	}

	n.listener = listener

	go n.server.Serve(n.listener)
	return nil
}

// Shutdown shuts down the HTTPListenNode by closing the server and its associated listener.
func (n *HTTPListenNode) Shutdown() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.listener == nil {
		return nil
	}

	err := n.server.Close()

	n.server = &http.Server{
		Addr:    n.server.Addr,
		Handler: n.server.Handler,
	}
	n.listener = nil

	return err
}

// ServeHTTP handles HTTP requests.
func (n *HTTPListenNode) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	proc := process.New()

	proc.SetValue(KeyHTTPResponseWriter, w)
	proc.SetValue(KeyHTTPRequest, r)

	outWriter := n.outPort.Open(proc)
	errWriter := n.errPort.Open(proc)

	var outPck, errPck *packet.Packet
	req, err := n.read(r)
	if err != nil {
		errPck = packet.New(types.NewError(err))
	} else if outPayload, err := types.Marshal(req); err != nil {
		errPck = packet.New(types.NewError(err))
	} else {
		outPck = packet.New(outPayload)
	}

	var backPck *packet.Packet
	if errPck != nil {
		backPck = packet.Send(errWriter, errPck)
	} else {
		backPck = packet.Send(outWriter, outPck)
		if _, ok := backPck.Payload().(types.Error); ok {
			backPck = packet.SendOrFallback(errWriter, backPck, backPck)
		}
	}

	if backPck != packet.None {
		var res *HTTPPayload
		if _, ok := backPck.Payload().(types.Error); ok {
			res = NewHTTPPayload(http.StatusInternalServerError)
		} else if err := types.Unmarshal(backPck.Payload(), &res); err != nil {
			res.Body = backPck.Payload()
		}

		if res.Status >= 400 && res.Status < 600 {
			err = errors.New(http.StatusText(res.Status))
		}

		if w, ok := proc.RemoveValue(KeyHTTPResponseWriter).(http.ResponseWriter); ok {
			n.negotiate(req, res)
			_ = n.write(w, res)
		}
	}

	go func() {
		proc.Join()
		proc.Exit(err)
	}()
}

// Close closes all ports and stops the HTTP server.
func (n *HTTPListenNode) Close() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.outPort.Close()
	n.errPort.Close()

	return n.server.Close()
}

func (n *HTTPListenNode) negotiate(req *HTTPPayload, res *HTTPPayload) {
	if res.Header == nil {
		res.Header = http.Header{}
	}

	if res.Header.Get(mime.HeaderContentEncoding) == "" {
		accept := req.Header.Get(mime.HeaderAcceptEncoding)
		encoding := mime.Negotiate(accept, []string{mime.EncodingIdentity, mime.EncodingGzip, mime.EncodingDeflate, mime.EncodingBr})
		if encoding != "" {
			res.Header.Set(mime.HeaderContentEncoding, encoding)
		}
	}

	if res.Header.Get(mime.HeaderContentType) == "" {
		accept := req.Header.Get(mime.HeaderAccept)
		offers := mime.DetectTypesFromValue(res.Body)
		contentType := mime.Negotiate(accept, offers)
		if contentType == "" && len(offers) > 0 {
			contentType = offers[0]
		}
		if contentType != "" {
			res.Header.Set(mime.HeaderContentType, contentType)
		}
	}
}

func (n *HTTPListenNode) read(r *http.Request) (*HTTPPayload, error) {
	body, err := mime.Decode(r.Body, textproto.MIMEHeader(r.Header))
	if err != nil {
		return nil, err
	}
	return &HTTPPayload{
		Method:   r.Method,
		Scheme:   r.URL.Scheme,
		Host:     r.Host,
		Path:     r.URL.Path,
		Query:    r.URL.Query(),
		Protocol: r.Proto,
		Header:   r.Header,
		Body:     body,
	}, nil
}

func (n *HTTPListenNode) write(w http.ResponseWriter, res *HTTPPayload) error {
	if res == nil {
		return nil
	}

	h := w.Header()
	for key := range h {
		h.Del(key)
	}
	for key, headers := range res.Header {
		if !mime.IsResponseHeader(key) {
			continue
		}
		for _, header := range headers {
			h.Add(key, header)
		}
	}

	status := res.Status
	if status == 0 {
		status = http.StatusOK
	}

	w.WriteHeader(status)
	return mime.Encode(w, res.Body, textproto.MIMEHeader(h))
}
