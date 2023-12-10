package networkx

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strconv"

	"github.com/siyul-park/uniflow/pkg/node"
	"github.com/siyul-park/uniflow/pkg/packet"
	"github.com/siyul-park/uniflow/pkg/primitive"
	"github.com/siyul-park/uniflow/pkg/process"
	"github.com/siyul-park/uniflow/pkg/scheme"
)

// ProxyNode represents a node that acts as a reverse proxy.
type ProxyNode struct {
	*node.OneToOneNode
	target *url.URL
}

// ProxySpec defines the specification for the ProxyNode.
type ProxySpec struct {
	scheme.SpecMeta `map:",inline"`
	Target          string `map:"target"`
}

// KindProxy is the kind identifier for ProxyNode.
const KindProxy = "proxy"

var _ node.Node = (*ProxyNode)(nil)

// NewProxyNode creates a new ProxyNode with the specified target URL.
func NewProxyNode(target string) (*ProxyNode, error) {
	t, err := url.Parse(target)
	if err != nil {
		return nil, err
	}

	n := &ProxyNode{target: t}
	n.OneToOneNode = node.NewOneToOneNode(n.action)

	return n, nil
}

func (n *ProxyNode) action(proc *process.Process, inPck *packet.Packet) (*packet.Packet, *packet.Packet) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-proc.Done()
		cancel()
	}()

	var inPayload HTTPPayload
	if err := primitive.Unmarshal(inPck.Payload(), &inPayload); err != nil {
		return nil, packet.WithError(err, inPck)
	}

	req, err := n.loadPayload(ctx, inPayload)
	if err != nil {
		return nil, packet.WithError(err, inPck)
	}
	rw := httptest.NewRecorder()

	proxy := httputil.NewSingleHostReverseProxy(n.target)
	proxy.ErrorHandler = func(_ http.ResponseWriter, _ *http.Request, localErr error) {
		err = localErr
	}

	proxy.ServeHTTP(rw, req)
	if err != nil {
		return nil, packet.WithError(err, inPck)
	}

	outPayload, err := n.storePayload(rw)
	if err != nil {
		return nil, packet.WithError(err, inPck)
	}

	outPayload.Proto = inPayload.Proto
	outPayload.Path = inPayload.Path
	outPayload.Method = inPayload.Method
	outPayload.Query = inPayload.Query
	outPayload.Cookies = inPayload.Cookies

	if outPayloadBinary, err := primitive.MarshalBinary(outPayload); err != nil {
		return nil, packet.WithError(err, inPck)
	} else {
		return packet.New(outPayloadBinary), nil
	}
}

func (n *ProxyNode) loadPayload(ctx context.Context, payload HTTPPayload) (*http.Request, error) {
	url := &url.URL{
		Path:     payload.Path,
		RawQuery: payload.Query.Encode(),
	}

	contentType := payload.Header.Get(HeaderContentType)
	body, err := MarshalMIME(payload.Body, &contentType)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		payload.Method,
		url.RequestURI(),
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, err
	}

	req.Proto = payload.Proto
	req.Header = payload.Header
	if req.Header == nil {
		req.Header = make(http.Header)
	}
	req.Header.Set(HeaderContentLength, strconv.Itoa(len(body)))
	for _, cookie := range payload.Cookies {
		req.AddCookie(cookie)
	}

	return req, nil
}

func (n *ProxyNode) storePayload(rw *httptest.ResponseRecorder) (HTTPPayload, error) {
	contentType := rw.Header().Get(HeaderContentType)

	bodyBytes, err := io.ReadAll(rw.Body)
	if err != nil {
		return HTTPPayload{}, err
	}

	body, err := UnmarshalMIME(bodyBytes, &contentType)
	if err != nil {
		return HTTPPayload{}, err
	}

	rw.Header().Set(HeaderContentType, contentType)

	return HTTPPayload{
		Header: rw.Header(),
		Body:   body,
	}, nil
}
