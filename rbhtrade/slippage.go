package rbhtrade

import (
	"fmt"
	"math/big"
)

// MinOutputWithSlippage applies a basis-point haircut to a quoted output.
// It uses integer floor division, matching the conservative value expected by
// min-output calldata fields.
func MinOutputWithSlippage(expectedOut *big.Int, slippageBPS uint64) (*big.Int, error) {
	if err := validateUint(expectedOut, 256, false, "expected output"); err != nil {
		return nil, err
	}
	if slippageBPS > BPSTotal {
		return nil, fmt.Errorf("slippage BPS %d: %w", slippageBPS, ErrInvalidBPS)
	}
	multiplier := new(big.Int).SetUint64(uint64(BPSTotal) - slippageBPS)
	minimum := new(big.Int).Mul(expectedOut, multiplier)
	return minimum.Div(minimum, big.NewInt(BPSTotal)), nil
}
