package text

import "github.com/siyul-park/uniflow/ext/language"

func NewCompiler() language.Compiler {
	return language.CompileFunc(func(code string) (language.Program, error) {
		return language.RunFunc(func(_ any) (any, error) {
			return code, nil
		}), nil
	})
}