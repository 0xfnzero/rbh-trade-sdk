package rbhtrade

import (
	"errors"
	"math/big"
	"testing"

	core "github.com/daoleno/uniswap-sdk-core/entities"
	v3constants "github.com/daoleno/uniswapv3-sdk/constants"
	v3 "github.com/daoleno/uniswapv3-sdk/entities"
	"github.com/ethereum/go-ethereum/common"
)

func TestQuoteV4ExactInputFromFullRangeEventState(t *testing.T) {
	token := common.HexToAddress("0x1000000000000000000000000000000000000001")
	state := V4PoolState{
		PoolKey:      PoolKey{Currency0: common.Address{}, Currency1: token, Fee: 3000, TickSpacing: 60},
		SqrtPriceX96: new(big.Int).Lsh(big.NewInt(1), 96), Tick: 0,
		Ranges: []V4LiquidityRange{{TickLower: -887220, TickUpper: 887220, Delta: big.NewInt(1_000_000_000)}},
	}
	gross, err := QuoteV4ExactInput(state, common.Address{}, big.NewInt(10_000))
	if err != nil || gross.Sign() <= 0 || gross.Cmp(big.NewInt(10_000)) >= 0 {
		t.Fatalf("gross quote = %v, %v", gross, err)
	}
	state.HookFeeBPS, state.CreatorTaxBPS = 100, 200
	net, err := QuoteV4ExactInput(state, common.Address{}, big.NewInt(10_000))
	if err != nil || net.Cmp(gross) >= 0 {
		t.Fatalf("post-hook quote = %v, gross %v, err %v", net, gross, err)
	}
	want := new(big.Int).Sub(gross, new(big.Int).Div(new(big.Int).Mul(gross, big.NewInt(100)), big.NewInt(10_000)))
	want.Sub(want, new(big.Int).Div(new(big.Int).Mul(gross, big.NewInt(200)), big.NewInt(10_000)))
	if net.Cmp(want) != 0 {
		t.Fatalf("post-hook quote = %s, want %s", net, want)
	}
}

func TestQuoteV4ExactInputRejectsMissingLiquidity(t *testing.T) {
	if _, err := QuoteV4ExactInput(V4PoolState{}, common.Address{}, big.NewInt(1)); err == nil {
		t.Fatal("missing state was accepted")
	}
}

func TestQuoteV4ExactInputRejectsOutOfRangeStateWithoutPanic(t *testing.T) {
	token := common.HexToAddress("0x1000000000000000000000000000000000000001")
	state := V4PoolState{
		PoolKey:      PoolKey{Currency0: common.Address{}, Currency1: token, Fee: 3_000, TickSpacing: 1},
		SqrtPriceX96: new(big.Int).Lsh(big.NewInt(1), 96), Tick: 0,
		Ranges: []V4LiquidityRange{{TickLower: -887273, TickUpper: 887272, Delta: big.NewInt(1)}},
	}
	if _, err := QuoteV4ExactInput(state, common.Address{}, big.NewInt(1)); !errors.Is(err, ErrInvalidV4State) {
		t.Fatalf("out-of-range tick error=%v", err)
	}
	state.Ranges[0] = V4LiquidityRange{TickLower: -1, TickUpper: 1, Delta: new(big.Int).Lsh(big.NewInt(1), 128)}
	if _, err := QuoteV4ExactInput(state, common.Address{}, big.NewInt(1)); !errors.Is(err, ErrInvalidV4State) {
		t.Fatalf("uint128 overflow error=%v", err)
	}
}

func TestQuoteV4ExactInputNetsHistoricalLiquidityDeltas(t *testing.T) {
	token := common.HexToAddress("0x1000000000000000000000000000000000000001")
	base := V4PoolState{
		PoolKey:      PoolKey{Currency0: common.Address{}, Currency1: token, Fee: 3_000, TickSpacing: 60},
		SqrtPriceX96: new(big.Int).Lsh(big.NewInt(1), 96), Tick: 0,
		Ranges: []V4LiquidityRange{{TickLower: -60, TickUpper: 60, Delta: big.NewInt(1_000_000_000)}},
	}
	want, err := QuoteV4ExactInput(base, common.Address{}, big.NewInt(10_000))
	if err != nil {
		t.Fatal(err)
	}
	history := base
	history.Ranges = []V4LiquidityRange{
		{TickLower: -60, TickUpper: 60, Delta: big.NewInt(1_500_000_000)},
		{TickLower: -60, TickUpper: 60, Delta: big.NewInt(-500_000_000)},
	}
	got, err := QuoteV4ExactInput(history, common.Address{}, big.NewInt(10_000))
	if err != nil || got.Cmp(want) != 0 {
		t.Fatalf("historical quote=%v want=%v err=%v", got, want, err)
	}
	history.Ranges[1].Delta = big.NewInt(-2_000_000_000)
	if _, err := QuoteV4ExactInput(history, common.Address{}, big.NewInt(10_000)); !errors.Is(err, ErrInvalidV4State) {
		t.Fatalf("net-negative liquidity error=%v", err)
	}
}

func TestQuoteV4ExactInputMatchesCanonicalV3Loop(t *testing.T) {
	currency0 := common.HexToAddress("0x1000000000000000000000000000000000000001")
	currency1 := common.HexToAddress("0x2000000000000000000000000000000000000002")
	for _, test := range []struct {
		fee     uint32
		spacing int32
	}{
		{fee: 100, spacing: 1},
		{fee: 500, spacing: 10},
		{fee: 3_000, spacing: 60},
		{fee: 10_000, spacing: 200},
	} {
		state := V4PoolState{
			PoolKey:      PoolKey{Currency0: currency0, Currency1: currency1, Fee: test.fee, TickSpacing: test.spacing},
			SqrtPriceX96: new(big.Int).Lsh(big.NewInt(1), 96), Tick: 0,
			Ranges: []V4LiquidityRange{{TickLower: -test.spacing, TickUpper: test.spacing, Delta: big.NewInt(1_000_000_000_000)}},
		}
		ticks := []v3.Tick{
			{Index: -int(test.spacing), LiquidityGross: big.NewInt(1_000_000_000_000), LiquidityNet: big.NewInt(1_000_000_000_000)},
			{Index: int(test.spacing), LiquidityGross: big.NewInt(1_000_000_000_000), LiquidityNet: big.NewInt(-1_000_000_000_000)},
		}
		provider, err := v3.NewTickListDataProvider(ticks, int(test.spacing))
		if err != nil {
			t.Fatal(err)
		}
		pool, err := v3.NewPool(
			core.NewToken(uint(ChainID), currency0, 18, "C0", "Currency0"),
			core.NewToken(uint(ChainID), currency1, 18, "C1", "Currency1"),
			v3constants.FeeAmount(test.fee), state.SqrtPriceX96, big.NewInt(1_000_000_000_000), 0, provider,
		)
		if err != nil {
			t.Fatal(err)
		}
		for _, currencyIn := range []common.Address{currency0, currency1} {
			want, _, err := pool.GetOutputAmount(core.FromRawAmount(core.NewToken(uint(ChainID), currencyIn, 18, "IN", "Input"), big.NewInt(10_000)), nil)
			if err != nil {
				t.Fatal(err)
			}
			got, err := QuoteV4ExactInput(state, currencyIn, big.NewInt(10_000))
			if err != nil || got.Cmp(want.Quotient()) != 0 {
				t.Fatalf("fee=%d currency=%s got=%v want=%v err=%v", test.fee, currencyIn, got, want.Quotient(), err)
			}
		}
	}
}
