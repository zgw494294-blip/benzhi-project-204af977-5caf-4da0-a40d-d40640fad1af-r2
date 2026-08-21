package store

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
)

type Store struct {
	mu   sync.RWMutex
	path string
	data Ledger
}

func New(path string) (*Store, error) {
	if path == "" {
		path = filepath.Join("data", "ledger.json")
	}
	ledger, err := loadLedger(path)
	if err != nil {
		return nil, err
	}
	ledger = normalizeLedger(ledger)
	if err := validateLedger(ledger); err != nil {
		return nil, err
	}
	return &Store{path: path, data: ledger}, nil
}

func (s *Store) Path() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.path
}

func (s *Store) Snapshot() Ledger {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snapshot, err := cloneLedger(s.data)
	if err != nil {
		return NewLedger()
	}
	return snapshot
}

// UpdateContext is the cancellation-aware entry point for callers that own a request lifecycle.
func (s *Store) UpdateContext(ctx context.Context, mutator func(*Ledger) error) error {
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	candidate, err := cloneLedger(s.data)
	if err != nil {
		return fmt.Errorf("复制账本失败: %w", err)
	}
	if err := mutator(&candidate); err != nil {
		return err
	}
	candidate = normalizeLedger(candidate)
	candidate.Revision++
	if err := validateLedgerForCommit(candidate); err != nil {
		return err
	}
	if err := persistLedger(s.path, candidate); err != nil {
		return err
	}
	s.data = candidate
	return nil
}

func (s *Store) Update(mutator func(*Ledger) error) error {
	return s.UpdateContext(context.Background(), mutator)
}
