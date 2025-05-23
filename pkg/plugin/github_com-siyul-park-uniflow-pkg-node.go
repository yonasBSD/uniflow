// Code generated by 'yaegi extract github.com/siyul-park/uniflow/pkg/node'. DO NOT EDIT.

package plugin

import (
	"github.com/siyul-park/uniflow/pkg/node"
	"github.com/siyul-park/uniflow/pkg/port"
	"go/constant"
	"go/token"
	"reflect"
)

func init() {
	Symbols["github.com/siyul-park/uniflow/pkg/node/node"] = map[string]reflect.Value{
		// function, constant and variable definitions
		"IndexOfPort":      reflect.ValueOf(node.IndexOfPort),
		"NameOfPort":       reflect.ValueOf(node.NameOfPort),
		"NewManyToOneNode": reflect.ValueOf(node.NewManyToOneNode),
		"NewOneToManyNode": reflect.ValueOf(node.NewOneToManyNode),
		"NewOneToOneNode":  reflect.ValueOf(node.NewOneToOneNode),
		"NoCloser":         reflect.ValueOf(node.NoCloser),
		"PortActive":       reflect.ValueOf(constant.MakeFromLiteral("\"active\"", token.STRING, 0)),
		"PortDeative":      reflect.ValueOf(constant.MakeFromLiteral("\"deactive\"", token.STRING, 0)),
		"PortDeinit":       reflect.ValueOf(constant.MakeFromLiteral("\"deinit\"", token.STRING, 0)),
		"PortError":        reflect.ValueOf(constant.MakeFromLiteral("\"error\"", token.STRING, 0)),
		"PortIO":           reflect.ValueOf(constant.MakeFromLiteral("\"io\"", token.STRING, 0)),
		"PortIn":           reflect.ValueOf(constant.MakeFromLiteral("\"in\"", token.STRING, 0)),
		"PortInit":         reflect.ValueOf(constant.MakeFromLiteral("\"init\"", token.STRING, 0)),
		"PortOut":          reflect.ValueOf(constant.MakeFromLiteral("\"out\"", token.STRING, 0)),
		"PortWithIndex":    reflect.ValueOf(node.PortWithIndex),
		"Unwrap":           reflect.ValueOf(node.Unwrap),

		// type definitions
		"ManyToOneNode": reflect.ValueOf((*node.ManyToOneNode)(nil)),
		"Node":          reflect.ValueOf((*node.Node)(nil)),
		"OneToManyNode": reflect.ValueOf((*node.OneToManyNode)(nil)),
		"OneToOneNode":  reflect.ValueOf((*node.OneToOneNode)(nil)),
		"Proxy":         reflect.ValueOf((*node.Proxy)(nil)),

		// interface wrapper definitions
		"_Node":  reflect.ValueOf((*_github_com_siyul_park_uniflow_pkg_node_Node)(nil)),
		"_Proxy": reflect.ValueOf((*_github_com_siyul_park_uniflow_pkg_node_Proxy)(nil)),
	}
}

// _github_com_siyul_park_uniflow_pkg_node_Node is an interface wrapper for Node type
type _github_com_siyul_park_uniflow_pkg_node_Node struct {
	IValue interface{}
	WClose func() error
	WIn    func(name string) *port.InPort
	WOut   func(name string) *port.OutPort
}

func (W _github_com_siyul_park_uniflow_pkg_node_Node) Close() error {
	return W.WClose()
}
func (W _github_com_siyul_park_uniflow_pkg_node_Node) In(name string) *port.InPort {
	return W.WIn(name)
}
func (W _github_com_siyul_park_uniflow_pkg_node_Node) Out(name string) *port.OutPort {
	return W.WOut(name)
}

// _github_com_siyul_park_uniflow_pkg_node_Proxy is an interface wrapper for Proxy type
type _github_com_siyul_park_uniflow_pkg_node_Proxy struct {
	IValue  interface{}
	WUnwrap func() node.Node
}

func (W _github_com_siyul_park_uniflow_pkg_node_Proxy) Unwrap() node.Node {
	return W.WUnwrap()
}
