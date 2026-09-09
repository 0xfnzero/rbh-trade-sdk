package rbhtrade

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type RawRPC interface {
	CallContext(context.Context, any, string, ...any) error
}

type Relay struct {
	Name   string
	Client RawRPC
}

type Broadcaster struct {
	relays []Relay
}

func NewBroadcaster(relays ...Relay) (*Broadcaster, error) {
	if len(relays) == 0 {
		return nil, ErrNoBroadcasters
	}
	copyRelays := make([]Relay, len(relays))
	copy(copyRelays, relays)
	for i := range copyRelays {
		if copyRelays[i].Client == nil || isNilInterface(copyRelays[i].Client) {
			return nil, fmt.Errorf("relay %d: %w", i, ErrNoBroadcasters)
		}
		if copyRelays[i].Name == "" {
			copyRelays[i].Name = fmt.Sprintf("relay-%d", i)
		}
	}
	return &Broadcaster{relays: copyRelays}, nil
}

// Broadcast sends the already serialized transaction to every warm relay and
// returns as soon as one endpoint accepts it. It never waits for a receipt.
func (b *Broadcaster) Broadcast(ctx context.Context, signed SignedTransaction) (common.Hash, error) {
	if ctx == nil || b == nil || len(b.relays) == 0 || signed.transaction == nil || len(signed.raw) == 0 {
		return common.Hash{}, ErrInvalidTransaction
	}
	expected := signed.transaction.Hash()
	return b.BroadcastRaw(ctx, hexutil.Encode(signed.raw), expected)
}

// BroadcastRaw avoids transaction serialization and hex encoding when the
// caller caches an already encoded signed transaction.
func (b *Broadcaster) BroadcastRaw(ctx context.Context, rawHex string, expected common.Hash) (common.Hash, error) {
	if ctx == nil || b == nil || len(b.relays) == 0 || len(rawHex) < 4 || !strings.HasPrefix(rawHex, "0x") || expected == (common.Hash{}) {
		return common.Hash{}, ErrInvalidTransaction
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	type result struct {
		name string
		hash common.Hash
		err  error
	}
	results := make(chan result, len(b.relays))
	for _, relay := range b.relays {
		go func() {
			var hash common.Hash
			err := relay.Client.CallContext(ctx, &hash, "eth_sendRawTransaction", rawHex)
			results <- result{name: relay.Name, hash: hash, err: err}
		}()
	}
	errs := make([]error, 0, len(b.relays))
	allRejected := true
	for range b.relays {
		result := <-results
		if result.err == nil && result.hash == expected {
			cancel()
			return result.hash, nil
		}
		if IsAlreadyKnownError(result.err) {
			cancel()
			return expected, nil
		}
		if result.err == nil {
			result.err = fmt.Errorf("returned hash %s, want %s", result.hash, expected)
		}
		if !IsDefinitiveBroadcastRejection(result.err) {
			allRejected = false
		}
		errs = append(errs, fmt.Errorf("%s: %w", result.name, result.err))
	}
	disposition := ErrBroadcastUnknown
	if allRejected {
		disposition = ErrBroadcastRejected
	}
	return common.Hash{}, fmt.Errorf("%w: %w: %w", ErrBroadcastFailed, disposition, errors.Join(errs...))
}

// IsAlreadyKnownError reports the narrow class of RPC responses that prove
// the exact raw transaction was previously accepted by the endpoint.
func IsAlreadyKnownError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.TrimSpace(strings.ToLower(err.Error()))
	if strings.Contains(message, "already known") {
		return true
	}
	for {
		if message == "known transaction" || strings.HasPrefix(message, "known transaction:") {
			return true
		}
		separator := strings.Index(message, ": ")
		if separator < 0 {
			return false
		}
		message = message[separator+2:]
	}
}

// IsDefinitiveBroadcastRejection reports errors proving that this exact raw
// transaction was not accepted. Transport failures, rate limits, nonce
// conflicts, and replacement errors remain unknown because another relay may
// already have accepted the transaction.
func IsDefinitiveBroadcastRejection(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	for _, marker := range [...]string{
		"insufficient funds",
		"intrinsic gas too low",
		"exceeds block gas limit",
		"invalid sender",
		"invalid chain id",
		"transaction type not supported",
		"unsupported transaction type",
		"tip higher than fee cap",
		"max priority fee per gas higher than max fee per gas",
		"max fee per gas less than block base fee",
		"oversized data",
		"rlp: ",
	} {
		if strings.Contains(message, marker) {
			return true
		}
	}
	return false
}
