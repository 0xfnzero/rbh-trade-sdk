package rbhtrade

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestLocalSignerRejectsMalformedPrivateKey(t *testing.T) {
	if _, err := NewLocalSigner(&ecdsa.PrivateKey{}); err == nil {
		t.Fatal("malformed private key was accepted")
	}
}

func TestLocalSignerBuildsRobinhoodDynamicFeeTransaction(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	signer, err := NewLocalSigner(key)
	if err != nil {
		t.Fatal(err)
	}
	call := Call{To: common.HexToAddress("0x1234"), Data: []byte{1, 2, 3, 4}, Value: big.NewInt(9)}
	signed, err := signer.SignDynamicFee(call, DynamicFeeParams{Nonce: 7, GasLimit: 100_000, GasTipCap: big.NewInt(2), GasFeeCap: big.NewInt(100)})
	if err != nil {
		t.Fatal(err)
	}
	tx := signed.Transaction()
	if tx == nil || tx.ChainId().Int64() != ChainID || tx.Nonce() != 7 || tx.Gas() != 100_000 || tx.To() == nil || *tx.To() != call.To || tx.Value().Cmp(call.Value) != 0 {
		t.Fatalf("unexpected signed transaction: %#v", tx)
	}
	from, err := gethtypes.Sender(gethtypes.LatestSignerForChainID(big.NewInt(ChainID)), tx)
	if err != nil || from != signer.Address() {
		t.Fatalf("recover sender: from=%s err=%v", from, err)
	}
	var decoded gethtypes.Transaction
	if err := decoded.UnmarshalBinary(signed.RawBytes()); err != nil || decoded.Hash() != tx.Hash() {
		t.Fatalf("raw transaction mismatch: err=%v", err)
	}
}

func TestLocalSignerRejectsUnsafeFeeAndCall(t *testing.T) {
	key, _ := crypto.GenerateKey()
	signer, err := NewLocalSigner(key)
	if err != nil {
		t.Fatal(err)
	}
	valid := DynamicFeeParams{GasLimit: 21_000, GasTipCap: big.NewInt(2), GasFeeCap: big.NewInt(3)}
	if _, err := signer.SignDynamicFee(Call{}, valid); err == nil {
		t.Fatal("invalid call accepted")
	}
	valid.GasFeeCap = big.NewInt(1)
	if _, err := signer.SignDynamicFee(Call{To: common.HexToAddress("0x1"), Data: []byte{1, 2, 3, 4}, Value: new(big.Int)}, valid); err == nil {
		t.Fatal("fee cap below tip accepted")
	}
}

func TestNonceSequenceConcurrentReservations(t *testing.T) {
	sequence := NewNonceSequence(100)
	const count = 1_000
	values := make(chan uint64, count)
	var wg sync.WaitGroup
	for range count {
		wg.Add(1)
		go func() {
			defer wg.Done()
			values <- sequence.Reserve()
		}()
	}
	wg.Wait()
	close(values)
	seen := make(map[uint64]bool, count)
	for value := range values {
		if seen[value] || value < 100 || value >= 100+count {
			t.Fatalf("invalid nonce reservation %d", value)
		}
		seen[value] = true
	}
	sequence.AdvanceTo(2_000)
	sequence.AdvanceTo(1_500)
	if got := sequence.Reserve(); got != 2_000 {
		t.Fatalf("nonce after reconciliation = %d", got)
	}
}

func TestNilNonceSequence(t *testing.T) {
	var sequence *NonceSequence
	if got := sequence.Reserve(); got != 0 {
		t.Fatalf("nil sequence reservation = %d", got)
	}
	if got := sequence.Peek(); got != 0 {
		t.Fatalf("nil sequence peek = %d", got)
	}
	sequence.AdvanceTo(1)
}

type fakeRawRPC struct {
	delay time.Duration
	hash  common.Hash
	err   error
}

func (f fakeRawRPC) CallContext(ctx context.Context, result any, method string, args ...any) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(f.delay):
	}
	if method != "eth_sendRawTransaction" || len(args) != 1 {
		return errors.New("unexpected RPC call")
	}
	if f.err != nil {
		return f.err
	}
	*(result.(*common.Hash)) = f.hash
	return nil
}

func TestBroadcasterReturnsFirstAcceptedRelay(t *testing.T) {
	expected := common.HexToHash("0x1234")
	broadcaster, err := NewBroadcaster(
		Relay{Name: "slow", Client: fakeRawRPC{delay: time.Second, hash: expected}},
		Relay{Name: "fast-error", Client: fakeRawRPC{err: errors.New("offline")}},
		Relay{Name: "fast", Client: fakeRawRPC{delay: time.Millisecond, hash: expected}},
	)
	if err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	hash, err := broadcaster.BroadcastRaw(context.Background(), "0x0102", expected)
	if err != nil || hash != expected {
		t.Fatalf("broadcast: hash=%s err=%v", hash, err)
	}
	if elapsed := time.Since(started); elapsed > 100*time.Millisecond {
		t.Fatalf("waited for slow relay: %s", elapsed)
	}
}

func TestBroadcasterRejectsHashMismatch(t *testing.T) {
	broadcaster, err := NewBroadcaster(Relay{Client: fakeRawRPC{hash: common.HexToHash("0x2")}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = broadcaster.BroadcastRaw(context.Background(), "0x0102", common.HexToHash("0x1"))
	if !errors.Is(err, ErrBroadcastFailed) {
		t.Fatalf("error = %v", err)
	}
}

func TestBroadcasterAcceptsAlreadyKnownButNotNonceTooLow(t *testing.T) {
	expected := common.HexToHash("0x1234")
	known, err := NewBroadcaster(Relay{Client: fakeRawRPC{err: errors.New("already known")}})
	if err != nil {
		t.Fatal(err)
	}
	if hash, err := known.BroadcastRaw(context.Background(), "0x0102", expected); err != nil || hash != expected {
		t.Fatalf("already-known broadcast: hash=%s err=%v", hash, err)
	}
	nonceLow, err := NewBroadcaster(Relay{Client: fakeRawRPC{err: errors.New("nonce too low")}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := nonceLow.BroadcastRaw(context.Background(), "0x0102", expected); !errors.Is(err, ErrBroadcastFailed) {
		t.Fatalf("nonce-too-low error = %v", err)
	}
	unknown, err := NewBroadcaster(Relay{Client: fakeRawRPC{err: errors.New("unknown transaction")}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := unknown.BroadcastRaw(context.Background(), "0x0102", expected); !errors.Is(err, ErrBroadcastUnknown) {
		t.Fatalf("unknown-transaction error = %v", err)
	}
}

func TestAlreadyKnownClassification(t *testing.T) {
	for _, message := range []string{"already known", "known transaction: 0x1234", "rpc error: known transaction"} {
		if !IsAlreadyKnownError(errors.New(message)) {
			t.Fatalf("%q was not classified as already known", message)
		}
	}
	for _, message := range []string{"unknown transaction", "transaction not known", "known transaction pool unavailable"} {
		if IsAlreadyKnownError(errors.New(message)) {
			t.Fatalf("%q was classified as already known", message)
		}
	}
}

func TestBroadcastFailureDisposition(t *testing.T) {
	to := common.HexToAddress("0x1234")
	tx := gethtypes.NewTx(&gethtypes.DynamicFeeTx{ChainID: big.NewInt(ChainID), To: &to, Value: new(big.Int), Gas: 21_000, GasTipCap: big.NewInt(1), GasFeeCap: big.NewInt(2), Data: []byte{1, 2, 3, 4}})
	_, err := NewSignedTransaction(tx)
	if err == nil {
		t.Fatal("unsigned transaction unexpectedly accepted")
	}
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	signer, err := NewLocalSigner(key)
	if err != nil {
		t.Fatal(err)
	}
	signed, err := signer.SignDynamicFee(Call{To: to, Value: new(big.Int), Data: []byte{1, 2, 3, 4}}, DynamicFeeParams{GasLimit: 21_000, GasTipCap: big.NewInt(1), GasFeeCap: big.NewInt(2)})
	if err != nil {
		t.Fatal(err)
	}

	rejected, _ := NewBroadcaster(Relay{Client: fakeRawRPC{err: errors.New("insufficient funds for gas * price + value")}})
	if _, err := rejected.Broadcast(context.Background(), signed); !errors.Is(err, ErrBroadcastRejected) || errors.Is(err, ErrBroadcastUnknown) {
		t.Fatalf("rejected disposition = %v", err)
	}
	unknown, _ := NewBroadcaster(Relay{Client: fakeRawRPC{err: context.DeadlineExceeded}})
	if _, err := unknown.Broadcast(context.Background(), signed); !errors.Is(err, ErrBroadcastUnknown) || errors.Is(err, ErrBroadcastRejected) {
		t.Fatalf("unknown disposition = %v", err)
	}
}

func TestDefinitiveBroadcastRejectionIsConservative(t *testing.T) {
	for _, message := range []string{"insufficient funds", "intrinsic gas too low", "invalid sender", "max fee per gas less than block base fee"} {
		if !IsDefinitiveBroadcastRejection(errors.New(message)) {
			t.Fatalf("%q was not classified as rejection", message)
		}
	}
	for _, message := range []string{"nonce too low", "nonce too high", "replacement transaction underpriced", "context deadline exceeded", "rate limited"} {
		if IsDefinitiveBroadcastRejection(errors.New(message)) {
			t.Fatalf("%q was unsafely classified as rejection", message)
		}
	}
}

func BenchmarkLocalSignDynamicFee(b *testing.B) {
	key, _ := crypto.GenerateKey()
	signer, _ := NewLocalSigner(key)
	call := Call{To: common.HexToAddress("0x1234"), Data: []byte{1, 2, 3, 4}, Value: new(big.Int)}
	params := DynamicFeeParams{GasLimit: 100_000, GasTipCap: big.NewInt(2), GasFeeCap: big.NewInt(100)}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := signer.SignDynamicFee(call, params); err != nil {
			b.Fatal(err)
		}
	}
}
