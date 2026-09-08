package rbhtrade

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

const (
	O1MaxBaseFeeBPS  = uint16(1_000)
	O1MaxTotalFeeBPS = uint16(9_900)
)

var ErrInvalidO1HookState = errors.New("invalid o1 hook state")

// O1HookFeeSchedule mirrors the immutable per-pool fee fields in the exact-
// match verified LaunchHook at the canonical Robinhood Chain o1 hook address.
// Source: robinhoodchain.blockscout.com/address/
// 0x0310cFEbE1D7A69f2414f6595bBe9d17c5342aCc?tab=contract
type O1HookFeeSchedule struct {
	BaseFeeBPS             uint16
	AntiSnipeStartTotalBPS uint16
	AntiSnipeWindowSeconds uint32
	LaunchTime             uint64
}

// O1HookFeeBPS reproduces LaunchHook._totalFeeBps for a known block timestamp.
func O1HookFeeBPS(schedule O1HookFeeSchedule, timestamp uint64) (uint16, error) {
	if schedule.BaseFeeBPS > O1MaxBaseFeeBPS ||
		schedule.AntiSnipeStartTotalBPS < schedule.BaseFeeBPS ||
		schedule.AntiSnipeStartTotalBPS > O1MaxTotalFeeBPS ||
		schedule.AntiSnipeWindowSeconds == 0 || schedule.LaunchTime == 0 {
		return 0, ErrInvalidO1HookState
	}
	if timestamp < schedule.LaunchTime {
		return 0, ErrInvalidO1HookState
	}
	if timestamp == schedule.LaunchTime {
		return schedule.AntiSnipeStartTotalBPS, nil
	}
	elapsed := timestamp - schedule.LaunchTime
	window := uint64(schedule.AntiSnipeWindowSeconds)
	if elapsed >= window {
		return schedule.BaseFeeBPS, nil
	}
	maxSurcharge := uint64(schedule.AntiSnipeStartTotalBPS - schedule.BaseFeeBPS)
	surcharge := maxSurcharge * (window - elapsed) / window
	total := uint64(schedule.BaseFeeBPS) + surcharge
	if total > uint64(O1MaxTotalFeeBPS) {
		total = uint64(O1MaxTotalFeeBPS)
	}
	return uint16(total), nil
}

// QuoteO1V4ExactInput applies the verified o1 LaunchHook fee to the quote
// currency. For buys the fee reduces quote input before core swap math; for
// sells it is deducted from quote output after core swap math.
func QuoteO1V4ExactInput(state V4PoolState, currencyIn, quote common.Address, amountIn *big.Int, feeBPS uint16) (*big.Int, error) {
	if state.PoolKey.Hooks != defaultAddresses.O1Hook || feeBPS > O1MaxTotalFeeBPS ||
		state.HookFeeBPS != 0 || state.CreatorTaxBPS != 0 ||
		(quote != state.PoolKey.Currency0 && quote != state.PoolKey.Currency1) ||
		(currencyIn != state.PoolKey.Currency0 && currencyIn != state.PoolKey.Currency1) ||
		amountIn == nil || amountIn.Sign() <= 0 {
		return nil, ErrInvalidO1HookState
	}
	fee := func(value *big.Int) *big.Int {
		out := new(big.Int).Mul(value, new(big.Int).SetUint64(uint64(feeBPS)))
		return out.Div(out, big.NewInt(BPSTotal))
	}
	if currencyIn == quote {
		effective := new(big.Int).Sub(amountIn, fee(amountIn))
		if effective.Sign() <= 0 {
			return nil, ErrInvalidO1HookState
		}
		return QuoteV4ExactInput(state, currencyIn, effective)
	}
	out, err := QuoteV4ExactInput(state, currencyIn, amountIn)
	if err != nil {
		return nil, err
	}
	out.Sub(out, fee(out))
	if out.Sign() <= 0 {
		return nil, ErrInvalidO1HookState
	}
	return out, nil
}
