package rbhtrade

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestQuoteAndStateViewBuilders(t *testing.T) {
	token := common.HexToAddress("0x10")
	key := PoolKey{Currency0: common.Address{}, Currency1: token, Fee: 3000, TickSpacing: 60}
	quoteCall, err := BuildV4QuoteExactInputSingle(QuoteExactInputRequest{PoolKey: key, CurrencyIn: common.Address{}, AmountIn: big.NewInt(123)})
	if err != nil {
		t.Fatal(err)
	}
	if quoteCall.To != DefaultAddressBook().V4Quoter || len(quoteCall.Data) < 4 {
		t.Fatalf("unexpected quote call: %#v", quoteCall)
	}
	id, err := PoolID(key)
	if err != nil {
		t.Fatal(err)
	}
	stateCall, err := BuildStateViewGetSlot0(id)
	if err != nil {
		t.Fatal(err)
	}
	if stateCall.To != DefaultAddressBook().StateView || len(stateCall.Data) != 36 {
		t.Fatalf("unexpected StateView call: %#v", stateCall)
	}
	o1Call, err := BuildO1HookPoolConfig(id)
	if err != nil || o1Call.To != DefaultAddressBook().O1Hook || len(o1Call.Data) != 36 {
		t.Fatalf("unexpected o1 poolConfig call: %#v, %v", o1Call, err)
	}
}

func TestDecodeO1HookPoolConfig(t *testing.T) {
	creator := common.HexToAddress("0x1000000000000000000000000000000000000001")
	recipient := common.HexToAddress("0x2000000000000000000000000000000000000002")
	data, err := o1HookReadABI.Methods["poolConfig"].Outputs.Pack(true, true, creator, recipient, uint16(100), uint16(9_900), uint32(120), big.NewInt(1_000))
	if err != nil {
		t.Fatal(err)
	}
	config, err := DecodeO1HookPoolConfig(data)
	if err != nil || !config.Initialized || !config.TokenIsCurrency0 || config.CurrentCreator != creator || config.CreatorFeeRecipient != recipient || config.FeeSchedule.LaunchTime != 1_000 {
		t.Fatalf("o1 config=%#v err=%v", config, err)
	}
	if _, err := DecodeO1HookPoolConfig(data[:32]); err == nil {
		t.Fatal("short o1 config response was accepted")
	}
}

func TestDecodeQuoteAndSlot0(t *testing.T) {
	quoteData, err := v4QuoterABI.Methods["quoteExactInputSingle"].Outputs.Pack(big.NewInt(100), big.NewInt(50_000))
	if err != nil {
		t.Fatal(err)
	}
	quote, err := DecodeV4QuoteExactInputSingle(quoteData)
	if err != nil || quote.AmountOut.Uint64() != 100 || quote.GasEstimate.Uint64() != 50_000 {
		t.Fatalf("quote=%#v err=%v", quote, err)
	}
	slotData, err := stateViewABI.Methods["getSlot0"].Outputs.Pack(big.NewInt(1).Lsh(big.NewInt(1), 96), big.NewInt(-10), big.NewInt(2), big.NewInt(3000))
	if err != nil {
		t.Fatal(err)
	}
	slot, err := DecodeStateViewSlot0(slotData)
	if err != nil || slot.Tick != -10 || slot.LPFee != 3000 {
		t.Fatalf("slot=%#v err=%v", slot, err)
	}
}
