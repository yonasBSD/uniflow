package json

import (
	"context"
	"encoding/json"

	"github.com/siyul-park/uniflow/pkg/language"
)

const Language = "json"

// NewCompiler returns a compiler that parses JSON code.
func NewCompiler() language.Compiler {
	return language.CompileFunc(func(code string) (language.Program, error) {
		var data any
		if err := json.Unmarshal([]byte(code), &data); err != nil {
			return nil, err
		}
		return language.RunFunc(func(_ context.Context, _ ...any) (any, error) {
			return data, nil
		}), nil
	})
}
