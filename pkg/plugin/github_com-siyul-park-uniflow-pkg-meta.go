// Code generated by 'yaegi extract github.com/siyul-park/uniflow/pkg/meta'. DO NOT EDIT.

package plugin

import (
	"github.com/gofrs/uuid"
	"github.com/siyul-park/uniflow/pkg/meta"
	"go/constant"
	"go/token"
	"reflect"
)

func init() {
	Symbols["github.com/siyul-park/uniflow/pkg/meta/meta"] = map[string]reflect.Value{
		// function, constant and variable definitions
		"DefaultNamespace": reflect.ValueOf(constant.MakeFromLiteral("\"default\"", token.STRING, 0)),
		"KeyAnnotations":   reflect.ValueOf(constant.MakeFromLiteral("\"annotations\"", token.STRING, 0)),
		"KeyID":            reflect.ValueOf(constant.MakeFromLiteral("\"id\"", token.STRING, 0)),
		"KeyName":          reflect.ValueOf(constant.MakeFromLiteral("\"name\"", token.STRING, 0)),
		"KeyNamespace":     reflect.ValueOf(constant.MakeFromLiteral("\"namespace\"", token.STRING, 0)),
		"NamespacedName":   reflect.ValueOf(meta.NamespacedName),

		// type definitions
		"Meta":         reflect.ValueOf((*meta.Meta)(nil)),
		"Unstructured": reflect.ValueOf((*meta.Unstructured)(nil)),

		// interface wrapper definitions
		"_Meta": reflect.ValueOf((*_github_com_siyul_park_uniflow_pkg_meta_Meta)(nil)),
	}
}

// _github_com_siyul_park_uniflow_pkg_meta_Meta is an interface wrapper for Meta type
type _github_com_siyul_park_uniflow_pkg_meta_Meta struct {
	IValue          interface{}
	WGetAnnotations func() map[string]string
	WGetID          func() uuid.UUID
	WGetName        func() string
	WGetNamespace   func() string
	WSetAnnotations func(val map[string]string)
	WSetID          func(val uuid.UUID)
	WSetName        func(val string)
	WSetNamespace   func(val string)
}

func (W _github_com_siyul_park_uniflow_pkg_meta_Meta) GetAnnotations() map[string]string {
	return W.WGetAnnotations()
}
func (W _github_com_siyul_park_uniflow_pkg_meta_Meta) GetID() uuid.UUID {
	return W.WGetID()
}
func (W _github_com_siyul_park_uniflow_pkg_meta_Meta) GetName() string {
	return W.WGetName()
}
func (W _github_com_siyul_park_uniflow_pkg_meta_Meta) GetNamespace() string {
	return W.WGetNamespace()
}
func (W _github_com_siyul_park_uniflow_pkg_meta_Meta) SetAnnotations(val map[string]string) {
	W.WSetAnnotations(val)
}
func (W _github_com_siyul_park_uniflow_pkg_meta_Meta) SetID(val uuid.UUID) {
	W.WSetID(val)
}
func (W _github_com_siyul_park_uniflow_pkg_meta_Meta) SetName(val string) {
	W.WSetName(val)
}
func (W _github_com_siyul_park_uniflow_pkg_meta_Meta) SetNamespace(val string) {
	W.WSetNamespace(val)
}
