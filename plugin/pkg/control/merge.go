package control

import (
	"sync"

	"github.com/samber/lo"
	"github.com/siyul-park/uniflow/pkg/node"
	"github.com/siyul-park/uniflow/pkg/packet"
	"github.com/siyul-park/uniflow/pkg/primitive"
	"github.com/siyul-park/uniflow/pkg/process"
	"github.com/siyul-park/uniflow/pkg/scheme"
)

// MergeNode represents a node that Merges multiple input packets into a single output packet.
type MergeNode struct {
	*node.ManyToOneNode
	depth   int
	inplace bool
	mu      sync.RWMutex
}

// MergeNodeSpec holds the specifications for creating a MergeNode.
type MergeNodeSpec struct {
	scheme.SpecMeta `map:",inline"`
	Depth           int  `map:"depth,omitempty"`
	Inplace         bool `map:"inplace,omitempty"`
}

const KindMerge = "merge"

// NewMergeNode creates a new MergeNode.
func NewMergeNode() *MergeNode {
	n := &MergeNode{
		depth:   0,
		inplace: false,
	}

	n.ManyToOneNode = node.NewManyToOneNode(n.action)

	return n
}

// Depth returns the depth of the MergeNode.
func (n *MergeNode) Depth() int {
	n.mu.RLock()
	defer n.mu.RUnlock()

	return n.depth
}

// SetDepth sets the depth of the MergeNode.
func (n *MergeNode) SetDepth(depth int) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.depth = depth
}

// Inplace returns true if the MergeNode operates inplace.
func (n *MergeNode) Inplace() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()

	return n.inplace
}

// SetInplace sets whether the MergeNode should operate inplace.
func (n *MergeNode) SetInplace(inplace bool) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if inplace && n.depth == 0 {
		n.depth = -1
	}
	n.inplace = inplace
}

func (n *MergeNode) action(proc *process.Process, inPcks []*packet.Packet) (*packet.Packet, *packet.Packet) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if !n.isFull(inPcks) {
		return nil, nil
	}

	inPayloads := lo.Map(inPcks, func(item *packet.Packet, _ int) primitive.Value {
		return item.Payload()
	})

	var outPayload primitive.Value
	if n.depth == 0 {
		outPayload = primitive.NewSlice(inPayloads...)
	} else {
		outPayload = lo.Reduce(inPayloads, func(agg primitive.Value, item primitive.Value, index int) primitive.Value {
			return n.merge(agg, item, n.depth-1)
		}, nil)
	}

	return packet.New(outPayload), nil
}

func (n *MergeNode) isFull(pcks []*packet.Packet) bool {
	for _, inPck := range pcks {
		if inPck == nil {
			return false
		}
	}
	return true
}

func (n *MergeNode) merge(x, y primitive.Value, depth int) primitive.Value {
	if depth == 0 {
		return y
	}

	if x == nil {
		return y
	}
	if y == nil {
		return x
	}

	switch x := x.(type) {
	case *primitive.Slice:
		if y, ok := y.(*primitive.Slice); ok {
			var values []primitive.Value
			if n.inplace {
				len := x.Len()
				if len < y.Len() {
					len = y.Len()
				}
				for i := 0; i < len; i++ {
					value1 := x.Get(i)
					value2 := y.Get(i)

					values = append(values, n.merge(value1, value2, depth-1))
				}
			} else {
				values = append(x.Values(), y.Values()...)
			}

			return primitive.NewSlice(values...)
		}
	case *primitive.Map:
		if y, ok := y.(*primitive.Map); ok {
			var pairs []primitive.Value
			for _, key := range x.Keys() {
				value1, ok1 := x.Get(key)
				value2, ok2 := y.Get(key)

				pairs = append(pairs, key)
				if !ok1 {
					pairs = append(pairs, value2)
				} else if !ok2 {
					pairs = append(pairs, value1)
				} else {
					pairs = append(pairs, n.merge(value1, value2, depth-1))
				}
			}
			for _, key := range y.Keys() {
				_, ok := x.Get(key)
				value, _ := y.Get(key)
				if ok {
					continue
				}
				pairs = append(pairs, key, value)
			}

			return primitive.NewMap(pairs...)
		}
	}

	return y
}

// NewMergeNodeCodec creates a new codec for MergeNodeSpec.
func NewMergeNodeCodec() scheme.Codec {
	return scheme.CodecWithType(func(spec *MergeNodeSpec) (node.Node, error) {
		n := NewMergeNode()
		n.SetDepth(spec.Depth)
		n.SetInplace(spec.Inplace)

		return n, nil
	})
}