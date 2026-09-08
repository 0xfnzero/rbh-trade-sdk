package pons

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

const (
	BPS       uint64 = 10_000
	OnePctBPS uint64 = 100
)

var (
	ErrCurveClosed     = errors.New("curve is closed for trading")
	ErrInvalidQuote    = errors.New("invalid quote state")
	ErrInvalidAmount   = errors.New("invalid arithmetic input")
	ErrUnquotableTrade = errors.New("trade output rounds to zero")
	bpsInt             = new(big.Int).SetUint64(BPS)
	onePctBPSInt       = new(big.Int).SetUint64(OnePctBPS)
)

func AmountOut(inAmount, reserveIn, reserveOut *big.Int) (*big.Int, error) {
	if err := validateUint256(inAmount, "input amount"); err != nil {
		return nil, err
	}
	if reserveIn == nil || reserveIn.Sign() <= 0 || reserveIn.BitLen() > 256 ||
		reserveOut == nil || reserveOut.Sign() <= 0 || reserveOut.BitLen() > 256 {
		return nil, fmt.Errorf("%w: reserves must be positive", ErrInvalidAmount)
	}
	denominator := new(big.Int).Add(reserveIn, inAmount)
	if denominator.BitLen() > 256 {
		return nil, fmt.Errorf("%w: reserve plus input exceeds uint256", ErrInvalidAmount)
	}
	result := new(big.Int).Mul(inAmount, reserveOut)
	if result.BitLen() > 256 {
		return nil, fmt.Errorf("%w: output numerator exceeds uint256", ErrInvalidAmount)
	}
	return result.Div(result, denominator), nil
}

func AmountIn(outAmount, reserveIn, reserveOut *big.Int) (*big.Int, error) {
	if outAmount == nil || outAmount.Sign() <= 0 || outAmount.BitLen() > 256 {
		return nil, fmt.Errorf("%w: output amount must be positive", ErrInvalidAmount)
	}
	if reserveIn == nil || reserveIn.Sign() <= 0 || reserveIn.BitLen() > 256 ||
		reserveOut == nil || reserveOut.Sign() <= 0 || reserveOut.BitLen() > 256 {
		return nil, fmt.Errorf("%w: reserves must be positive", ErrInvalidAmount)
	}
	denominator := new(big.Int).Sub(reserveOut, outAmount)
	if denominator.Sign() <= 0 {
		return nil, fmt.Errorf("%w: output amount exceeds available reserve", ErrInvalidAmount)
	}
	numerator := new(big.Int).Mul(outAmount, reserveIn)
	if numerator.BitLen() > 256 {
		return nil, fmt.Errorf("%w: input numerator exceeds uint256", ErrInvalidAmount)
	}
	result := new(big.Int).Add(new(big.Int).Div(numerator, denominator), big.NewInt(1))
	if result.BitLen() > 256 {
		return nil, fmt.Errorf("%w: required input exceeds uint256", ErrInvalidAmount)
	}
	return result, nil
}

func CeilDiv(a, b *big.Int) (*big.Int, error) {
	if a == nil || a.Sign() < 0 || a.BitLen() > 256 {
		return nil, fmt.Errorf("%w: ceil division numerator must be non-negative", ErrInvalidAmount)
	}
	if b == nil || b.Sign() <= 0 || b.BitLen() > 256 {
		return nil, fmt.Errorf("%w: ceil division denominator must be positive", ErrInvalidAmount)
	}
	return new(big.Int).Div(new(big.Int).Sub(new(big.Int).Add(a, b), big.NewInt(1)), b), nil
}

func ReservedTokensForPool(supply, phantomQuote, graduationThreshold *big.Int) (*big.Int, error) {
	if supply == nil || supply.Sign() < 0 || supply.BitLen() > 256 ||
		phantomQuote == nil || phantomQuote.Sign() < 0 || phantomQuote.BitLen() > 256 {
		return nil, fmt.Errorf("%w: supply and phantom quote must be non-negative", ErrInvalidAmount)
	}
	if graduationThreshold == nil || graduationThreshold.Sign() <= 0 || graduationThreshold.BitLen() > 256 {
		return nil, fmt.Errorf("%w: graduation threshold must be positive", ErrInvalidAmount)
	}
	denominator := new(big.Int).Add(phantomQuote, graduationThreshold)
	if denominator.BitLen() > 256 {
		return nil, fmt.Errorf("%w: reserve denominator exceeds uint256", ErrInvalidAmount)
	}
	return new(big.Int).Div(new(big.Int).Mul(supply, phantomQuote), denominator), nil
}

func GraduationProgress(raised, threshold *big.Int) (*big.Rat, error) {
	if raised == nil || raised.Sign() < 0 || raised.BitLen() > 256 {
		return nil, fmt.Errorf("%w: raised amount must be non-negative", ErrInvalidAmount)
	}
	if threshold == nil || threshold.Sign() <= 0 || threshold.BitLen() > 256 {
		return nil, fmt.Errorf("%w: graduation threshold must be positive", ErrInvalidAmount)
	}
	return new(big.Rat).SetFrac(raised, threshold), nil
}

func MinOutputWithSlippage(expectedOut *big.Int, slippageBps uint64) (*big.Int, error) {
	if expectedOut == nil || expectedOut.Sign() < 0 || expectedOut.BitLen() > 256 {
		return nil, fmt.Errorf("%w: expected output must be non-negative", ErrInvalidAmount)
	}
	if slippageBps > BPS {
		return nil, fmt.Errorf("%w: slippage bps must not exceed %d", ErrInvalidAmount, BPS)
	}
	result := new(big.Int).Mul(expectedOut, new(big.Int).SetUint64(BPS-slippageBps))
	return result.Div(result, bpsInt), nil
}

// MinTokensOutForBuy converts a quoted fill rate into the minTokensOut value
// expected by the curve. This differs from a plain output haircut when a buy
// is partially filled and only part of quoteIn is spent.
func MinTokensOutForBuy(quoteIn *big.Int, quote QuoteBuyResult, slippageBps uint64) (*big.Int, error) {
	if quoteIn == nil || quoteIn.Sign() < 0 || quoteIn.BitLen() > 256 {
		return nil, errors.New("quote input must fit uint256")
	}
	if quoteIn.Sign() == 0 {
		return nil, errors.New("quote input must be positive")
	}
	if quote.TokensOut == nil || quote.TokensOut.Sign() <= 0 || quote.TokensOut.BitLen() > 256 {
		return nil, errors.New("quoted token output must be a positive uint256")
	}
	if quote.Spent == nil || quote.Spent.Sign() <= 0 || quote.Spent.BitLen() > 256 {
		return nil, errors.New("quoted spend must be a positive uint256")
	}
	if quote.Spent.Cmp(quoteIn) > 0 {
		return nil, errors.New("quoted spend exceeds quote input")
	}
	if slippageBps > BPS {
		return nil, fmt.Errorf("%w: slippage bps must not exceed %d", ErrInvalidAmount, BPS)
	}
	onChainRight := new(big.Int).Mul(quoteIn, quote.TokensOut)
	if onChainRight.BitLen() > 256 {
		return nil, errors.New("quoted values would overflow the on-chain slippage check")
	}
	if slippageBps == BPS {
		return new(big.Int), nil
	}

	numerator := new(big.Int).Mul(quoteIn, quote.TokensOut)
	numerator.Mul(numerator, new(big.Int).SetUint64(BPS-slippageBps))
	denominator := new(big.Int).Mul(quote.Spent, bpsInt)
	result := numerator.Div(numerator, denominator)
	if result.BitLen() > 256 {
		return nil, errors.New("minimum token output exceeds uint256")
	}
	if new(big.Int).Mul(quote.Spent, result).BitLen() > 256 {
		return nil, errors.New("minimum token output would overflow the on-chain slippage check")
	}
	return result, nil
}

func (c *Client) QuoteBuy(ctx context.Context, curve common.Address, quoteIn *big.Int, recipient common.Address, opts *bind.CallOpts) (QuoteBuyResult, error) {
	if recipient == (common.Address{}) {
		return QuoteBuyResult{}, fmt.Errorf("%w: recipient must not be the zero address", ErrInvalidQuote)
	}
	callOpts, err := c.snapshotCallOpts(ctx, opts)
	if err != nil {
		return QuoteBuyResult{}, err
	}
	contract := c.curve(curve)
	var reserves CurveReserves
	var sellable, feeBps, creatorTaxBps, snipeBps *big.Int
	var graduated bool
	err = parallelCalls(callOpts.Context,
		func(taskCtx context.Context) error {
			out, err := c.call(taskCtx, contract, taskCallOpts(callOpts, taskCtx), "getReserves")
			if err == nil {
				reserves = CurveReserves{QuoteReserve: cloneBig(out[0].(*big.Int)), TokenReserve: cloneBig(out[1].(*big.Int))}
			}
			return err
		},
		c.callBigTask(contract, callOpts, &sellable, "sellableTokens"),
		c.callBigTask(contract, callOpts, &feeBps, "feeBps"),
		c.callBigTask(contract, callOpts, &creatorTaxBps, "creatorTaxBps"),
		c.callBigTask(contract, callOpts, &snipeBps, "currentSnipeTaxBps", recipient),
		c.callBoolTask(contract, callOpts, &graduated, "graduated"),
	)
	if err != nil {
		return QuoteBuyResult{}, err
	}
	if graduated || sellable.Sign() == 0 {
		return QuoteBuyResult{}, ErrCurveClosed
	}
	return QuoteBuyFromState(reserves, sellable, quoteIn, feeBps, creatorTaxBps, snipeBps)
}

func QuoteBuyFromState(reserves CurveReserves, sellable, quoteIn, feeBps, creatorTaxBps, rawSnipeBps *big.Int) (QuoteBuyResult, error) {
	if err := validateTradeState(reserves, quoteIn, feeBps, creatorTaxBps); err != nil {
		return QuoteBuyResult{}, err
	}
	if sellable == nil || sellable.Sign() <= 0 || sellable.BitLen() > 256 {
		return QuoteBuyResult{}, fmt.Errorf("%w: sellable tokens must be positive", ErrInvalidQuote)
	}
	if sellable.Cmp(reserves.TokenReserve) >= 0 {
		return QuoteBuyResult{}, fmt.Errorf("%w: sellable tokens must be below token reserve", ErrInvalidQuote)
	}
	if rawSnipeBps == nil || rawSnipeBps.Sign() < 0 || rawSnipeBps.BitLen() > 256 {
		return QuoteBuyResult{}, fmt.Errorf("%w: snipe tax bps must be non-negative", ErrInvalidQuote)
	}

	spent := cloneBig(quoteIn)
	snipeBps := cloneBig(rawSnipeBps)
	if snipeBps.Sign() > 0 {
		maxSnipeBps := new(big.Int).Sub(bpsInt, feeBps)
		maxSnipeBps.Sub(maxSnipeBps, creatorTaxBps)
		maxSnipeBps.Sub(maxSnipeBps, onePctBPSInt)
		if maxSnipeBps.Sign() < 0 {
			return QuoteBuyResult{}, fmt.Errorf("%w: fee bps leave less than one percent of spend", ErrInvalidQuote)
		}
		if snipeBps.Cmp(maxSnipeBps) > 0 {
			snipeBps = maxSnipeBps
		}
	}

	fee, err := bpsAmount(spent, feeBps)
	if err != nil {
		return QuoteBuyResult{}, err
	}
	tax, err := bpsAmount(spent, creatorTaxBps)
	if err != nil {
		return QuoteBuyResult{}, err
	}
	snipeTax, err := bpsAmount(spent, snipeBps)
	if err != nil {
		return QuoteBuyResult{}, err
	}
	net := new(big.Int).Sub(spent, fee)
	net.Sub(net, tax)
	net.Sub(net, snipeTax)
	if net.Sign() <= 0 {
		return QuoteBuyResult{}, fmt.Errorf("%w: fees consume the quote input", ErrInvalidQuote)
	}
	tokensOut, err := AmountOut(net, reserves.QuoteReserve, reserves.TokenReserve)
	if err != nil {
		return QuoteBuyResult{}, err
	}
	if tokensOut.Sign() == 0 {
		return QuoteBuyResult{}, ErrUnquotableTrade
	}

	if tokensOut.Cmp(sellable) > 0 {
		tokensOut = cloneBig(sellable)
		netNeeded, err := AmountIn(sellable, reserves.QuoteReserve, reserves.TokenReserve)
		if err != nil {
			return QuoteBuyResult{}, err
		}
		totalFeeBps := new(big.Int).Add(feeBps, creatorTaxBps)
		totalFeeBps.Add(totalFeeBps, snipeBps)
		grossed, err := mulDivCeil(netNeeded, bpsInt, new(big.Int).Sub(bpsInt, totalFeeBps))
		if err != nil {
			return QuoteBuyResult{}, err
		}
		if grossed.Cmp(quoteIn) < 0 {
			spent = grossed
		} else {
			spent = cloneBig(quoteIn)
		}
		fee, err = bpsAmount(spent, feeBps)
		if err != nil {
			return QuoteBuyResult{}, err
		}
		tax, err = bpsAmount(spent, creatorTaxBps)
		if err != nil {
			return QuoteBuyResult{}, err
		}
		snipeTax, err = bpsAmount(spent, snipeBps)
		if err != nil {
			return QuoteBuyResult{}, err
		}
	}

	refund := new(big.Int).Sub(quoteIn, spent)
	if refund.Sign() < 0 {
		refund.SetInt64(0)
	}
	return QuoteBuyResult{
		TokensOut:     tokensOut,
		Spent:         spent,
		Refund:        refund,
		Fee:           fee,
		Tax:           tax,
		SnipeTax:      snipeTax,
		FeeBps:        cloneBig(feeBps),
		CreatorTaxBps: cloneBig(creatorTaxBps),
		SnipeTaxBps:   snipeBps,
	}, nil
}

// ApplyBuyToState advances a local CurveState using a QuoteBuyFromState
// result. It mirrors the pricing and fee-accounting writes in buy().
func ApplyBuyToState(state CurveState, quote QuoteBuyResult, policy FeePolicy) (CurveState, error) {
	values := []*big.Int{quote.TokensOut, quote.Spent, quote.Fee, quote.Tax, quote.SnipeTax, state.TrackedQuote, state.TrackedTokens, state.QuoteFeeBalance, state.CreatorTaxBalance, state.BuybackQuoteBalance, state.Reserves.QuoteReserve, state.Reserves.TokenReserve, state.SellableTokens}
	for _, value := range values {
		if value == nil || value.Sign() < 0 || value.BitLen() > 256 {
			return CurveState{}, fmt.Errorf("%w: incomplete buy state", ErrInvalidQuote)
		}
	}
	if quote.TokensOut.Sign() == 0 || quote.Spent.Sign() == 0 || quote.TokensOut.Cmp(state.SellableTokens) > 0 {
		return CurveState{}, fmt.Errorf("%w: invalid buy result", ErrInvalidQuote)
	}
	result := cloneCurveState(state)
	feeBucket := new(big.Int).Add(quote.Fee, quote.SnipeTax)
	result.TrackedQuote.Add(result.TrackedQuote, quote.Spent)
	result.TrackedTokens.Sub(result.TrackedTokens, quote.TokensOut)
	result.QuoteFeeBalance.Add(result.QuoteFeeBalance, feeBucket)
	result.CreatorTaxBalance.Add(result.CreatorTaxBalance, quote.Tax)
	if result.BuybackEnabled && feeBucket.Sign() > 0 {
		protocol := new(big.Int).Mul(feeBucket, new(big.Int).SetUint64(uint64(policy.ProtocolFeeShareBps)))
		protocol.Div(protocol, bpsInt)
		creatorSlice := new(big.Int).Sub(feeBucket, protocol)
		buyback := new(big.Int).Mul(creatorSlice, new(big.Int).SetUint64(uint64(policy.BuybackBurnBps)))
		buyback.Div(buyback, bpsInt)
		result.BuybackQuoteBalance.Add(result.BuybackQuoteBalance, buyback)
	}
	result.Reserves.QuoteReserve.Add(result.Reserves.QuoteReserve, quote.Spent)
	result.Reserves.QuoteReserve.Sub(result.Reserves.QuoteReserve, feeBucket)
	result.Reserves.QuoteReserve.Sub(result.Reserves.QuoteReserve, quote.Tax)
	result.Reserves.TokenReserve.Sub(result.Reserves.TokenReserve, quote.TokensOut)
	result.SellableTokens.Sub(result.SellableTokens, quote.TokensOut)
	result.RealQuoteReserve = new(big.Int).Sub(result.TrackedQuote, result.QuoteFeeBalance)
	result.RealQuoteReserve.Sub(result.RealQuoteReserve, result.CreatorTaxBalance)
	result.ReadyToGraduate = result.SellableTokens.Sign() == 0
	return result, nil
}

func cloneCurveState(state CurveState) CurveState {
	state.Reserves.QuoteReserve = cloneBig(state.Reserves.QuoteReserve)
	state.Reserves.TokenReserve = cloneBig(state.Reserves.TokenReserve)
	state.PhantomQuote = cloneBig(state.PhantomQuote)
	state.RealQuoteReserve = cloneBig(state.RealQuoteReserve)
	state.GraduationThreshold = cloneBig(state.GraduationThreshold)
	state.SellableTokens = cloneBig(state.SellableTokens)
	state.ReservedTokens = cloneBig(state.ReservedTokens)
	state.TrackedQuote = cloneBig(state.TrackedQuote)
	state.TrackedTokens = cloneBig(state.TrackedTokens)
	state.QuoteFeeBalance = cloneBig(state.QuoteFeeBalance)
	state.BuybackQuoteBalance = cloneBig(state.BuybackQuoteBalance)
	state.CreatorTaxBalance = cloneBig(state.CreatorTaxBalance)
	state.FeeBps = cloneBig(state.FeeBps)
	state.CreatorTaxBps = cloneBig(state.CreatorTaxBps)
	return state
}

func mulDivCeil(a, b, denominator *big.Int) (*big.Int, error) {
	if err := validateUint256(a, "multiplicand"); err != nil {
		return nil, err
	}
	if err := validateUint256(b, "multiplier"); err != nil {
		return nil, err
	}
	if denominator == nil || denominator.Sign() <= 0 || denominator.BitLen() > 256 {
		return nil, fmt.Errorf("%w: denominator must be a positive uint256", ErrInvalidAmount)
	}
	product := new(big.Int).Mul(a, b)
	quotient, remainder := new(big.Int).QuoRem(product, denominator, new(big.Int))
	if remainder.Sign() != 0 {
		quotient.Add(quotient, big.NewInt(1))
	}
	if quotient.BitLen() > 256 {
		return nil, fmt.Errorf("%w: multiplication result exceeds uint256", ErrInvalidAmount)
	}
	return quotient, nil
}

func (c *Client) QuoteSell(ctx context.Context, curve common.Address, tokensIn *big.Int, opts *bind.CallOpts) (QuoteSellResult, error) {
	callOpts, err := c.snapshotCallOpts(ctx, opts)
	if err != nil {
		return QuoteSellResult{}, err
	}
	contract := c.curve(curve)
	var reserves CurveReserves
	var feeBps, creatorTaxBps *big.Int
	var ready, graduated bool
	err = parallelCalls(callOpts.Context,
		func(taskCtx context.Context) error {
			out, err := c.call(taskCtx, contract, taskCallOpts(callOpts, taskCtx), "getReserves")
			if err == nil {
				reserves = CurveReserves{QuoteReserve: cloneBig(out[0].(*big.Int)), TokenReserve: cloneBig(out[1].(*big.Int))}
			}
			return err
		},
		c.callBigTask(contract, callOpts, &feeBps, "feeBps"),
		c.callBigTask(contract, callOpts, &creatorTaxBps, "creatorTaxBps"),
		c.callBoolTask(contract, callOpts, &ready, "readyToGraduate"),
		c.callBoolTask(contract, callOpts, &graduated, "graduated"),
	)
	if err != nil {
		return QuoteSellResult{}, err
	}
	if ready || graduated {
		return QuoteSellResult{}, ErrCurveClosed
	}
	return QuoteSellFromState(reserves, tokensIn, feeBps, creatorTaxBps)
}

func QuoteSellFromState(reserves CurveReserves, tokensIn, feeBps, creatorTaxBps *big.Int) (QuoteSellResult, error) {
	if err := validateTradeState(reserves, tokensIn, feeBps, creatorTaxBps); err != nil {
		return QuoteSellResult{}, err
	}
	gross, err := AmountOut(tokensIn, reserves.TokenReserve, reserves.QuoteReserve)
	if err != nil {
		return QuoteSellResult{}, err
	}
	if gross.Sign() == 0 {
		return QuoteSellResult{}, ErrUnquotableTrade
	}
	fee, err := bpsAmount(gross, feeBps)
	if err != nil {
		return QuoteSellResult{}, err
	}
	tax, err := bpsAmount(gross, creatorTaxBps)
	if err != nil {
		return QuoteSellResult{}, err
	}
	quoteOut := new(big.Int).Sub(gross, fee)
	quoteOut.Sub(quoteOut, tax)
	if quoteOut.Sign() <= 0 {
		return QuoteSellResult{}, fmt.Errorf("%w: fees consume the quote output", ErrInvalidQuote)
	}
	return QuoteSellResult{
		QuoteOut:      quoteOut,
		GrossQuoteOut: gross,
		Fee:           fee,
		Tax:           tax,
		FeeBps:        cloneBig(feeBps),
		CreatorTaxBps: cloneBig(creatorTaxBps),
	}, nil
}

func bpsAmount(amount, bps *big.Int) (*big.Int, error) {
	result := new(big.Int).Mul(amount, bps)
	if result.BitLen() > 256 {
		return nil, fmt.Errorf("%w: basis-point multiplication exceeds uint256", ErrInvalidQuote)
	}
	return result.Div(result, bpsInt), nil
}

func validateTradeState(reserves CurveReserves, amount, feeBps, creatorTaxBps *big.Int) error {
	if amount == nil || amount.Sign() <= 0 || amount.BitLen() > 256 {
		return fmt.Errorf("%w: trade input must be a positive uint256", ErrInvalidQuote)
	}
	if reserves.QuoteReserve == nil || reserves.QuoteReserve.Sign() <= 0 || reserves.QuoteReserve.BitLen() > 256 ||
		reserves.TokenReserve == nil || reserves.TokenReserve.Sign() <= 0 || reserves.TokenReserve.BitLen() > 256 {
		return fmt.Errorf("%w: reserves must be positive uint256 values", ErrInvalidQuote)
	}
	if feeBps == nil || feeBps.Sign() < 0 || feeBps.BitLen() > 256 ||
		creatorTaxBps == nil || creatorTaxBps.Sign() < 0 || creatorTaxBps.BitLen() > 256 {
		return fmt.Errorf("%w: fee bps must be non-negative", ErrInvalidQuote)
	}
	totalFeeBps := new(big.Int).Add(feeBps, creatorTaxBps)
	if totalFeeBps.Cmp(bpsInt) >= 0 {
		return fmt.Errorf("%w: combined fee bps must be below %d", ErrInvalidQuote, BPS)
	}
	return nil
}

func cloneBig(v *big.Int) *big.Int {
	if v == nil {
		return new(big.Int)
	}
	return new(big.Int).Set(v)
}
