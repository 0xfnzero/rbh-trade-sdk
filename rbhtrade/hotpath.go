package rbhtrade

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

var robinhoodChainID = big.NewInt(ChainID)

type DynamicFeeParams struct {
	Nonce      uint64
	GasLimit   uint64
	GasTipCap  *big.Int
	GasFeeCap  *big.Int
	AccessList gethtypes.AccessList
}

type SignedTransaction struct {
	transaction *gethtypes.Transaction
	raw         []byte
}

// LocalSigner signs Robinhood Chain transactions entirely in-process. It
// performs no RPC calls and is safe for concurrent use.
type LocalSigner struct {
	key     *ecdsa.PrivateKey
	address common.Address
	signer  gethtypes.Signer
}

func NewLocalSigner(key *ecdsa.PrivateKey) (*LocalSigner, error) {
	ownedKey, err := clonePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("private key: %w", ErrInvalidTransaction)
	}
	return &LocalSigner{
		key:     ownedKey,
		address: crypto.PubkeyToAddress(ownedKey.PublicKey),
		signer:  gethtypes.LatestSignerForChainID(robinhoodChainID),
	}, nil
}

func clonePrivateKey(key *ecdsa.PrivateKey) (owned *ecdsa.PrivateKey, err error) {
	if key == nil {
		return nil, ErrInvalidTransaction
	}
	defer func() {
		if recover() != nil {
			owned, err = nil, ErrInvalidTransaction
		}
	}()
	return crypto.ToECDSA(crypto.FromECDSA(key))
}

func (s *LocalSigner) Address() common.Address {
	if s == nil {
		return common.Address{}
	}
	return s.address
}

// SignDynamicFee builds, signs, and serializes one EIP-1559 transaction. The
// caller supplies cached nonce, fee, and gas values so this method stays off RPC.
func (s *LocalSigner) SignDynamicFee(call Call, params DynamicFeeParams) (SignedTransaction, error) {
	if s == nil || s.key == nil || s.signer == nil {
		return SignedTransaction{}, fmt.Errorf("signer: %w", ErrInvalidTransaction)
	}
	if err := validateHotTransaction(call, params); err != nil {
		return SignedTransaction{}, err
	}
	tx := gethtypes.NewTx(&gethtypes.DynamicFeeTx{
		ChainID:    robinhoodChainID,
		Nonce:      params.Nonce,
		GasTipCap:  params.GasTipCap,
		GasFeeCap:  params.GasFeeCap,
		Gas:        params.GasLimit,
		To:         &call.To,
		Value:      call.Value,
		Data:       call.Data,
		AccessList: params.AccessList,
	})
	signed, err := gethtypes.SignTx(tx, s.signer, s.key)
	if err != nil {
		return SignedTransaction{}, fmt.Errorf("sign transaction: %w", err)
	}
	raw, err := signed.MarshalBinary()
	if err != nil {
		return SignedTransaction{}, fmt.Errorf("serialize signed transaction: %w", err)
	}
	return SignedTransaction{transaction: signed, raw: raw}, nil
}

func NewSignedTransaction(tx *gethtypes.Transaction) (SignedTransaction, error) {
	if tx == nil {
		return SignedTransaction{}, ErrInvalidTransaction
	}
	chainID := tx.ChainId()
	if chainID == nil || chainID.Cmp(robinhoodChainID) != 0 {
		return SignedTransaction{}, ErrInvalidChainID
	}
	if _, err := gethtypes.Sender(gethtypes.LatestSignerForChainID(chainID), tx); err != nil {
		return SignedTransaction{}, fmt.Errorf("verify transaction signature: %w", err)
	}
	raw, err := tx.MarshalBinary()
	if err != nil {
		return SignedTransaction{}, fmt.Errorf("serialize signed transaction: %w", err)
	}
	return SignedTransaction{transaction: tx, raw: raw}, nil
}

func (s SignedTransaction) Transaction() *gethtypes.Transaction { return s.transaction }

func (s SignedTransaction) Hash() common.Hash {
	if s.transaction == nil {
		return common.Hash{}
	}
	return s.transaction.Hash()
}

func (s SignedTransaction) RawBytes() []byte { return append([]byte(nil), s.raw...) }

func validateHotTransaction(call Call, params DynamicFeeParams) error {
	if call.To == (common.Address{}) || call.Value == nil || call.Value.Sign() < 0 || call.Value.BitLen() > 256 || len(call.Data) < 4 || params.GasLimit == 0 {
		return ErrInvalidTransaction
	}
	if err := validateUint(params.GasTipCap, 256, false, "gas tip cap"); err != nil {
		return err
	}
	if err := validateUint(params.GasFeeCap, 256, true, "gas fee cap"); err != nil {
		return err
	}
	if params.GasFeeCap.Cmp(params.GasTipCap) < 0 {
		return fmt.Errorf("gas fee cap below tip cap: %w", ErrInvalidTransaction)
	}
	return nil
}

// NonceSequence reserves nonces without locks or RPC. Initialize it from
// PendingNonceAt during warmup and reconcile it outside the transaction path.
type NonceSequence struct {
	next atomic.Uint64
}

func NewNonceSequence(next uint64) *NonceSequence {
	sequence := new(NonceSequence)
	sequence.next.Store(next)
	return sequence
}

func (s *NonceSequence) Reserve() uint64 {
	return s.next.Add(1) - 1
}

func (s *NonceSequence) Peek() uint64 {
	if s == nil {
		return 0
	}
	return s.next.Load()
}

// AdvanceTo only moves forward, so stale RPC responses cannot reuse a nonce.
func (s *NonceSequence) AdvanceTo(next uint64) {
	if s == nil {
		return
	}
	for current := s.next.Load(); next > current; current = s.next.Load() {
		if s.next.CompareAndSwap(current, next) {
			return
		}
	}
}
