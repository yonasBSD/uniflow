package fmt

import (
	"io"

	"gopkg.in/yaml.v3"

	"github.com/siyul-park/uniflow/pkg/types"
)

// Reader reads and processes YAML data from an fmt.Reader.
type Reader struct {
	reader io.Reader
}

// NewReader returns a new Reader for the given fmt.Reader.
func NewReader(reader io.Reader) *Reader {
	return &Reader{reader: reader}
}

// Read reads from the Reader, parses YAML, encodes, and decodes into value.
func (r *Reader) Read(value any) error {
	bytes, err := io.ReadAll(r.reader)
	if err != nil {
		return err
	}

	var data any
	if err := yaml.Unmarshal(bytes, &data); err != nil {
		return err
	}

	doc, err := types.Marshal(data)
	if err != nil {
		return err
	}

	return types.Unmarshal(doc, value)
}
