package rbhtrade

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestQuoteBagsV4ExactInputChargesWETHLeg(t *testing.T) {
	weth := DefaultAddressBook().WETH
	token := common.HexToAddress("0x1000000000000000000000000000000000000001")
	state := V4PoolState{
		PoolKey: PoolKey{
			Currency0: weth, Currency1: token,
			Fee: 1 << 23, TickSpacing: 60, Hooks: DefaultAddressBook().BagsHook,
		},
		SqrtPriceX96: new(big.Int).Lsh(big.NewInt(1), 96), Tick: 0,
		Ranges: []V4LiquidityRange{{TickLower: -887220, TickUpper: 887220, Delta: big.NewInt(1_000_000_000)}},
	}
	amount := big.NewInt(10_000)
	zeroFeeState := state
	zeroFeeState.PoolKey.Fee = 0

	effectiveBuy := big.NewInt(9_800)
	wantBuy, err := QuoteV4ExactInput(zeroFeeState, weth, effectiveBuy)
	if err != nil {
		t.Fatal(err)
	}
	buy, err := QuoteBagsV4ExactInput(state, weth, amount)
	if err != nil || buy.Cmp(wantBuy) != 0 {
		t.Fatalf("buy=%v want=%v err=%v", buy, wantBuy, err)
	}

	grossSell, err := QuoteV4ExactInput(zeroFeeState, token, amount)
	if err != nil {
		t.Fatal(err)
	}
	wantSell := new(big.Int).Sub(grossSell, new(big.Int).Div(new(big.Int).Mul(grossSell, big.NewInt(200)), big.NewInt(10_000)))
	sell, err := QuoteBagsV4ExactInput(state, token, amount)
	if err != nil || sell.Cmp(wantSell) != 0 {
		t.Fatalf("sell=%v want=%v err=%v", sell, wantSell, err)
	}
}

func TestQuoteBagsV4ExactInputRejectsUnverifiedState(t *testing.T) {
	state := V4PoolState{PoolKey: PoolKey{Hooks: DefaultAddressBook().O1Hook}}
	if _, err := QuoteBagsV4ExactInput(state, common.Address{}, big.NewInt(1)); !errors.Is(err, ErrInvalidBagsV4HookState) {
		t.Fatalf("wrong hook error=%v", err)
	}
}
