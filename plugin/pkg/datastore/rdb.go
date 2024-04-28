package datastore

import (
	"database/sql"
	"sync"

	"github.com/jmoiron/sqlx"
	"github.com/siyul-park/uniflow/pkg/node"
	"github.com/siyul-park/uniflow/pkg/packet"
	"github.com/siyul-park/uniflow/pkg/primitive"
	"github.com/siyul-park/uniflow/pkg/process"
	"github.com/siyul-park/uniflow/pkg/scheme"
)

// RDBNode represents a node for interacting with a relational database.
type RDBNode struct {
	*node.OneToOneNode
	db        *sqlx.DB
	txs       *process.Local
	isolation sql.IsolationLevel
	mu        sync.RWMutex
}

// RDBNodeSpec holds the specifications for creating a RDBNode.
type RDBNodeSpec struct {
	scheme.SpecMeta `map:",inline"`
	Driver          string             `map:"driver"`
	Source          string             `map:"source"`
	Isolation       sql.IsolationLevel `map:"isolation"`
}

const KindRDB = "rdb"

// NewRDBNode creates a new RDBNode.
func NewRDBNode(db *sqlx.DB) *RDBNode {
	n := &RDBNode{
		db:  db,
		txs: process.NewLocal(),
	}
	n.OneToOneNode = node.NewOneToOneNode(n.action)

	return n
}

// SetIsolation sets the isolation level.
func (n *RDBNode) SetIsolation(isolation sql.IsolationLevel) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.isolation = isolation
}

// Isolation returns the isolation level.
func (n *RDBNode) Isolation() sql.IsolationLevel {
	n.mu.RLock()
	defer n.mu.RUnlock()

	return n.isolation
}

// Close closes resource associated with the node.
func (n *RDBNode) Close() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if err := n.OneToOneNode.Close(); err != nil {
		return err
	}

	n.txs.Close()
	return n.db.Close()
}

func (n *RDBNode) action(proc *process.Process, inPck *packet.Packet) (*packet.Packet, *packet.Packet) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	ctx := proc.Context()

	query, ok := primitive.Pick[string](inPck.Payload())
	if !ok {
		query, ok = primitive.Pick[string](inPck.Payload(), "0")
	}
	if !ok {
		return nil, packet.WithError(packet.ErrInvalidPacket, inPck)
	}

	val, err := n.txs.LoadOrStore(proc, func() (any, error) {
		tx, err := n.db.BeginTxx(proc.Context(), &sql.TxOptions{
			Isolation: n.isolation,
		})
		if err != nil {
			return nil, err
		}

		go func() {
			<-proc.Done()
			defer tx.Rollback()
			if proc.Err() == nil {
				tx.Commit()
			}
		}()

		return tx, nil
	})
	if err != nil {
		return nil, packet.WithError(err, inPck)
	}

	tx := val.(*sqlx.Tx)

	stmt, err := tx.PrepareNamedContext(ctx, query)
	if err != nil {
		return nil, packet.WithError(err, inPck)
	}

	var rows *sqlx.Rows
	if len(stmt.Params) == 0 {
		args, _ := primitive.Pick[[]any](inPck.Payload(), "1")
		if rows, err = tx.QueryxContext(ctx, query, args...); err != nil {
			return nil, packet.WithError(err, inPck)
		}
	} else {
		var args any
		var ok bool
		args, ok = primitive.Pick[map[string]any](inPck.Payload(), "1")
		if !ok {
			args, _ = primitive.Pick[[]map[string]any](inPck.Payload(), "1")
		}
		if rows, err = stmt.QueryxContext(ctx, args); err != nil {
			return nil, packet.WithError(err, inPck)
		}
	}

	defer rows.Close()

	var results []map[string]any
	for rows.Next() {
		result := make(map[string]any)
		if err := rows.MapScan(result); err != nil {
			return nil, packet.WithError(err, inPck)
		}
		results = append(results, result)
	}

	outPayload, err := primitive.MarshalText(results)
	if err != nil {
		return nil, packet.WithError(err, inPck)
	}

	return packet.New(outPayload), nil
}

// NewRDBNodeCodec creates a new codec for RDBNodeSpec.
func NewRDBNodeCodec() scheme.Codec {
	return scheme.CodecWithType(func(spec *RDBNodeSpec) (node.Node, error) {
		db, err := sqlx.Connect(spec.Driver, spec.Source)
		if err != nil {
			return nil, err
		}

		n := NewRDBNode(db)
		n.SetIsolation(spec.Isolation)
		return n, nil
	})
}
