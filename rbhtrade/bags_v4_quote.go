package rbhtrade

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

const BagsV4HookFeeBPS = uint16(200)

var ErrInvalidBagsV4HookState = errors.New("invalid Bags V4 hook state")

// QuoteBagsV4ExactInput reproduces the exact-match verified canonical
// BagsV4Hook. The hook overrides the core LP fee to zero and charges 2% on
// the WETH leg: before core math for buys and after core math for sells.
// Source: robinhoodchain.blockscout.com/address/
// 0x2380aBf72C17aABAb76480244759AC7E2932EEcC?tab=contract
func QuoteBagsV4ExactInput(state V4PoolState, currencyIn common.Address, amountIn *big.Int) (*big.Int, error) {
	weth := defaultAddresses.WETH
	if state.PoolKey.Hooks != defaultAddresses.BagsHook ||
		state.HookFeeBPS != 0 || state.CreatorTaxBPS != 0 ||
		(state.PoolKey.Currency0 != weth && state.PoolKey.Currency1 != weth) ||
		(currencyIn != state.PoolKey.Currency0 && currencyIn != state.PoolKey.Currency1) ||
		amountIn == nil || amountIn.Sign() <= 0 || amountIn.BitLen() > 256 {
		return nil, ErrInvalidBagsV4HookState
	}

	fee := func(value *big.Int) *big.Int {
		out := new(big.Int).Mul(value, new(big.Int).SetUint64(uint64(BagsV4HookFeeBPS)))
		return out.Div(out, big.NewInt(BPSTotal))
	}
	coreState := state
	coreState.PoolKey.Fee = 0
	if currencyIn == weth {
		effective := new(big.Int).Sub(amountIn, fee(amountIn))
		if effective.Sign() <= 0 {
			return nil, ErrInvalidBagsV4HookState
		}
		return QuoteV4ExactInput(coreState, currencyIn, effective)
	}

	out, err := QuoteV4ExactInput(coreState, currencyIn, amountIn)
	if err != nil {
		return nil, err
	}
	out.Sub(out, fee(out))
	if out.Sign() <= 0 {
		return nil, ErrInvalidBagsV4HookState
	}
	return out, nil
}
