package rbhtrade

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	CommandV4Swap           byte = 0x10
	ActionSwapExactInSingle byte = 0x06
	ActionSettleAll         byte = 0x0c
	ActionTakeAll           byte = 0x0f
)

func PoolID(key PoolKey) ([32]byte, error) {
	if err := validatePoolKey(key); err != nil {
		return [32]byte{}, err
	}
	var encoded [5 * abiWordSize]byte
	putAddressWord(encoded[:], 0, key.Currency0)
	putAddressWord(encoded[:], abiWordSize, key.Currency1)
	putUint64Word(encoded[:], 2*abiWordSize, uint64(key.Fee))
	putUint64Word(encoded[:], 3*abiWordSize, uint64(key.TickSpacing))
	putAddressWord(encoded[:], 4*abiWordSize, key.Hooks)
	return crypto.Keccak256Hash(encoded[:]), nil
}

func BuildV4ExactInputSingle(req ExactInputRequest) (Call, error) {
	if len(req.HookData) > MaxHookDataBytes {
		return Call{}, fmt.Errorf("hook data exceeds %d bytes", MaxHookDataBytes)
	}
	if err := validatePoolKey(req.PoolKey); err != nil {
		return Call{}, err
	}
	if req.CurrencyIn == req.CurrencyOut {
		return Call{}, ErrSameCurrency
	}
	if err := validateUint(req.AmountIn, 128, true, "amount in"); err != nil {
		return Call{}, err
	}
	if err := validateUint(req.AmountOutMinimum, 128, false, "minimum amount out"); err != nil {
		return Call{}, err
	}
	if req.Deadline == 0 {
		return Call{}, ErrExpiredDeadline
	}

	zeroForOne := false
	switch {
	case req.CurrencyIn == req.PoolKey.Currency0 && req.CurrencyOut == req.PoolKey.Currency1:
		zeroForOne = true
	case req.CurrencyIn == req.PoolKey.Currency1 && req.CurrencyOut == req.PoolKey.Currency0:
	default:
		return Call{}, ErrCurrencyNotInPool
	}

	minHopPrice := req.MinHopPriceX36
	if minHopPrice == nil {
		minHopPrice = new(big.Int)
	}
	if err := validateUint(minHopPrice, 256, false, "minimum hop price"); err != nil {
		return Call{}, err
	}
	data, err := encodeV4ExactInputSingle(req, zeroForOne, minHopPrice)
	if err != nil {
		return Call{}, err
	}
	value := new(big.Int)
	if req.CurrencyIn == (common.Address{}) {
		value.Set(req.AmountIn)
	}
	return Call{To: defaultAddresses.UniversalRouter, Data: data, Value: value}, nil
}

func toPoolKeyABI(key PoolKey) poolKeyABI {
	return poolKeyABI{
		Currency0:   key.Currency0,
		Currency1:   key.Currency1,
		Fee:         new(big.Int).SetUint64(uint64(key.Fee)),
		TickSpacing: big.NewInt(int64(key.TickSpacing)),
		Hooks:       key.Hooks,
	}
}
