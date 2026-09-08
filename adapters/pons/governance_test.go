package pons

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestIsLaunchEconomicsEvent(t *testing.T) {
	if !IsLaunchEconomicsEvent(PonsV2Addresses.Factory, FactoryABI.Events["LaunchConfigUpdated"].ID) {
		t.Fatal("factory launch config update was not classified as governance")
	}
	if !IsLaunchEconomicsEvent(PonsV2Addresses.MemeHook, HookABI.Events["HookFeeBpsUpdated"].ID) {
		t.Fatal("hook fee update was not classified as governance")
	}
	if IsLaunchEconomicsEvent(PonsV2Addresses.Factory, FactoryABI.Events["TokenLaunched"].ID) {
		t.Fatal("ordinary launch was classified as governance")
	}
	if IsLaunchEconomicsEvent(PonsV2Addresses.MemeHook, HookABI.Events["HookFeeCollected"].ID) {
		t.Fatal("ordinary hook fee collection was classified as governance")
	}
	if IsLaunchEconomicsEvent(common.HexToAddress("0x1"), FactoryABI.Events["LaunchConfigUpdated"].ID) {
		t.Fatal("event from unrelated contract was classified as governance")
	}
}

func TestIsLaunchEconomicsCall(t *testing.T) {
	governance, err := FactoryWriteABI.Pack("setLaunchFee", big.NewInt(1))
	if err != nil {
		t.Fatal(err)
	}
	if !IsLaunchEconomicsCall(PonsV2Addresses.Factory, governance) {
		t.Fatal("factory governance call was not classified")
	}
	ordinary, err := FactoryWriteABI.Pack("graduate", common.HexToAddress("0x1"))
	if err != nil {
		t.Fatal(err)
	}
	if IsLaunchEconomicsCall(PonsV2Addresses.Factory, ordinary) {
		t.Fatal("ordinary factory call was classified as governance")
	}
	if IsLaunchEconomicsCall(PonsV2Addresses.Factory, []byte{1, 2, 3}) {
		t.Fatal("short calldata was classified as governance")
	}
}

func TestTokenLaunchedPairToken(t *testing.T) {
	event := FactoryABI.Events["TokenLaunched"]
	pair := common.HexToAddress("0xd0601CE157Db5bdC3162BbaC2a2C8aF5320D9EEC")
	data, err := event.Inputs.NonIndexed().Pack(pair, big.NewInt(0), big.NewInt(100))
	if err != nil {
		t.Fatal(err)
	}
	log := types.Log{Address: PonsV2Addresses.Factory, Topics: []common.Hash{event.ID, {}, {}, {}}, Data: data}
	got, err := TokenLaunchedPairToken(log)
	if err != nil || got != pair {
		t.Fatalf("pair = %s, err = %v", got, err)
	}
	log.Address = common.HexToAddress("0x1")
	if _, err := TokenLaunchedPairToken(log); err == nil {
		t.Fatal("unrelated log was accepted")
	}
}
