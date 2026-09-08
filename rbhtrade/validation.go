package rbhtrade

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

var (
	maxUint256 = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
)

const (
	MaxHookDataBytes       = 64 << 10
	MaxDynamicPayloadBytes = 1 << 20
	MaxDynamicItems        = 4096
	MaxTokenNameBytes      = 256
	MaxTokenSymbolBytes    = 64
)

func validateUint(value *big.Int, bits int, positive bool, field string) error {
	if value == nil || value.Sign() < 0 || (positive && value.Sign() == 0) || value.BitLen() > bits {
		return fmt.Errorf("%s: %w", field, ErrInvalidAmount)
	}
	return nil
}

func validateAddress(address common.Address, field string) error {
	if address == (common.Address{}) {
		return fmt.Errorf("%s: %w", field, ErrInvalidAddress)
	}
	return nil
}

func validatePoolKey(key PoolKey) error {
	if key.Currency0 == key.Currency1 {
		return ErrInvalidPoolKey
	}
	if key.Currency0.Cmp(key.Currency1) >= 0 {
		return fmt.Errorf("currency0 must sort before currency1: %w", ErrInvalidPoolKey)
	}
	if key.TickSpacing <= 0 || key.TickSpacing < -8388608 || key.TickSpacing > 8388607 || key.Fee > 0xffffff {
		return ErrInvalidPoolKey
	}
	return nil
}

func validateMetadata(name, symbol string) error {
	if name == "" || symbol == "" {
		return fmt.Errorf("name and symbol must be non-empty")
	}
	if len(name) > MaxTokenNameBytes || len(symbol) > MaxTokenSymbolBytes {
		return fmt.Errorf("token name or symbol exceeds SDK metadata limit")
	}
	return nil
}

func validateDynamicPayload(field string, items int, sizes ...int) error {
	if items < 0 || items > MaxDynamicItems {
		return fmt.Errorf("%s contains too many items", field)
	}
	total := 0
	for _, size := range sizes {
		if size < 0 || size > MaxDynamicPayloadBytes-total {
			return fmt.Errorf("%s exceeds %d bytes", field, MaxDynamicPayloadBytes)
		}
		total += size
	}
	return nil
}

func cloneBig(value *big.Int) *big.Int {
	if value == nil {
		return new(big.Int)
	}
	return new(big.Int).Set(value)
}
