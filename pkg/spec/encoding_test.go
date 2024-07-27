package spec

import (
	"testing"

	"github.com/go-faker/faker/v4"
	"github.com/gofrs/uuid"
	"github.com/siyul-park/uniflow/pkg/encoding"
	"github.com/siyul-park/uniflow/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestSpecDecoder_Decode(t *testing.T) {
	dec := encoding.NewDecodeAssembler[types.Value, any]()
	dec.Add(newSpecDecoder(types.Decoder))

	unstructured := &Unstructured{
		Meta: Meta{
			ID:        uuid.Must(uuid.NewV7()),
			Namespace: DefaultNamespace,
			Name:      faker.Word(),
		},
		Fields: map[string]any{},
	}
	v, _ := types.Encoder.Encode(unstructured)

	var decoded Spec
	err := dec.Decode(v, &decoded)
	assert.NoError(t, err)

	assert.Equal(t, unstructured, decoded)
}
