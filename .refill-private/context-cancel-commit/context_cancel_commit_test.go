package context_cancel_commit_test

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/domain"
	"benzhi-project-204af977-5caf-4da0-a40d-d40640fad1af/internal/store"
)

type gateContext struct {
	done    chan struct{}
	entered chan struct{}
	once    sync.Once
}

func (c *gateContext) Deadline() (time.Time, bool) { return time.Time{}, false }

func (c *gateContext) Done() <-chan struct{} {
	c.once.Do(func() { close(c.entered) })
	return c.done
}

func (c *gateContext) Err() error {
	select {
	case <-c.done:
		return context.Canceled
	default:
		return nil
	}
}

func (c *gateContext) Value(interface{}) interface{} { return nil }

func TestContextCancellationDoesNotCommitTransaction(t *testing.T) {
	t.Run("canceled while waiting for store lock", func(t *testing.T) {
		repository, err := store.New(filepath.Join(t.TempDir(), "ledger.json"))
		if err != nil {
			t.Fatal(err)
		}
		firstStarted := make(chan struct{})
		releaseFirst := make(chan struct{})
		firstResult := make(chan error, 1)
		go func() {
			firstResult <- repository.UpdateContext(context.Background(), func(ledger *store.Ledger) error {
				close(firstStarted)
				<-releaseFirst
				ledger.Sessions["first"] = testSession("first")
				return nil
			})
		}()
		<-firstStarted

		ctx := &gateContext{done: make(chan struct{}), entered: make(chan struct{})}
		secondResult := make(chan error, 1)
		go func() {
			secondResult <- repository.UpdateContext(ctx, func(ledger *store.Ledger) error {
				ledger.Sessions["second"] = testSession("second")
				return nil
			})
		}()
		<-ctx.entered
		close(ctx.done)
		close(releaseFirst)

		if err := <-firstResult; err != nil {
			t.Fatalf("first transaction failed: %v", err)
		}
		if err := <-secondResult; !errors.Is(err, context.Canceled) {
			t.Fatalf("canceled waiter returned %v, want context.Canceled", err)
		}
		if _, ok := repository.Snapshot().Sessions["second"]; ok {
			t.Fatal("canceled waiter committed its session")
		}
	})

	t.Run("canceled by mutator", func(t *testing.T) {
		repository, err := store.New(filepath.Join(t.TempDir(), "ledger.json"))
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		err = repository.UpdateContext(ctx, func(ledger *store.Ledger) error {
			ledger.Sessions["mutator"] = testSession("mutator")
			cancel()
			return nil
		})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("mutator cancellation returned %v, want context.Canceled", err)
		}
		if _, ok := repository.Snapshot().Sessions["mutator"]; ok {
			t.Fatal("mutator cancellation committed its session")
		}
	})
}

func testSession(id string) domain.CalibrationSession {
	now := time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC)
	return domain.CalibrationSession{
		ID:            id,
		DeviceID:      "AST-" + id,
		DeviceName:    "测试仪器",
		ObservingBand: "可见光",
		Owner:         "工程师",
		Status:        domain.StatusDraft,
		Version:       1,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}
