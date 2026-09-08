package pons

import (
	"context"
	"errors"
	"math/big"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

type snapshotCaller struct {
	blockCalls atomic.Int32
}

func (*snapshotCaller) CodeAt(context.Context, common.Address, *big.Int) ([]byte, error) {
	return nil, nil
}

func (*snapshotCaller) CallContract(context.Context, ethereum.CallMsg, *big.Int) ([]byte, error) {
	return nil, nil
}

func (c *snapshotCaller) BlockNumber(context.Context) (uint64, error) {
	c.blockCalls.Add(1)
	return 123, nil
}

func TestSnapshotCallOptsPinsLatestBlock(t *testing.T) {
	caller := &snapshotCaller{}
	client := &Client{caller: caller}
	original := &bind.CallOpts{}
	got, err := client.snapshotCallOpts(context.Background(), original)
	if err != nil {
		t.Fatal(err)
	}
	if got.BlockNumber == nil || got.BlockNumber.Uint64() != 123 {
		t.Fatalf("block number = %v, want 123", got.BlockNumber)
	}
	if original.BlockNumber != nil {
		t.Fatal("snapshotCallOpts mutated caller-owned options")
	}
	if caller.blockCalls.Load() != 1 {
		t.Fatalf("BlockNumber calls = %d, want 1", caller.blockCalls.Load())
	}
}

func TestSnapshotCallOptsPreservesExplicitBlock(t *testing.T) {
	caller := &snapshotCaller{}
	client := &Client{caller: caller}
	got, err := client.snapshotCallOpts(context.Background(), &bind.CallOpts{BlockNumber: big.NewInt(77)})
	if err != nil {
		t.Fatal(err)
	}
	if got.BlockNumber.Cmp(big.NewInt(77)) != 0 {
		t.Fatalf("block number = %s, want 77", got.BlockNumber)
	}
	if caller.blockCalls.Load() != 0 {
		t.Fatalf("unexpected BlockNumber calls: %d", caller.blockCalls.Load())
	}
}

func TestNormalizeCallOptsUsesMethodContext(t *testing.T) {
	type contextKey string
	methodCtx := context.WithValue(context.Background(), contextKey("source"), "method")
	optionCtx := context.WithValue(context.Background(), contextKey("source"), "options")
	original := &bind.CallOpts{Context: optionCtx}

	got := normalizeCallOpts(methodCtx, original)
	if got.Context != methodCtx || got.Context.Value(contextKey("source")) != "method" {
		t.Fatal("method context must control cancellation and deadlines")
	}
	if original.Context != optionCtx {
		t.Fatal("normalizeCallOpts mutated caller-owned options")
	}
}

func TestParallelCallsRunsConcurrently(t *testing.T) {
	var active, maximum atomic.Int32
	task := func(context.Context) error {
		current := active.Add(1)
		for {
			previous := maximum.Load()
			if current <= previous || maximum.CompareAndSwap(previous, current) {
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
		active.Add(-1)
		return nil
	}
	if err := parallelCalls(context.Background(), task, task, task, task); err != nil {
		t.Fatal(err)
	}
	if maximum.Load() < 2 {
		t.Fatalf("maximum concurrency = %d, want at least 2", maximum.Load())
	}
}

func TestParallelCallsCancelsPeers(t *testing.T) {
	wantErr := errors.New("rpc failed")
	peerStarted := make(chan struct{})
	peerCanceled := make(chan struct{})
	err := parallelCalls(context.Background(),
		func(context.Context) error {
			<-peerStarted
			return wantErr
		},
		func(ctx context.Context) error {
			close(peerStarted)
			<-ctx.Done()
			close(peerCanceled)
			return ctx.Err()
		},
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
	select {
	case <-peerCanceled:
	default:
		t.Fatal("peer task did not observe cancellation")
	}
}

func TestParallelCallsLimitBoundsConcurrency(t *testing.T) {
	var active, maximum atomic.Int32
	tasks := make([]func(context.Context) error, 24)
	for i := range tasks {
		tasks[i] = func(context.Context) error {
			current := active.Add(1)
			for {
				previous := maximum.Load()
				if current <= previous || maximum.CompareAndSwap(previous, current) {
					break
				}
			}
			time.Sleep(5 * time.Millisecond)
			active.Add(-1)
			return nil
		}
	}

	if err := parallelCallsLimit(context.Background(), 3, tasks...); err != nil {
		t.Fatal(err)
	}
	if got := maximum.Load(); got < 2 || got > 3 {
		t.Fatalf("maximum concurrency = %d, want 2..3", got)
	}
}

func TestParallelCallsLimitStopsAfterFirstError(t *testing.T) {
	wantErr := errors.New("rpc failed")
	var calls atomic.Int32
	tasks := make([]func(context.Context) error, 24)
	for i := range tasks {
		i := i
		tasks[i] = func(context.Context) error {
			calls.Add(1)
			if i == 0 {
				return wantErr
			}
			return nil
		}
	}

	err := parallelCallsLimit(context.Background(), 1, tasks...)
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("calls = %d, want 1", got)
	}
}

func TestLaunchConfigEnumerationSizeRejectsLimitBeforeAllocation(t *testing.T) {
	if _, err := launchConfigEnumerationSize(big.NewInt(4097), 4096); !errors.Is(err, ErrLaunchConfigLimit) {
		t.Fatalf("expected ErrLaunchConfigLimit, got %v", err)
	}
	if got, err := launchConfigEnumerationSize(big.NewInt(4096), 4096); err != nil || got != 4096 {
		t.Fatalf("size = %d, err = %v", got, err)
	}
}
