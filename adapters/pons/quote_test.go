package pons

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
)

var benchmarkQuoteBuyResult QuoteBuyResult

type quoteCaller struct {
	responses map[string][]byte
}

func (*quoteCaller) CodeAt(context.Context, common.Address, *big.Int) ([]byte, error) {
	return []byte{1}, nil
}

func (c *quoteCaller) CallContract(_ context.Context, call ethereum.CallMsg, _ *big.Int) ([]byte, error) {
	if len(call.Data) < 4 {
		return nil, errors.New("missing method selector")
	}
	response, ok := c.responses[string(call.Data[:4])]
	if !ok {
		return nil, errors.New("unexpected method selector")
	}
	return response, nil
}

func curveCallResponse(t *testing.T, method string, values ...interface{}) (string, []byte) {
	t.Helper()
	abiMethod := CurveABI.Methods[method]
	data, err := abiMethod.Outputs.Pack(values...)
	if err != nil {
		t.Fatal(err)
	}
	return string(abiMethod.ID), data
}

func TestQuoteBuyAppliesFeesBeforeCurve(t *testing.T) {
	reserves := CurveReserves{
		QuoteReserve: big.NewInt(1_000_000),
		TokenReserve: big.NewInt(10_000_000),
	}
	got, err := QuoteBuyFromState(reserves, big.NewInt(9_000_000), big.NewInt(100_000), big.NewInt(100), big.NewInt(50), big.NewInt(0))
	if err != nil {
		t.Fatal(err)
	}
	if got.Fee.Cmp(big.NewInt(1_000)) != 0 {
		t.Fatalf("fee = %s", got.Fee)
	}
	if got.Tax.Cmp(big.NewInt(500)) != 0 {
		t.Fatalf("tax = %s", got.Tax)
	}
	if got.TokensOut.Cmp(big.NewInt(896_677)) != 0 {
		t.Fatalf("tokensOut = %s", got.TokensOut)
	}
}

func TestQuoteBuyClampsToSellableAndRefunds(t *testing.T) {
	reserves := CurveReserves{
		QuoteReserve: big.NewInt(1_000_000),
		TokenReserve: big.NewInt(10_000_000),
	}
	got, err := QuoteBuyFromState(reserves, big.NewInt(100_000), big.NewInt(9_000_000), big.NewInt(0), big.NewInt(0), big.NewInt(0))
	if err != nil {
		t.Fatal(err)
	}
	if got.TokensOut.Cmp(big.NewInt(100_000)) != 0 {
		t.Fatalf("tokensOut = %s", got.TokensOut)
	}
	if got.Refund.Sign() <= 0 {
		t.Fatalf("expected positive refund, got %s", got.Refund)
	}
}

func TestQuoteBuyClampNeverSpendsMoreThanInput(t *testing.T) {
	reserves := CurveReserves{
		QuoteReserve: big.NewInt(100),
		TokenReserve: big.NewInt(230),
	}
	quoteIn := big.NewInt(110)
	got, err := QuoteBuyFromState(reserves, big.NewInt(111), quoteIn, big.NewInt(600), big.NewInt(900), big.NewInt(0))
	if err != nil {
		t.Fatal(err)
	}
	if got.Spent.Cmp(quoteIn) > 0 {
		t.Fatalf("spent %s exceeds input %s", got.Spent, quoteIn)
	}
}

func TestMinTokensOutForBuyPreservesPartialFillRate(t *testing.T) {
	quoteIn := big.NewInt(9_000_000)
	quote := QuoteBuyResult{
		TokensOut: big.NewInt(100_000),
		Spent:     big.NewInt(10_102),
	}

	got, err := MinTokensOutForBuy(quoteIn, quote, 500)
	if err != nil {
		t.Fatal(err)
	}
	want := new(big.Int).Mul(quoteIn, quote.TokensOut)
	want.Mul(want, big.NewInt(9_500))
	want.Div(want, new(big.Int).Mul(quote.Spent, big.NewInt(10_000)))
	if got.Cmp(want) != 0 {
		t.Fatalf("minTokensOut = %s, want %s", got, want)
	}
	simple, err := MinOutputWithSlippage(quote.TokensOut, 500)
	if err != nil {
		t.Fatal(err)
	}
	if got.Cmp(simple) <= 0 {
		t.Fatalf("partial-fill price bound was not scaled: %s", got)
	}
}

func TestMinTokensOutForBuyMatchesSimpleSlippageForFullFill(t *testing.T) {
	quoteIn := big.NewInt(100_000)
	quote := QuoteBuyResult{TokensOut: big.NewInt(896_677), Spent: cloneBig(quoteIn)}
	got, err := MinTokensOutForBuy(quoteIn, quote, 500)
	if err != nil {
		t.Fatal(err)
	}
	want, err := MinOutputWithSlippage(quote.TokensOut, 500)
	if err != nil {
		t.Fatal(err)
	}
	if got.Cmp(want) != 0 {
		t.Fatalf("minTokensOut = %s, want %s", got, want)
	}
}

func TestMinTokensOutForBuyRejectsImpossibleQuoteState(t *testing.T) {
	overflow := new(big.Int).Lsh(big.NewInt(1), 256)
	maxUint256 := new(big.Int).Sub(new(big.Int).Set(overflow), big.NewInt(1))
	tests := []struct {
		quoteIn *big.Int
		quote   QuoteBuyResult
	}{
		{new(big.Int), QuoteBuyResult{TokensOut: big.NewInt(1), Spent: big.NewInt(1)}},
		{big.NewInt(1), QuoteBuyResult{TokensOut: new(big.Int), Spent: big.NewInt(1)}},
		{big.NewInt(1), QuoteBuyResult{TokensOut: big.NewInt(1), Spent: big.NewInt(2)}},
		{overflow, QuoteBuyResult{TokensOut: big.NewInt(1), Spent: big.NewInt(1)}},
		{big.NewInt(1), QuoteBuyResult{TokensOut: overflow, Spent: big.NewInt(1)}},
		{big.NewInt(1), QuoteBuyResult{TokensOut: big.NewInt(1), Spent: overflow}},
		{maxUint256, QuoteBuyResult{TokensOut: big.NewInt(2), Spent: big.NewInt(1)}},
	}
	for i, test := range tests {
		if _, err := MinTokensOutForBuy(test.quoteIn, test.quote, 100); err == nil {
			t.Fatalf("case %d: expected invalid quote error", i)
		}
	}
}

func TestQuotesRejectValuesOutsideUint256(t *testing.T) {
	overflow := new(big.Int).Lsh(big.NewInt(1), 256)
	validReserves := CurveReserves{QuoteReserve: big.NewInt(1), TokenReserve: big.NewInt(2)}
	for name, reserves := range map[string]CurveReserves{
		"quote reserve": {QuoteReserve: overflow, TokenReserve: big.NewInt(2)},
		"token reserve": {QuoteReserve: big.NewInt(1), TokenReserve: overflow},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := QuoteSellFromState(reserves, big.NewInt(1), new(big.Int), new(big.Int)); !errors.Is(err, ErrInvalidQuote) {
				t.Fatalf("expected ErrInvalidQuote, got %v", err)
			}
		})
	}
	if _, err := QuoteSellFromState(validReserves, overflow, new(big.Int), new(big.Int)); !errors.Is(err, ErrInvalidQuote) {
		t.Fatalf("overflow input: expected ErrInvalidQuote, got %v", err)
	}
}

func TestQuoteSellFeesComeOffOutput(t *testing.T) {
	reserves := CurveReserves{
		QuoteReserve: big.NewInt(1_000_000),
		TokenReserve: big.NewInt(10_000_000),
	}
	got, err := QuoteSellFromState(reserves, big.NewInt(100_000), big.NewInt(100), big.NewInt(50))
	if err != nil {
		t.Fatal(err)
	}
	if got.GrossQuoteOut.Cmp(big.NewInt(9_900)) != 0 {
		t.Fatalf("gross = %s", got.GrossQuoteOut)
	}
	if got.QuoteOut.Cmp(big.NewInt(9_752)) != 0 {
		t.Fatalf("quoteOut = %s", got.QuoteOut)
	}
}

func TestQuoteBuyRejectsClosedCurveState(t *testing.T) {
	reserves := CurveReserves{QuoteReserve: big.NewInt(1_000_000), TokenReserve: big.NewInt(10_000_000)}
	_, err := QuoteBuyFromState(reserves, new(big.Int), big.NewInt(100_000), big.NewInt(100), big.NewInt(50), new(big.Int))
	if !errors.Is(err, ErrInvalidQuote) {
		t.Fatalf("expected ErrInvalidQuote, got %v", err)
	}
}

func TestQuotesRejectZeroInputAndZeroOutput(t *testing.T) {
	reserves := CurveReserves{QuoteReserve: big.NewInt(1_000_000), TokenReserve: big.NewInt(10_000_000)}
	if _, err := QuoteSellFromState(reserves, new(big.Int), big.NewInt(100), big.NewInt(50)); !errors.Is(err, ErrInvalidQuote) {
		t.Fatalf("zero sell input: expected ErrInvalidQuote, got %v", err)
	}

	tinyReserves := CurveReserves{QuoteReserve: big.NewInt(1_000_000_000), TokenReserve: big.NewInt(2)}
	if _, err := QuoteBuyFromState(tinyReserves, big.NewInt(1), big.NewInt(1), new(big.Int), new(big.Int), new(big.Int)); !errors.Is(err, ErrUnquotableTrade) {
		t.Fatalf("zero buy output: expected ErrUnquotableTrade, got %v", err)
	}
}

func TestArithmeticHelpersRejectInvalidInputs(t *testing.T) {
	invalid := []*big.Int{nil, big.NewInt(-1)}
	for _, value := range invalid {
		if _, err := AmountOut(value, big.NewInt(1), big.NewInt(1)); !errors.Is(err, ErrInvalidAmount) {
			t.Fatalf("AmountOut(%v): expected ErrInvalidAmount, got %v", value, err)
		}
		if _, err := AmountIn(value, big.NewInt(1), big.NewInt(1)); !errors.Is(err, ErrInvalidAmount) {
			t.Fatalf("AmountIn(%v): expected ErrInvalidAmount, got %v", value, err)
		}
		if _, err := CeilDiv(value, big.NewInt(1)); !errors.Is(err, ErrInvalidAmount) {
			t.Fatalf("CeilDiv(%v): expected ErrInvalidAmount, got %v", value, err)
		}
		if _, err := ReservedTokensForPool(value, big.NewInt(1), big.NewInt(1)); !errors.Is(err, ErrInvalidAmount) {
			t.Fatalf("ReservedTokensForPool(%v): expected ErrInvalidAmount, got %v", value, err)
		}
		if _, err := GraduationProgress(value, big.NewInt(1)); !errors.Is(err, ErrInvalidAmount) {
			t.Fatalf("GraduationProgress(%v): expected ErrInvalidAmount, got %v", value, err)
		}
		if _, err := MinOutputWithSlippage(value, 100); !errors.Is(err, ErrInvalidAmount) {
			t.Fatalf("MinOutputWithSlippage(%v): expected ErrInvalidAmount, got %v", value, err)
		}
	}
	if _, err := AmountIn(new(big.Int), big.NewInt(1), big.NewInt(1)); !errors.Is(err, ErrInvalidAmount) {
		t.Fatalf("zero AmountIn: expected ErrInvalidAmount, got %v", err)
	}
	if _, err := MinOutputWithSlippage(big.NewInt(1), BPS+1); !errors.Is(err, ErrInvalidAmount) {
		t.Fatalf("excessive slippage: expected ErrInvalidAmount, got %v", err)
	}
	if _, err := MinTokensOutForBuy(big.NewInt(1), QuoteBuyResult{TokensOut: big.NewInt(1), Spent: big.NewInt(1)}, BPS+1); !errors.Is(err, ErrInvalidAmount) {
		t.Fatalf("buy slippage: expected ErrInvalidAmount, got %v", err)
	}
}

func TestArithmeticHelpersRejectValuesOutsideUint256(t *testing.T) {
	overflow := new(big.Int).Lsh(big.NewInt(1), 256)
	checks := []func() error{
		func() error { _, err := AmountOut(overflow, big.NewInt(1), big.NewInt(1)); return err },
		func() error { _, err := AmountOut(big.NewInt(1), overflow, big.NewInt(1)); return err },
		func() error { _, err := AmountIn(big.NewInt(1), overflow, big.NewInt(2)); return err },
		func() error { _, err := CeilDiv(overflow, big.NewInt(1)); return err },
		func() error { _, err := ReservedTokensForPool(overflow, big.NewInt(1), big.NewInt(1)); return err },
		func() error { _, err := GraduationProgress(overflow, big.NewInt(1)); return err },
		func() error { _, err := MinOutputWithSlippage(overflow, 100); return err },
	}
	for i, check := range checks {
		if err := check(); !errors.Is(err, ErrInvalidAmount) {
			t.Fatalf("check %d: expected ErrInvalidAmount, got %v", i, err)
		}
	}

	max := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
	if _, err := AmountIn(new(big.Int).Sub(max, big.NewInt(1)), max, max); !errors.Is(err, ErrInvalidAmount) {
		t.Fatalf("expected uint256 result overflow, got %v", err)
	}
}

func TestQuoteBuyRejectsOversizedStateFields(t *testing.T) {
	overflow := new(big.Int).Lsh(big.NewInt(1), 256)
	reserves := CurveReserves{QuoteReserve: big.NewInt(1_000), TokenReserve: big.NewInt(10_000)}
	tests := []struct {
		name     string
		sellable *big.Int
		fee      *big.Int
		creator  *big.Int
		snipe    *big.Int
	}{
		{"sellable", overflow, big.NewInt(1), big.NewInt(1), big.NewInt(1)},
		{"fee", big.NewInt(100), overflow, big.NewInt(1), big.NewInt(1)},
		{"creator tax", big.NewInt(100), big.NewInt(1), overflow, big.NewInt(1)},
		{"snipe tax", big.NewInt(100), big.NewInt(1), big.NewInt(1), overflow},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := QuoteBuyFromState(reserves, tc.sellable, big.NewInt(10), tc.fee, tc.creator, tc.snipe)
			if !errors.Is(err, ErrInvalidQuote) {
				t.Fatalf("expected ErrInvalidQuote, got %v", err)
			}
		})
	}
}

func TestQuoteMathProperties(t *testing.T) {
	for i := uint64(1); i <= 1_000; i++ {
		reserveIn := new(big.Int).SetUint64((i*7_919)%1_000_000 + 2)
		reserveOut := new(big.Int).SetUint64((i*104_729)%1_000_000 + 2)
		in := new(big.Int).SetUint64((i*65_537)%1_000_000 + 1)

		out, err := AmountOut(in, reserveIn, reserveOut)
		if err != nil {
			t.Fatal(err)
		}
		if out.Sign() < 0 || out.Cmp(reserveOut) >= 0 {
			t.Fatalf("iteration %d: output %s outside [0, %s)", i, out, reserveOut)
		}
		if out.Sign() == 0 {
			continue
		}

		required, err := AmountIn(out, reserveIn, reserveOut)
		if err != nil {
			t.Fatal(err)
		}
		settled, err := AmountOut(required, reserveIn, reserveOut)
		if err != nil {
			t.Fatal(err)
		}
		if settled.Cmp(out) < 0 {
			t.Fatalf("iteration %d: inverse input %s yields %s below %s", i, required, settled, out)
		}
	}
}

func TestMulDivCeilSupportsWideIntermediate(t *testing.T) {
	max := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
	got, err := mulDivCeil(max, big.NewInt(10_000), big.NewInt(10_000))
	if err != nil {
		t.Fatal(err)
	}
	if got.Cmp(max) != 0 {
		t.Fatalf("result = %s, want %s", got, max)
	}
}

func TestQuoteMathRejectsUint256IntermediateOverflow(t *testing.T) {
	max := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
	half := new(big.Int).Rsh(new(big.Int).Set(max), 1)

	if _, err := AmountOut(max, big.NewInt(1), big.NewInt(2)); !errors.Is(err, ErrInvalidAmount) {
		t.Fatalf("AmountOut denominator overflow: %v", err)
	}
	if _, err := AmountOut(half, big.NewInt(1), big.NewInt(3)); !errors.Is(err, ErrInvalidAmount) {
		t.Fatalf("AmountOut numerator overflow: %v", err)
	}
	if _, err := AmountIn(half, half, max); !errors.Is(err, ErrInvalidAmount) {
		t.Fatalf("AmountIn numerator overflow: %v", err)
	}
	if _, err := ReservedTokensForPool(big.NewInt(1), max, big.NewInt(1)); !errors.Is(err, ErrInvalidAmount) {
		t.Fatalf("ReservedTokensForPool denominator overflow: %v", err)
	}

	reserves := CurveReserves{QuoteReserve: big.NewInt(1), TokenReserve: big.NewInt(2)}
	if _, err := QuoteBuyFromState(reserves, big.NewInt(1), half, big.NewInt(3), new(big.Int), new(big.Int)); !errors.Is(err, ErrInvalidQuote) {
		t.Fatalf("quote fee multiplication overflow: %v", err)
	}
	quote := QuoteBuyResult{TokensOut: half, Spent: half}
	if _, err := MinTokensOutForBuy(max, quote, 0); err == nil {
		t.Fatal("expected on-chain slippage multiplication overflow")
	}
	if _, err := MinTokensOutForBuy(max, quote, BPS); err == nil {
		t.Fatal("expected on-chain slippage multiplication overflow at full slippage")
	}
}

func TestClientQuotesRejectClosedCurve(t *testing.T) {
	responses := make(map[string][]byte)
	add := func(method string, values ...interface{}) {
		selector, data := curveCallResponse(t, method, values...)
		responses[selector] = data
	}
	add("getReserves", big.NewInt(1_000_000), big.NewInt(10_000_000))
	add("sellableTokens", big.NewInt(9_000_000))
	add("feeBps", big.NewInt(100))
	add("creatorTaxBps", big.NewInt(50))
	add("currentSnipeTaxBps", new(big.Int))
	add("readyToGraduate", true)
	add("graduated", false)

	client := &Client{caller: &quoteCaller{responses: responses}}
	curve := common.HexToAddress("0x0000000000000000000000000000000000000001")
	if _, err := client.QuoteSell(context.Background(), curve, big.NewInt(100_000), nil); !errors.Is(err, ErrCurveClosed) {
		t.Fatalf("sell: expected ErrCurveClosed, got %v", err)
	}

	add("readyToGraduate", false)
	add("graduated", true)
	if _, err := client.QuoteBuy(context.Background(), curve, big.NewInt(100_000), curve, nil); !errors.Is(err, ErrCurveClosed) {
		t.Fatalf("buy: expected ErrCurveClosed, got %v", err)
	}
}

func TestQuoteBuyRejectsZeroRecipientBeforeRPC(t *testing.T) {
	client := &Client{caller: &quoteCaller{responses: map[string][]byte{}}}
	curve := common.HexToAddress("0x0000000000000000000000000000000000000001")
	_, err := client.QuoteBuy(context.Background(), curve, big.NewInt(100_000), common.Address{}, nil)
	if !errors.Is(err, ErrInvalidQuote) {
		t.Fatalf("expected ErrInvalidQuote, got %v", err)
	}
}

func BenchmarkQuoteBuyFromState(b *testing.B) {
	reserves := CurveReserves{
		QuoteReserve: big.NewInt(5_000_000_000),
		TokenReserve: big.NewInt(1_000_000_000_000),
	}
	sellable := big.NewInt(800_000_000_000)
	quoteIn := big.NewInt(10_000_000)
	feeBps := big.NewInt(100)
	creatorTaxBps := big.NewInt(50)
	snipeBps := big.NewInt(200)

	if _, err := QuoteBuyFromState(reserves, sellable, quoteIn, feeBps, creatorTaxBps, snipeBps); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := QuoteBuyFromState(reserves, sellable, quoteIn, feeBps, creatorTaxBps, snipeBps)
		if err != nil {
			b.Fatal(err)
		}
		benchmarkQuoteBuyResult = result
	}
}
