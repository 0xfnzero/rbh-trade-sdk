package pons

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func validParams() TokenParams {
	return TokenParams{Name: "Test", Symbol: "TST", CreatorFeeRecipient: common.HexToAddress("0x11")}
}

func TestBuildLaunchRejectsOversizedValue(t *testing.T) {
	oversized := new(big.Int).Lsh(big.NewInt(1), 256)
	if _, err := BuildLaunch(validParams(), new(big.Int), NativeQuote, nil, oversized); err == nil {
		t.Fatal("oversized launch value accepted")
	}
}

func TestBuildLaunchAndBuyRejectsUnderfundedNativeQuote(t *testing.T) {
	if _, err := BuildLaunchAndBuy(validParams(), new(big.Int), NativeQuote, big.NewInt(10), new(big.Int), common.HexToAddress("0x12"), nil, big.NewInt(9)); err == nil {
		t.Fatal("underfunded native launch-and-buy accepted")
	}
}

func TestBuildLaunchAndBuyRejectsNilNativeQuoteWithoutPanic(t *testing.T) {
	if _, err := BuildLaunchAndBuy(validParams(), new(big.Int), NativeQuote, nil, new(big.Int), common.HexToAddress("0x12"), nil, new(big.Int)); err == nil {
		t.Fatal("nil native quote input accepted")
	}
}
