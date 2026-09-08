package rbhtrade

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestO1HookFeeBPSMatchesLinearDecay(t *testing.T) {
	schedule := O1HookFeeSchedule{BaseFeeBPS: 100, AntiSnipeStartTotalBPS: 9_900, AntiSnipeWindowSeconds: 100, LaunchTime: 1_000}
	for _, test := range []struct {
		at   uint64
		want uint16
	}{{1_000, 9_900}, {1_050, 5_000}, {1_099, 198}, {1_100, 100}, {2_000, 100}} {
		got, err := O1HookFeeBPS(schedule, test.at)
		if err != nil || got != test.want {
			t.Fatalf("fee at %d = %d, %v; want %d", test.at, got, err, test.want)
		}
	}
}

func TestO1HookFeeBPSRejectsInvalidSchedule(t *testing.T) {
	for _, schedule := range []O1HookFeeSchedule{
		{},
		{BaseFeeBPS: 1_001, AntiSnipeStartTotalBPS: 1_001, AntiSnipeWindowSeconds: 1, LaunchTime: 1},
		{BaseFeeBPS: 100, AntiSnipeStartTotalBPS: 99, AntiSnipeWindowSeconds: 1, LaunchTime: 1},
		{BaseFeeBPS: 100, AntiSnipeStartTotalBPS: 9_901, AntiSnipeWindowSeconds: 1, LaunchTime: 1},
	} {
		if _, err := O1HookFeeBPS(schedule, 1); !errors.Is(err, ErrInvalidO1HookState) {
			t.Fatalf("invalid schedule error = %v", err)
		}
	}
	valid := O1HookFeeSchedule{BaseFeeBPS: 100, AntiSnipeStartTotalBPS: 1_000, AntiSnipeWindowSeconds: 10, LaunchTime: 100}
	if _, err := O1HookFeeBPS(valid, 99); !errors.Is(err, ErrInvalidO1HookState) {
		t.Fatalf("pre-launch timestamp error = %v", err)
	}
}

func TestQuoteO1V4ExactInputChargesQuoteCurrency(t *testing.T) {
	token := common.HexToAddress("0x1000000000000000000000000000000000000001")
	state := V4PoolState{
		PoolKey:      PoolKey{Currency0: common.Address{}, Currency1: token, Fee: 3_000, TickSpacing: 60, Hooks: DefaultAddressBook().O1Hook},
		SqrtPriceX96: new(big.Int).Lsh(big.NewInt(1), 96), Tick: 0,
		Ranges: []V4LiquidityRange{{TickLower: -887220, TickUpper: 887220, Delta: big.NewInt(1_000_000_000)}},
	}
	amount := big.NewInt(10_000)
	buy, err := QuoteO1V4ExactInput(state, common.Address{}, common.Address{}, amount, 1_000)
	wantBuy, wantErr := QuoteV4ExactInput(state, common.Address{}, big.NewInt(9_000))
	if err != nil || wantErr != nil || buy.Cmp(wantBuy) != 0 {
		t.Fatalf("buy quote = %v, %v; want %v, %v", buy, err, wantBuy, wantErr)
	}
	sellGross, err := QuoteV4ExactInput(state, token, amount)
	if err != nil {
		t.Fatal(err)
	}
	wantSell := new(big.Int).Sub(sellGross, new(big.Int).Div(new(big.Int).Mul(sellGross, big.NewInt(1_000)), big.NewInt(BPSTotal)))
	sell, err := QuoteO1V4ExactInput(state, token, common.Address{}, amount, 1_000)
	if err != nil || sell.Cmp(wantSell) != 0 {
		t.Fatalf("sell quote = %v, %v; want %v", sell, err, wantSell)
	}
}

func TestQuoteO1V4ExactInputRejectsWrongHook(t *testing.T) {
	state := V4PoolState{PoolKey: PoolKey{Currency0: common.Address{}, Currency1: common.HexToAddress("0x1"), Fee: 3_000, TickSpacing: 60}}
	if _, err := QuoteO1V4ExactInput(state, common.Address{}, common.Address{}, big.NewInt(1), 100); !errors.Is(err, ErrInvalidO1HookState) {
		t.Fatalf("wrong hook error = %v", err)
	}
}
