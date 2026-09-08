package rbhtrade

import (
	"errors"
	"fmt"
	"math/big"
	"sort"

	v3constants "github.com/daoleno/uniswapv3-sdk/constants"
	v3 "github.com/daoleno/uniswapv3-sdk/entities"
	v3utils "github.com/daoleno/uniswapv3-sdk/utils"
	"github.com/ethereum/go-ethereum/common"
)

var ErrInvalidV4State = errors.New("invalid V4 pool state")

type V4LiquidityRange struct {
	TickLower int32
	TickUpper int32
	Delta     *big.Int
}

type V4PoolState struct {
	PoolKey       PoolKey
	SqrtPriceX96  *big.Int
	Tick          int32
	Ranges        []V4LiquidityRange
	HookFeeBPS    uint16
	CreatorTaxBPS uint16
}

// QuoteV4ExactInput applies Uniswap's audited V3/V4 core swap math to an
// event-derived tick snapshot, then applies the Pons-style output hook cuts.
// The function is local-only and returns the post-hook amount received.
func QuoteV4ExactInput(state V4PoolState, currencyIn common.Address, amountIn *big.Int) (*big.Int, error) {
	if amountIn == nil || amountIn.Sign() <= 0 || amountIn.BitLen() > 256 || state.SqrtPriceX96 == nil || state.SqrtPriceX96.Sign() <= 0 || len(state.Ranges) == 0 {
		return nil, ErrInvalidV4State
	}
	if currencyIn != state.PoolKey.Currency0 && currencyIn != state.PoolKey.Currency1 || state.PoolKey.Fee >= 1_000_000 || state.PoolKey.TickSpacing <= 0 {
		return nil, ErrInvalidV4State
	}
	type rangeKey struct{ lower, upper int32 }
	rangeLiquidity := make(map[rangeKey]*big.Int, len(state.Ranges))
	for _, item := range state.Ranges {
		if item.Delta == nil || item.Delta.Sign() == 0 || item.TickLower >= item.TickUpper ||
			item.TickLower < int32(v3utils.MinTick) || item.TickUpper > int32(v3utils.MaxTick) ||
			item.TickLower%state.PoolKey.TickSpacing != 0 || item.TickUpper%state.PoolKey.TickSpacing != 0 {
			return nil, ErrInvalidV4State
		}
		key := rangeKey{lower: item.TickLower, upper: item.TickUpper}
		if rangeLiquidity[key] == nil {
			rangeLiquidity[key] = new(big.Int)
		}
		rangeLiquidity[key].Add(rangeLiquidity[key], item.Delta)
	}
	type tickTotals struct{ gross, net *big.Int }
	totals := make(map[int32]tickTotals)
	active := new(big.Int)
	for key, delta := range rangeLiquidity {
		if delta.Sign() < 0 || delta.BitLen() > 128 {
			return nil, ErrInvalidV4State
		}
		if delta.Sign() == 0 {
			continue
		}
		lower := totals[key.lower]
		if lower.gross == nil {
			lower.gross, lower.net = new(big.Int), new(big.Int)
		}
		lower.gross.Add(lower.gross, delta)
		lower.net.Add(lower.net, delta)
		totals[key.lower] = lower
		upper := totals[key.upper]
		if upper.gross == nil {
			upper.gross, upper.net = new(big.Int), new(big.Int)
		}
		upper.gross.Add(upper.gross, delta)
		upper.net.Sub(upper.net, delta)
		totals[key.upper] = upper
		if key.lower <= state.Tick && state.Tick < key.upper {
			active.Add(active, delta)
		}
	}
	if active.Sign() <= 0 || active.BitLen() > 128 {
		return nil, ErrInvalidV4State
	}
	indexes := make([]int, 0, len(totals))
	for index := range totals {
		indexes = append(indexes, int(index))
	}
	sort.Ints(indexes)
	ticks := make([]v3.Tick, 0, len(indexes))
	for _, index := range indexes {
		total := totals[int32(index)]
		if total.gross.Sign() <= 0 || total.gross.BitLen() > 128 || !fitsInt128(total.net) {
			return nil, ErrInvalidV4State
		}
		ticks = append(ticks, v3.Tick{Index: index, LiquidityGross: total.gross, LiquidityNet: total.net})
	}
	tickSpacing := int(state.PoolKey.TickSpacing)
	provider, err := v3.NewTickListDataProvider(ticks, tickSpacing)
	if err != nil {
		return nil, fmt.Errorf("%w: ticks: %v", ErrInvalidV4State, err)
	}
	result, err := quoteV4CoreExactInput(state, currencyIn, amountIn, active, provider, tickSpacing, len(ticks))
	if err != nil {
		return nil, fmt.Errorf("quote V4: %w", err)
	}
	gross := new(big.Int).Set(result)
	for _, bps := range []uint16{state.HookFeeBPS, state.CreatorTaxBPS} {
		cut := new(big.Int).Mul(gross, new(big.Int).SetUint64(uint64(bps)))
		cut.Div(cut, big.NewInt(10_000))
		result.Sub(result, cut)
	}
	if result.Sign() <= 0 {
		return nil, ErrInvalidV4State
	}
	return result, nil
}

func fitsInt128(value *big.Int) bool {
	if value.Sign() >= 0 {
		return value.BitLen() <= 127
	}
	if value.BitLen() < 128 {
		return true
	}
	if value.BitLen() > 128 {
		return false
	}
	minimumMagnitude := new(big.Int).Lsh(big.NewInt(1), 127)
	return new(big.Int).Neg(value).Cmp(minimumMagnitude) <= 0
}

// quoteV4CoreExactInput follows the audited Uniswap V3/V4 swap loop while
// taking tickSpacing explicitly from the V4 PoolKey. The upstream V3 SDK
// derives spacing from a fixed fee enum, which is incorrect for V4 dynamic or
// hook-overridden fees and can otherwise leave its loop unable to advance.
func quoteV4CoreExactInput(state V4PoolState, currencyIn common.Address, amountIn, active *big.Int, provider *v3.TickListDataProvider, tickSpacing, initializedTicks int) (*big.Int, error) {
	tick := int(state.Tick)
	if tick < v3utils.MinTick || tick >= v3utils.MaxTick {
		return nil, ErrInvalidV4State
	}
	atTick, err := v3utils.GetSqrtRatioAtTick(tick)
	if err != nil {
		return nil, err
	}
	atNextTick, err := v3utils.GetSqrtRatioAtTick(tick + 1)
	if err != nil || state.SqrtPriceX96.Cmp(atTick) < 0 || state.SqrtPriceX96.Cmp(atNextTick) > 0 {
		return nil, ErrInvalidV4State
	}

	zeroForOne := currencyIn == state.PoolKey.Currency0
	priceLimit := new(big.Int).Sub(v3utils.MaxSqrtRatio, big.NewInt(1))
	if zeroForOne {
		priceLimit = new(big.Int).Add(v3utils.MinSqrtRatio, big.NewInt(1))
	}
	remaining := new(big.Int).Set(amountIn)
	calculated := new(big.Int)
	sqrtPrice := new(big.Int).Set(state.SqrtPriceX96)
	liquidity := new(big.Int).Set(active)
	fee := v3constants.FeeAmount(state.PoolKey.Fee)
	maxIterations := initializedTicks + (v3utils.MaxTick-v3utils.MinTick+tickSpacing*256-1)/(tickSpacing*256) + 8

	for iteration := 0; remaining.Sign() != 0 && sqrtPrice.Cmp(priceLimit) != 0; iteration++ {
		if iteration >= maxIterations || liquidity.Sign() <= 0 {
			return nil, ErrInvalidV4State
		}
		startPrice := sqrtPrice
		nextTick, initialized := provider.NextInitializedTickWithinOneWord(tick, zeroForOne, tickSpacing)
		if nextTick < v3utils.MinTick {
			nextTick = v3utils.MinTick
		} else if nextTick > v3utils.MaxTick {
			nextTick = v3utils.MaxTick
		}
		nextPrice, err := v3utils.GetSqrtRatioAtTick(nextTick)
		if err != nil {
			return nil, err
		}
		target := nextPrice
		if zeroForOne && nextPrice.Cmp(priceLimit) < 0 || !zeroForOne && nextPrice.Cmp(priceLimit) > 0 {
			target = priceLimit
		}
		var amountInStep, amountOutStep, feeAmount *big.Int
		sqrtPrice, amountInStep, amountOutStep, feeAmount, err = v3utils.ComputeSwapStep(sqrtPrice, target, liquidity, remaining, fee)
		if err != nil {
			return nil, err
		}
		remaining.Sub(remaining, new(big.Int).Add(amountInStep, feeAmount))
		calculated.Sub(calculated, amountOutStep)

		if sqrtPrice.Cmp(nextPrice) == 0 {
			if initialized {
				liquidityNet := provider.GetTick(nextTick).LiquidityNet
				if zeroForOne {
					liquidityNet = new(big.Int).Neg(liquidityNet)
				}
				liquidity = v3utils.AddDelta(liquidity, liquidityNet)
			}
			if zeroForOne {
				tick = nextTick - 1
			} else {
				tick = nextTick
			}
		} else if sqrtPrice.Cmp(startPrice) != 0 {
			tick, err = v3utils.GetTickAtSqrtRatio(sqrtPrice)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, ErrInvalidV4State
		}
	}

	result := new(big.Int).Neg(calculated)
	if result.Sign() <= 0 {
		return nil, ErrInvalidV4State
	}
	return result, nil
}
