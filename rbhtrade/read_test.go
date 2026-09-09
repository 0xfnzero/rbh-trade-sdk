package rbhtrade

import (
	"errors"
	"fmt"
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
	bagsCall, err := BuildBagsGetTokenState(token)
	if err != nil || bagsCall.To != DefaultAddressBook().BagsLens || len(bagsCall.Data) != 36 {
		t.Fatalf("unexpected Bags getTokenState call: %#v, %v", bagsCall, err)
	}
	if _, err := BuildBagsGetTokenState(common.Address{}); err == nil {
		t.Fatal("zero Bags token was accepted")
	}
	claimer := common.HexToAddress("0x20")
	claimableCall, err := BuildBagsClaimableOf(token, claimer)
	if err != nil || claimableCall.To != DefaultAddressBook().BagsLens || len(claimableCall.Data) != 68 {
		t.Fatalf("unexpected Bags claimableOf call: %#v, %v", claimableCall, err)
	}
	if _, err := BuildBagsClaimableOf(token, common.Address{}); err == nil {
		t.Fatal("zero Bags claimer was accepted")
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
	for _, wordIndex := range []int{2, 3} {
		t.Run(fmt.Sprintf("address-word-%d", wordIndex), func(t *testing.T) {
			malformed := append([]byte(nil), data...)
			malformed[wordIndex*32] = 1
			if _, err := DecodeO1HookPoolConfig(malformed); !errors.Is(err, ErrInvalidO1HookState) {
				t.Fatalf("non-canonical o1 address word error = %v", err)
			}
		})
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
	oversizedSqrtPrice := append([]byte(nil), slotData...)
	oversizedSqrtPrice[0] = 1
	if _, err := DecodeStateViewSlot0(oversizedSqrtPrice); err == nil {
		t.Fatal("oversized uint160 sqrt price was accepted")
	}
}

func TestDecodeBagsTokenState(t *testing.T) {
	curve := common.HexToAddress("0x1000000000000000000000000000000000000001")
	feeShare := common.HexToAddress("0x2000000000000000000000000000000000000002")
	poolID := common.HexToHash("0x3000")
	state := bagsTokenStateABI{true, false, curve, feeShare, poolID, big.NewInt(1), big.NewInt(2), big.NewInt(3), big.NewInt(4), big.NewInt(5), big.NewInt(6), big.NewInt(7), big.NewInt(8)}
	data, err := bagsLensABI.Methods["getTokenState"].Outputs.Pack(state)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeBagsTokenState(data)
	if err != nil {
		t.Fatal(err)
	}
	if !decoded.Exists || decoded.Migrated || decoded.Curve != curve || decoded.FeeShare != feeShare || decoded.PoolID != poolID || decoded.TotalRaised.Cmp(big.NewInt(8)) != 0 {
		t.Fatalf("unexpected Bags token state: %#v", decoded)
	}
	if _, err := DecodeBagsTokenState(data[:len(data)-32]); err == nil {
		t.Fatal("short Bags token state response was accepted")
	}
	badPadding := append([]byte(nil), data...)
	badPadding[2*32] = 1
	if _, err := DecodeBagsTokenState(badPadding); !errors.Is(err, ErrInvalidBagsState) {
		t.Fatalf("non-canonical Bags curve address error = %v", err)
	}

	state.Exists = false
	state.Migrated = true
	data, err = bagsLensABI.Methods["getTokenState"].Outputs.Pack(state)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeBagsTokenState(data); err == nil {
		t.Fatal("migrated nonexistent Bags token state was accepted")
	}

	state.Exists = true
	state.Migrated = false
	state.Curve = common.Address{}
	data, err = bagsLensABI.Methods["getTokenState"].Outputs.Pack(state)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeBagsTokenState(data); !errors.Is(err, ErrInvalidBagsState) {
		t.Fatalf("existing state without curve error = %v", err)
	}
}

func TestDecodeRealBagsTokenStateFixture(t *testing.T) {
	data := common.FromHex("0x000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000c68fe19e53a72986fa9663cadbc6334199efebc0000000000000000000000001c50f087a05fd24b897787b39116258f5dbfc28573d405dbb7932e89df48c8aaecca60fa898b7118ad407edb2db75287c68934810000000000000000000000000000000000000000000000004563918244f4000000000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000033b2e3c9fd0803ce80000000000000000000000000000000000000000000000035f66ae4a2fdb2fbf6c9b2700000000000000000000000000000000000000000000000011df76ef21469b2e00000000000000000000000000000000000000000000000000000000498b12ba00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000008")
	state, err := DecodeBagsTokenState(data)
	if err != nil {
		t.Fatal(err)
	}
	if !state.Exists || state.Migrated || state.Curve != common.HexToAddress("0x0c68fe19e53a72986fa9663cadbc6334199efebc") || state.RealQuoteReserves.Cmp(big.NewInt(8)) != 0 || state.TotalRaised.Cmp(big.NewInt(8)) != 0 {
		t.Fatalf("unexpected live Bags state fixture: %#v", state)
	}
}

func TestBagsCurveQuoteBuildersAndDecoders(t *testing.T) {
	curve := common.HexToAddress("0x1000000000000000000000000000000000000001")
	buyCall, err := BuildBagsQuoteBuy(curve, big.NewInt(100))
	if err != nil || buyCall.To != curve || buyCall.Value.Sign() != 0 || common.Bytes2Hex(buyCall.Data[:4]) != "4beb394c" {
		t.Fatalf("unexpected Bags buy quote call: %#v, %v", buyCall, err)
	}
	sellCall, err := BuildBagsQuoteSell(curve, big.NewInt(90))
	if err != nil || sellCall.To != curve || sellCall.Value.Sign() != 0 || common.Bytes2Hex(sellCall.Data[:4]) != "a64190c4" {
		t.Fatalf("unexpected Bags sell quote call: %#v, %v", sellCall, err)
	}
	if _, err := BuildBagsQuoteBuy(common.Address{}, big.NewInt(1)); err == nil {
		t.Fatal("zero Bags curve was accepted")
	}
	if _, err := BuildBagsQuoteSell(curve, new(big.Int)); err == nil {
		t.Fatal("zero Bags sell quote input was accepted")
	}
	maximum := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
	if _, err := BuildBagsQuoteBuy(curve, maximum); err != nil {
		t.Fatalf("maximum Bags buy quote input: %v", err)
	}
	if _, err := BuildBagsQuoteBuy(curve, new(big.Int).Add(maximum, big.NewInt(1))); !errors.Is(err, ErrInvalidAmount) {
		t.Fatalf("oversized Bags buy quote input error = %v", err)
	}

	buyData, err := bagsCurveABI.Methods["quoteBuy"].Outputs.Pack(big.NewInt(90), big.NewInt(5), big.NewInt(95), big.NewInt(100), new(big.Int))
	if err != nil {
		t.Fatal(err)
	}
	buy, err := DecodeBagsQuoteBuy(buyData)
	if err != nil || buy.TokensOut.Cmp(big.NewInt(90)) != 0 || buy.GrossUsed.Cmp(big.NewInt(100)) != 0 || buy.RefundQuote.Sign() != 0 {
		t.Fatalf("unexpected Bags buy quote: %#v, %v", buy, err)
	}
	sellData, err := bagsCurveABI.Methods["quoteSell"].Outputs.Pack(big.NewInt(80), big.NewInt(4), big.NewInt(84))
	if err != nil {
		t.Fatal(err)
	}
	sell, err := DecodeBagsQuoteSell(sellData)
	if err != nil || sell.QuoteToSeller.Cmp(big.NewInt(80)) != 0 || sell.FeeQuote.Cmp(big.NewInt(4)) != 0 || sell.GrossQuoteOut.Cmp(big.NewInt(84)) != 0 {
		t.Fatalf("unexpected Bags sell quote: %#v, %v", sell, err)
	}
	if _, err := DecodeBagsQuoteBuy(buyData[:len(buyData)-32]); err == nil {
		t.Fatal("short Bags buy quote response was accepted")
	}
	if _, err := DecodeBagsQuoteSell(sellData[:len(sellData)-32]); err == nil {
		t.Fatal("short Bags sell quote response was accepted")
	}
	inconsistentBuy, err := bagsCurveABI.Methods["quoteBuy"].Outputs.Pack(big.NewInt(90), big.NewInt(6), big.NewInt(95), big.NewInt(100), new(big.Int))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeBagsQuoteBuy(inconsistentBuy); !errors.Is(err, ErrInvalidBagsQuote) {
		t.Fatalf("inconsistent Bags buy quote error = %v", err)
	}
	inconsistentSell, err := bagsCurveABI.Methods["quoteSell"].Outputs.Pack(big.NewInt(80), big.NewInt(5), big.NewInt(84))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeBagsQuoteSell(inconsistentSell); !errors.Is(err, ErrInvalidBagsQuote) {
		t.Fatalf("inconsistent Bags sell quote error = %v", err)
	}
}

func TestDecodeRealBagsCurveQuoteFixtures(t *testing.T) {
	buyData := common.FromHex("0x0000000000000000000000000000000000000000000000000002d260574849df0000000000000000000000000000000000000000000000000000000000004e2000000000000000000000000000000000000000000000000000000000000ef42000000000000000000000000000000000000000000000000000000000000f42400000000000000000000000000000000000000000000000000000000000000000")
	buy, err := DecodeBagsQuoteBuy(buyData)
	if err != nil || buy.GrossUsed.Cmp(big.NewInt(1_000_000)) != 0 || buy.NetQuoteIn.Cmp(big.NewInt(980_000)) != 0 || buy.FeeQuote.Cmp(big.NewInt(20_000)) != 0 {
		t.Fatalf("live Bags buy quote fixture = %#v, %v", buy, err)
	}

	sellData := common.FromHex("0x000000000000000000000000000000000000000000000000000000004812881c0000000000000000000000000000000000000000000000000000000001788a9d00000000000000000000000000000000000000000000000000000000498b12b9")
	sell, err := DecodeBagsQuoteSell(sellData)
	if err != nil || sell.QuoteToSeller.Cmp(new(big.Int).SetBytes(common.FromHex("0x4812881c"))) != 0 || sell.FeeQuote.Cmp(new(big.Int).SetBytes(common.FromHex("0x1788a9d"))) != 0 {
		t.Fatalf("live Bags sell quote fixture = %#v, %v", sell, err)
	}
}

func TestDecodeBagsClaimableOf(t *testing.T) {
	data, err := bagsLensABI.Methods["claimableOf"].Outputs.Pack(big.NewInt(123))
	if err != nil {
		t.Fatal(err)
	}
	amount, err := DecodeBagsClaimableOf(data)
	if err != nil || amount.Cmp(big.NewInt(123)) != 0 {
		t.Fatalf("claimable = %v, %v", amount, err)
	}
	if _, err := DecodeBagsClaimableOf(data[:0]); err == nil {
		t.Fatal("empty Bags claimable response was accepted")
	}
}

func FuzzDecodeBagsReadsNoPanic(f *testing.F) {
	f.Add([]byte(nil))
	f.Add(make([]byte, 32))
	f.Add(make([]byte, 8*32))
	f.Add(make([]byte, 13*32))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = DecodeO1HookPoolConfig(data)
		_, _ = DecodeBagsTokenState(data)
		_, _ = DecodeBagsClaimableOf(data)
		_, _ = DecodeBagsQuoteBuy(data)
		_, _ = DecodeBagsQuoteSell(data)
	})
}
