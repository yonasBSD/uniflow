package io

import (
	"bytes"
	"io"
	"net/http"
	"net/textproto"
	"os"
	"sync"

	"github.com/siyul-park/uniflow/ext/pkg/mime"
	"github.com/siyul-park/uniflow/pkg/node"
	"github.com/siyul-park/uniflow/pkg/packet"
	"github.com/siyul-park/uniflow/pkg/process"
	"github.com/siyul-park/uniflow/pkg/scheme"
	"github.com/siyul-park/uniflow/pkg/spec"
	"github.com/siyul-park/uniflow/pkg/types"
)

// ReadNode represents a node responsible for reading data from an io.ReadCloser.
type ReadNode struct {
	*node.OneToOneNode
	reader io.ReadCloser
	mu     sync.RWMutex
}

// ReadNodeSpec holds the specifications for creating a ReadNode.
type ReadNodeSpec struct {
	spec.Meta `map:",inline"`
	Filename  string `map:"filename"`
}

const KindRead = "read"

// NewReadNode creates a new ReadNode with the provided reader.
func NewReadNode(reader io.ReadCloser) *ReadNode {
	n := &ReadNode{reader: reader}
	n.OneToOneNode = node.NewOneToOneNode(n.action)
	return n
}

// Close closes the ReadNode and its underlying reader.
func (n *ReadNode) Close() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if err := n.OneToOneNode.Close(); err != nil {
		return err
	}
	return n.reader.Close()
}

func (n *ReadNode) action(proc *process.Process, inPck *packet.Packet) (*packet.Packet, *packet.Packet) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	inPayload := inPck.Payload()

	var length int
	_ = types.Decoder.Decode(inPayload, &length)

	var buf []byte
	var err error

	if length <= 0 {
		buf, err = io.ReadAll(n.reader)
		if err != nil {
			return nil, packet.New(types.NewError(err))
		}
	} else {
		buf = make([]byte, length)
		if _, err = n.reader.Read(buf); err != nil && err != io.EOF {
			return nil, packet.New(types.NewError(err))
		}
	}

	typ := http.DetectContentType(buf)
	header := textproto.MIMEHeader{mime.HeaderContentType: []string{typ}}
	if v, err := mime.Decode(bytes.NewBuffer(buf), header); err != nil {
		return packet.New(types.NewBinary(buf)), nil
	} else {
		return packet.New(v), nil
	}
}

// NewReadNodeCodec creates a codec for ReadNodeSpec to ReadNode conversion.
func NewReadNodeCodec() scheme.Codec {
	return scheme.CodecWithType(func(spec *ReadNodeSpec) (node.Node, error) {
		reader, err := OpenFile(spec.Filename, os.O_RDONLY|os.O_CREATE, 0644)
		if err != nil {
			return nil, err
		}
		return NewReadNode(reader), nil
	})
}
