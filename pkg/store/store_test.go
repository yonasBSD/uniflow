package store

import (
	"context"
	"testing"

	"github.com/go-faker/faker/v4"
	"github.com/gofrs/uuid"
	"github.com/siyul-park/uniflow/pkg/database/memdb"
	"github.com/siyul-park/uniflow/pkg/spec"
	"github.com/stretchr/testify/assert"
)

const batchSize = 100

func TestStore_Index(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	st := New(memdb.NewCollection(""))

	err := st.Index(ctx)
	assert.NoError(t, err)

	err = st.Index(ctx)
	assert.NoError(t, err)
}

func TestStore_Watch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	kind := faker.UUIDHyphenated()

	st := New(memdb.NewCollection(""))

	stream, err := st.Watch(ctx, nil)
	assert.NoError(t, err)
	assert.NotNil(t, stream)

	defer stream.Close()

	go func() {
		for {
			if event, ok := <-stream.Next(); ok {
				assert.NotNil(t, event.NodeID)
			} else {
				return
			}
		}
	}()

	meta := &spec.Meta{
		ID:        uuid.Must(uuid.NewV7()),
		Kind:      kind,
		Namespace: spec.DefaultNamespace,
	}

	_, _ = st.InsertOne(ctx, meta)
	_, _ = st.UpdateOne(ctx, meta)
	_, _ = st.DeleteOne(ctx, Where[uuid.UUID](spec.KeyID).Equal(meta.GetID()))
}

func TestStore_InsertOne(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	kind := faker.UUIDHyphenated()

	st := New(memdb.NewCollection(""))

	spec := &spec.Meta{
		ID:   uuid.Must(uuid.NewV7()),
		Kind: kind,
	}

	id, err := st.InsertOne(ctx, spec)
	assert.NoError(t, err)
	assert.Equal(t, spec.GetID(), id)
}

func TestStore_InsertMany(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	kind := faker.UUIDHyphenated()

	st := New(memdb.NewCollection(""))

	var specs []spec.Spec
	for i := 0; i < batchSize; i++ {
		specs = append(specs, &spec.Meta{
			ID:   uuid.Must(uuid.NewV7()),
			Kind: kind,
		})
	}

	ids, err := st.InsertMany(ctx, specs)
	assert.NoError(t, err)
	assert.Len(t, ids, len(specs))
	for i, spec := range specs {
		assert.Equal(t, spec.GetID(), ids[i])
	}
}

func TestStore_UpdateOne(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	kind := faker.UUIDHyphenated()

	st := New(memdb.NewCollection(""))

	id := uuid.Must(uuid.NewV7())

	origin := &spec.Meta{
		ID:        id,
		Kind:      kind,
		Namespace: spec.DefaultNamespace,
		Name:      faker.UUIDHyphenated(),
	}
	patch := &spec.Meta{
		ID:        id,
		Kind:      kind,
		Namespace: spec.DefaultNamespace,
		Name:      faker.UUIDHyphenated(),
	}

	ok, err := st.UpdateOne(ctx, patch)
	assert.NoError(t, err)
	assert.False(t, ok)

	_, _ = st.InsertOne(ctx, origin)

	ok, err = st.UpdateOne(ctx, patch)
	assert.NoError(t, err)
	assert.True(t, ok)
}

func TestStore_UpdateMany(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	kind := faker.UUIDHyphenated()

	st := New(memdb.NewCollection(""))

	var ids []uuid.UUID
	for i := 0; i < batchSize; i++ {
		ids = append(ids, uuid.Must(uuid.NewV7()))
	}

	var origins []spec.Spec
	var patches []spec.Spec
	for _, id := range ids {
		origins = append(origins, &spec.Meta{
			ID:        id,
			Kind:      kind,
			Namespace: spec.DefaultNamespace,
			Name:      faker.UUIDHyphenated(),
		})
		patches = append(patches, &spec.Meta{
			ID:        id,
			Kind:      kind,
			Namespace: spec.DefaultNamespace,
			Name:      faker.UUIDHyphenated(),
		})
	}

	count, err := st.UpdateMany(ctx, patches)
	assert.NoError(t, err)
	assert.Equal(t, 0, count)

	_, _ = st.InsertMany(ctx, origins)

	count, err = st.UpdateMany(ctx, patches)
	assert.NoError(t, err)
	assert.Equal(t, len(patches), count)
}

func TestStore_DeleteOne(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	kind := faker.UUIDHyphenated()

	st := New(memdb.NewCollection(""))

	meta := &spec.Meta{
		ID:        uuid.Must(uuid.NewV7()),
		Kind:      kind,
		Namespace: spec.DefaultNamespace,
	}

	ok, err := st.DeleteOne(ctx, Where[uuid.UUID](spec.KeyID).Equal(meta.GetID()))
	assert.NoError(t, err)
	assert.False(t, ok)

	_, _ = st.InsertOne(ctx, meta)

	ok, err = st.DeleteOne(ctx, Where[uuid.UUID](spec.KeyID).Equal(meta.GetID()))
	assert.NoError(t, err)
	assert.True(t, ok)
}

func TestStore_DeleteMany(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	kind := faker.UUIDHyphenated()

	st := New(memdb.NewCollection(""))

	var ids []uuid.UUID
	for i := 0; i < batchSize; i++ {
		ids = append(ids, uuid.Must(uuid.NewV7()))
	}

	var spcs []spec.Spec
	for _, id := range ids {
		spcs = append(spcs, &spec.Meta{
			ID:        id,
			Kind:      kind,
			Namespace: spec.DefaultNamespace,
			Name:      faker.UUIDHyphenated(),
		})
	}

	count, err := st.DeleteMany(ctx, Where[uuid.UUID](spec.KeyID).In(ids...))
	assert.NoError(t, err)
	assert.Equal(t, 0, count)

	_, _ = st.InsertMany(ctx, spcs)

	count, err = st.DeleteMany(ctx, Where[uuid.UUID](spec.KeyID).In(ids...))
	assert.NoError(t, err)
	assert.Equal(t, len(spcs), count)
}

func TestStore_FindOne(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	kind := faker.UUIDHyphenated()

	st := New(memdb.NewCollection(""))

	meta := &spec.Meta{
		ID:        uuid.Must(uuid.NewV7()),
		Kind:      kind,
		Namespace: spec.DefaultNamespace,
	}

	_, _ = st.InsertOne(ctx, meta)

	def, err := st.FindOne(ctx, Where[uuid.UUID](spec.KeyID).Equal(meta.GetID()))
	assert.NoError(t, err)
	assert.NotNil(t, def)
	assert.Equal(t, meta.GetID(), def.GetID())
}

func TestStore_FindMany(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	kind := faker.UUIDHyphenated()

	st := New(memdb.NewCollection(""))

	var ids []uuid.UUID
	for i := 0; i < batchSize; i++ {
		ids = append(ids, uuid.Must(uuid.NewV7()))
	}

	var specs []spec.Spec
	for _, id := range ids {
		specs = append(specs, &spec.Meta{
			ID:        id,
			Kind:      kind,
			Namespace: spec.DefaultNamespace,
			Name:      faker.UUIDHyphenated(),
		})
	}

	_, _ = st.InsertMany(ctx, specs)

	res, err := st.FindMany(ctx, Where[uuid.UUID](spec.KeyID).In(ids...))
	assert.NoError(t, err)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Len(t, res, len(specs))
}

func BenchmarkStore_InsertOne(b *testing.B) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	kind := faker.UUIDHyphenated()

	st := New(memdb.NewCollection(""))

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		spec := &spec.Meta{
			ID:   uuid.Must(uuid.NewV7()),
			Kind: kind,
		}

		_, _ = st.InsertOne(ctx, spec)
	}
}

func BenchmarkStore_InsertMany(b *testing.B) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	kind := faker.UUIDHyphenated()

	st := New(memdb.NewCollection(""))

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var specs []spec.Spec
		for j := 0; j < batchSize; j++ {
			specs = append(specs, &spec.Meta{
				ID:   uuid.Must(uuid.NewV7()),
				Kind: kind,
			})
		}

		_, _ = st.InsertMany(ctx, specs)
	}
}

func BenchmarkStore_UpdateOne(b *testing.B) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	kind := faker.UUIDHyphenated()

	st := New(memdb.NewCollection(""))

	id := uuid.Must(uuid.NewV7())

	origin := &spec.Meta{
		ID:        id,
		Kind:      kind,
		Namespace: spec.DefaultNamespace,
		Name:      faker.UUIDHyphenated(),
	}

	_, _ = st.InsertOne(ctx, origin)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		patch := &spec.Meta{
			ID:        id,
			Kind:      kind,
			Namespace: spec.DefaultNamespace,
			Name:      faker.UUIDHyphenated(),
		}

		_, _ = st.UpdateOne(ctx, patch)
	}
}

func BenchmarkStore_UpdateMany(b *testing.B) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	kind := faker.UUIDHyphenated()

	st := New(memdb.NewCollection(""))

	var ids []uuid.UUID
	for i := 0; i < batchSize; i++ {
		ids = append(ids, uuid.Must(uuid.NewV7()))
	}

	var origins []spec.Spec
	for _, id := range ids {
		origins = append(origins, &spec.Meta{
			ID:        id,
			Kind:      kind,
			Namespace: spec.DefaultNamespace,
			Name:      faker.UUIDHyphenated(),
		})
	}

	_, _ = st.InsertMany(ctx, origins)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()

		var patches []spec.Spec
		for _, id := range ids {
			patches = append(patches, &spec.Meta{
				ID:        id,
				Kind:      kind,
				Namespace: spec.DefaultNamespace,
				Name:      faker.UUIDHyphenated(),
			})
		}

		b.StartTimer()

		_, _ = st.UpdateMany(ctx, patches)
	}
}

func BenchmarkStore_DeleteOne(b *testing.B) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	kind := faker.UUIDHyphenated()

	st := New(memdb.NewCollection(""))

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()

		meta := &spec.Meta{
			ID:        uuid.Must(uuid.NewV7()),
			Kind:      kind,
			Namespace: spec.DefaultNamespace,
		}
		_, _ = st.InsertOne(ctx, meta)

		b.StartTimer()

		_, _ = st.DeleteOne(ctx, Where[uuid.UUID](spec.KeyID).Equal(meta.GetID()))
	}
}

func BenchmarkStore_DeleteMany(b *testing.B) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	kind := faker.UUIDHyphenated()

	st := New(memdb.NewCollection(""))

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()

		var ids []uuid.UUID
		for i := 0; i < batchSize; i++ {
			ids = append(ids, uuid.Must(uuid.NewV7()))
		}

		var specs []spec.Spec
		for _, id := range ids {
			specs = append(specs, &spec.Meta{
				ID:        id,
				Kind:      kind,
				Namespace: spec.DefaultNamespace,
				Name:      faker.UUIDHyphenated(),
			})
		}

		_, _ = st.InsertMany(ctx, specs)

		b.StartTimer()

		_, _ = st.DeleteMany(ctx, Where[uuid.UUID](spec.KeyID).In(ids...))
	}

}

func BenchmarkStore_FindOne(b *testing.B) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	kind := faker.UUIDHyphenated()

	st := New(memdb.NewCollection(""))

	meta := &spec.Meta{
		ID:        uuid.Must(uuid.NewV7()),
		Kind:      kind,
		Namespace: spec.DefaultNamespace,
	}

	for i := 0; i < batchSize; i++ {
		_, _ = st.InsertOne(ctx, &spec.Meta{
			ID:        uuid.Must(uuid.NewV7()),
			Kind:      kind,
			Namespace: spec.DefaultNamespace,
		})
	}
	_, _ = st.InsertOne(ctx, meta)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = st.FindOne(ctx, Where[uuid.UUID](spec.KeyID).Equal(meta.GetID()))
	}
}

func BenchmarkStore_FindMany(b *testing.B) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	kind := faker.UUIDHyphenated()

	st := New(memdb.NewCollection(""))

	var ids []uuid.UUID
	for i := 0; i < batchSize; i++ {
		ids = append(ids, uuid.Must(uuid.NewV7()))
	}

	var specs []spec.Spec
	for _, id := range ids {
		specs = append(specs, &spec.Meta{
			ID:        id,
			Kind:      kind,
			Namespace: spec.DefaultNamespace,
			Name:      faker.UUIDHyphenated(),
		})
	}

	_, _ = st.InsertMany(ctx, specs)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = st.FindMany(ctx, Where[uuid.UUID](spec.KeyID).In(ids...))
	}
}
