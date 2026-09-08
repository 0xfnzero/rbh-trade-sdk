package pons

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var poolIDArguments = mustPoolIDArguments()

var ErrInvalidPoolKey = errors.New("invalid pool key")

func BuildPoolKey(launch LaunchedToken, hook common.Address) PoolKey {
	currency0, currency1 := SortCurrencies(launch.PairToken, launch.Token)
	return PoolKey{
		Currency0:   currency0,
		Currency1:   currency1,
		Fee:         launch.PoolFee,
		TickSpacing: launch.TickSpacing,
		Hooks:       hook,
	}
}

func SortCurrencies(a, b common.Address) (common.Address, common.Address) {
	if bytes.Compare(a[:], b[:]) <= 0 {
		return a, b
	}
	return b, a
}

func PoolID(key PoolKey) (common.Hash, error) {
	if bytes.Compare(key.Currency0[:], key.Currency1[:]) >= 0 {
		return common.Hash{}, fmt.Errorf("%w: currencies must be distinct and sorted", ErrInvalidPoolKey)
	}
	if key.Fee > (1<<24)-1 {
		return common.Hash{}, fmt.Errorf("%w: fee exceeds uint24", ErrInvalidPoolKey)
	}
	if key.TickSpacing < -(1<<23) || key.TickSpacing > (1<<23)-1 {
		return common.Hash{}, fmt.Errorf("%w: tick spacing exceeds int24", ErrInvalidPoolKey)
	}
	packed, err := poolIDArguments.Pack(
		key.Currency0,
		key.Currency1,
		new(big.Int).SetUint64(uint64(key.Fee)),
		big.NewInt(int64(key.TickSpacing)),
		key.Hooks,
	)
	if err != nil {
		return common.Hash{}, err
	}
	return crypto.Keccak256Hash(packed), nil
}

func mustPoolIDArguments() abi.Arguments {
	addressType, err := abi.NewType("address", "", nil)
	if err != nil {
		panic(err)
	}
	uint24Type, err := abi.NewType("uint24", "", nil)
	if err != nil {
		panic(err)
	}
	int24Type, err := abi.NewType("int24", "", nil)
	if err != nil {
		panic(err)
	}
	return abi.Arguments{
		{Type: addressType},
		{Type: addressType},
		{Type: uint24Type},
		{Type: int24Type},
		{Type: addressType},
	}
}
