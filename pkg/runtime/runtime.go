package runtime

import (
	"context"
	"errors"
	"reflect"
	"sync"

	"github.com/gofrs/uuid"
	"golang.org/x/sync/errgroup"

	"github.com/siyul-park/uniflow/pkg/driver"
	"github.com/siyul-park/uniflow/pkg/hook"
	"github.com/siyul-park/uniflow/pkg/meta"
	"github.com/siyul-park/uniflow/pkg/node"
	"github.com/siyul-park/uniflow/pkg/scheme"
	"github.com/siyul-park/uniflow/pkg/spec"
	"github.com/siyul-park/uniflow/pkg/symbol"
	"github.com/siyul-park/uniflow/pkg/value"
)

// Config defines configuration options for the Runtime.
type Config struct {
	Namespace   string            // Namespace defines the isolated execution environment for workflows.
	Environment map[string]string // Environment holds the variables for the loader.
	Hook        *hook.Hook        // Hook is a collection of hook functions for managing symbols.
	Scheme      *scheme.Scheme    // Scheme defines the scheme and behaviors for symbols.
	SpecStore   driver.Store      // SpecStore is responsible for persisting specifications.
	ValueStore  driver.Store      // ValueStore is responsible for persisting values.
}

// Runtime represents an environment for executing Workflows.
type Runtime struct {
	namespace   string
	environment map[string]string
	scheme      *scheme.Scheme
	symbolTable *symbol.Table
	specStore   driver.Store
	valueStore  driver.Store
	specStream  driver.Stream
	valueStream driver.Stream
	mu          sync.RWMutex
}

// New creates a new Runtime instance with the specified configuration.
func New(config Config) *Runtime {
	if config.Namespace == "" {
		config.Namespace = meta.DefaultNamespace
	}
	if config.Hook == nil {
		config.Hook = hook.New()
	}
	if config.Scheme == nil {
		config.Scheme = scheme.New()
	}
	if config.SpecStore == nil {
		config.SpecStore = driver.NewStore()
	}
	if config.ValueStore == nil {
		config.ValueStore = driver.NewStore()
	}

	config.Hook.AddLoadHook(symbol.LoadListenerHook(config.Hook))
	config.Hook.AddUnloadHook(symbol.UnloadListenerHook(config.Hook))

	symbolTable := symbol.NewTable(symbol.TableOption{
		LoadHooks:   []symbol.LoadHook{config.Hook},
		UnloadHooks: []symbol.UnloadHook{config.Hook},
	})

	return &Runtime{
		namespace:   config.Namespace,
		environment: config.Environment,
		scheme:      config.Scheme,
		symbolTable: symbolTable,
		specStore:   config.SpecStore,
		valueStore:  config.ValueStore,
	}
}

// Load loads symbols from the spec store into the symbol table.
func (r *Runtime) Load(ctx context.Context, filter any) error {
	if filter == nil {
		filter = map[string]any{meta.KeyNamespace: r.namespace}
	} else {
		filter = map[string]any{"$and": []any{filter, map[string]any{meta.KeyNamespace: r.namespace}}}
	}

	cursor, err := r.specStore.Find(ctx, filter)
	if err != nil {
		return err
	}

	var specs []*spec.Unstructured
	if err := cursor.All(ctx, &specs); err != nil {
		return err
	}

	var filters []any
	for _, sp := range specs {
		for _, val := range sp.GetEnv() {
			if val.ID != uuid.Nil {
				filters = append(filters, map[string]any{value.KeyNamespace: sp.GetNamespace(), value.KeyID: val.ID})
			} else if val.Name != "" {
				filters = append(filters, map[string]any{value.KeyNamespace: sp.GetNamespace(), value.KeyName: val.Name})
			}
		}
	}

	var values []*value.Value
	if len(filters) > 0 {
		cursor, err = r.valueStore.Find(ctx, map[string]any{"$or": filters})
		if err != nil {
			return err
		}
		if err := cursor.All(ctx, &values); err != nil {
			return err
		}
	}

	if len(r.environment) > 0 {
		values = append(values, &value.Value{Data: r.environment})
	}

	var symbols []*symbol.Symbol
	var errs []error
	for _, unstructured := range specs {
		sp := spec.Spec(unstructured)
		if err := unstructured.Bind(values...); err != nil {
			errs = append(errs, err)
		} else if err := unstructured.Build(); err != nil {
			errs = append(errs, err)
		} else if decode, err := r.scheme.Decode(unstructured); err != nil {
			errs = append(errs, err)
		} else {
			sp = decode
		}

		sb := r.symbolTable.Lookup(sp.GetID())
		if sb == nil || !reflect.DeepEqual(sb.Spec, sp) {
			var n node.Node
			if sp != unstructured {
				if n, err = r.scheme.Compile(sp); err != nil {
					errs = append(errs, err)
				}
			}

			sb = &symbol.Symbol{Spec: unstructured, Node: n}
			if err := r.symbolTable.Insert(sb); err != nil {
				errs = append(errs, err)
			}
		}

		symbols = append(symbols, sb)
	}

	for _, id := range r.symbolTable.Keys() {
		sb := r.symbolTable.Lookup(id)
		if sb == nil {
			continue
		}

		local := driver.NewStore()
		if err := local.Insert(ctx, []any{sb.Spec}); err != nil {
			errs = append(errs, err)
			continue
		}

		cursor, err := local.Find(ctx, filter)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if !cursor.Next(ctx) {
			_ = cursor.Close(ctx)
			continue
		}
		_ = cursor.Close(ctx)

		ok := false
		for _, s := range symbols {
			if s.ID() == id {
				ok = true
				break
			}
		}
		if !ok {
			if _, err := r.symbolTable.Free(id); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errors.Join(errs...)
}

// Watch sets up watchers for specification and value changes.
func (r *Runtime) Watch(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.specStream != nil {
		if err := r.specStream.Close(ctx); err != nil {
			return err
		}
	}
	specStream, err := r.specStore.Watch(ctx, map[string]any{spec.KeyNamespace: r.namespace})
	if err != nil {
		return err
	}
	r.specStream = specStream

	if r.valueStream != nil {
		if err := r.valueStream.Close(ctx); err != nil {
			return err
		}
	}
	valueStream, err := r.valueStore.Watch(ctx, map[string]any{value.KeyNamespace: r.namespace})
	if err != nil {
		return err
	}
	r.valueStream = valueStream

	return nil
}

// Reconcile reconciles the state of symbols based on changes in specifications and values.
func (r *Runtime) Reconcile(ctx context.Context) error {
	r.mu.RLock()

	specStream := r.specStream
	valueStream := r.valueStream

	r.mu.RUnlock()

	if specStream == nil || valueStream == nil {
		return nil
	}

	g, _ := errgroup.WithContext(ctx)

	g.Go(func() error {
		for specStream.Next(ctx) {
			var event driver.Event
			if err := specStream.Decode(&event); err != nil {
				return err
			}

			_ = r.Load(ctx, map[string]any{spec.KeyID: event.ID})
		}
		return nil
	})

	g.Go(func() error {
		for valueStream.Next(ctx) {
			var event driver.Event
			if err := valueStream.Decode(&event); err != nil {
				return err
			}

			cursor, err := r.valueStore.Find(ctx, map[string]any{value.KeyID: event.ID})
			if err != nil {
				return err
			}

			var values []*value.Value
			if err := cursor.All(ctx, &values); err != nil {
				return err
			}
			values = append(values, &value.Value{ID: event.ID})

			var filters []any
			for _, id := range r.symbolTable.Keys() {
				if sb := r.symbolTable.Lookup(id); sb != nil {
					unstructured := &spec.Unstructured{}
					if err := spec.As(sb.Spec, unstructured); err != nil {
						return err
					} else if unstructured.IsBound(values...) {
						filters = append(filters, map[string]any{spec.KeyID: id})
					}
				}
			}

			if len(filters) > 0 {
				_ = r.Load(ctx, map[string]any{"$or": filters})
			}
		}
		return nil
	})

	return g.Wait()
}

// Close shuts down the Runtime by closing streams and clearing the symbol table.
func (r *Runtime) Close(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.specStream != nil {
		if err := r.specStream.Close(ctx); err != nil {
			return err
		}
		r.specStream = nil
	}
	if r.valueStream != nil {
		if err := r.valueStream.Close(ctx); err != nil {
			return err
		}
		r.valueStream = nil
	}
	return r.symbolTable.Close()
}
