package symbol

import (
	"sync"

	"github.com/siyul-park/uniflow/pkg/node"
	"github.com/siyul-park/uniflow/pkg/port"
	"github.com/siyul-park/uniflow/pkg/spec"
)

// Cluster manages the ports and symbol table for the cluster.
type Cluster struct {
	symbols   []*Symbol
	table     *Table
	inPorts   map[string]*port.InPort
	outPorts  map[string]*port.OutPort
	_inPorts  map[string]*port.InPort
	_outPorts map[string]*port.OutPort
	mu        sync.RWMutex
}

var (
	_ node.Node      = (*Cluster)(nil)
	_ LoadListener   = (*Cluster)(nil)
	_ UnloadListener = (*Cluster)(nil)
)

// NewCluster creates a new Cluster with the provided symbol table.
func NewCluster(symbols []*Symbol) *Cluster {
	return &Cluster{
		symbols:   symbols,
		table:     NewTable(),
		inPorts:   make(map[string]*port.InPort),
		outPorts:  make(map[string]*port.OutPort),
		_inPorts:  make(map[string]*port.InPort),
		_outPorts: make(map[string]*port.OutPort),
	}
}

// Inbound links an external input to an internal symbol's input port.
func (n *Cluster) Inbound(source string, target spec.Port) bool {
	n.mu.Lock()
	defer n.mu.Unlock()

	var sb *Symbol
	for _, s := range n.symbols {
		if (target.ID == s.ID()) || (target.Name != "" && target.Name == s.Name()) {
			sb = s
			break
		}
	}
	if sb == nil {
		return false
	}

	prt := sb.In(target.Port)
	if prt == nil {
		return false
	}

	outPort, ok := n.outPorts[source]
	if !ok {
		var inPort *port.InPort
		inPort, outPort = port.Pipe()
		n.inPorts[source] = inPort
		n._outPorts[source] = outPort
	}

	outPort.Link(prt)
	return true
}

// Outbound links an external output to an internal symbol's output port.
func (n *Cluster) Outbound(source string, target spec.Port) bool {
	n.mu.Lock()
	defer n.mu.Unlock()

	var sb *Symbol
	for _, s := range n.symbols {
		if (target.ID == s.ID()) || (target.Name != "" && target.Name == s.Name()) {
			sb = s
			break
		}
	}
	if sb == nil {
		return false
	}

	prt := sb.Out(target.Port)
	if prt == nil {
		return false
	}

	inPort, ok := n._inPorts[source]
	if !ok {
		var outPort *port.OutPort
		inPort, outPort = port.Pipe()
		n._inPorts[source] = inPort
		n.outPorts[source] = outPort
	}

	prt.Link(inPort)
	return true
}

// Load processes all initialization hooks for symbols.
func (n *Cluster) Load(hook LoadHook) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.table.AddLoadHook(hook)
	defer n.table.RemoveLoadHook(hook)

	for _, sb := range n.symbols {
		if n.table.Lookup(sb.ID()) != nil {
			continue
		}

		sb := &Symbol{
			Spec: sb.Spec,
			Node: node.NoCloser(sb.Node),
		}
		if err := n.table.Insert(sb); err != nil {
			return err
		}
	}
	return nil
}

// Unload processes all termination hooks for symbols.
func (n *Cluster) Unload(hook UnloadHook) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.table.AddUnloadHook(hook)
	defer n.table.RemoveUnloadHook(hook)

	return n.table.Close()
}

// In returns the input port by name.
func (n *Cluster) In(name string) *port.InPort {
	n.mu.RLock()
	defer n.mu.RUnlock()

	return n.inPorts[name]
}

// Out returns the output port by name.
func (n *Cluster) Out(name string) *port.OutPort {
	n.mu.RLock()
	defer n.mu.RUnlock()

	return n.outPorts[name]
}

// Close shuts down all ports and the symbol table.
func (n *Cluster) Close() error {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if err := n.table.Close(); err != nil {
		return err
	}

	for _, sb := range n.symbols {
		if err := sb.Close(); err != nil {
			return err
		}
	}

	for _, inPort := range n.inPorts {
		inPort.Close()
	}
	for _, inPort := range n._inPorts {
		inPort.Close()
	}
	for _, outPort := range n.outPorts {
		outPort.Close()
	}
	for _, outPort := range n._outPorts {
		outPort.Close()
	}
	return nil
}
