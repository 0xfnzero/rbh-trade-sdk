package rbhtrade

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func assertCall(t *testing.T, call Call, target common.Address, selector string, value *big.Int) {
	t.Helper()
	if call.To != target || len(call.Data) < 4 || common.Bytes2Hex(call.Data[:4]) != selector || call.Value.Cmp(value) != 0 {
		t.Fatalf("unexpected call: %#v", call)
	}
}

func TestBuildLongCreateSelectorAndTarget(t *testing.T) {
	params := LongCreateParams{
		InitialSupply:     new(big.Int).Exp(big.NewInt(10), big.NewInt(27), nil),
		NumTokensToSell:   new(big.Int).Exp(big.NewInt(10), big.NewInt(27), nil),
		Numeraire:         common.HexToAddress("0x11"),
		TokenFactory:      DefaultAddressBook().LongTokenFactory,
		GovernanceFactory: common.HexToAddress("0x12"),
		PoolInitializer:   DefaultAddressBook().LongDopplerHook,
		LiquidityMigrator: common.HexToAddress("0x13"),
	}
	call, err := BuildLongCreate(params)
	if err != nil {
		t.Fatal(err)
	}
	if call.To != DefaultAddressBook().LongLauncher || common.Bytes2Hex(call.Data[:4]) != "882db707" || call.Value.Sign() != 0 {
		t.Fatalf("unexpected Long call: %#v", call)
	}
}

func TestBuildPAIRLaunchValidatesWeightsAndNoBuySentinel(t *testing.T) {
	params := PAIRLaunchParams{
		Name:                    "Pair Test",
		Symbol:                  "PAIRTEST",
		Allocations:             []PAIRAllocation{{QuoteToken: common.HexToAddress("0x11"), WeightBPS: BPSTotal}},
		CreatorFeeRecipient:     common.HexToAddress("0x12"),
		DeveloperBuyRecipient:   common.HexToAddress("0x12"),
		DeveloperBuyPairIndex:   PAIRNoDeveloperBuy,
		DeveloperTokenAmountOut: new(big.Int),
		MaxQuoteAmountIn:        new(big.Int),
		Deadline:                big.NewInt(100),
	}
	if _, err := BuildPAIRLaunch(params, new(big.Int)); err != nil {
		t.Fatal(err)
	}
	params.Allocations[0].WeightBPS = 9999
	if _, err := BuildPAIRLaunch(params, new(big.Int)); err == nil {
		t.Fatal("invalid allocation weights accepted")
	}
}

func TestBuildUniversalRouterApprovals(t *testing.T) {
	token := common.HexToAddress("0x11")
	calls, err := BuildUniversalRouterApprovals(token, big.NewInt(123), 999)
	if err != nil {
		t.Fatal(err)
	}
	if calls[0].To != token || calls[1].To != DefaultAddressBook().Permit2 {
		t.Fatalf("unexpected approvals: %#v", calls)
	}
	if common.Bytes2Hex(calls[0].Data[:4]) != "095ea7b3" {
		t.Fatalf("unexpected ERC-20 approve selector")
	}
	if _, err := BuildUniversalRouterApprovals(token, new(big.Int).Lsh(big.NewInt(1), 160), 999); err == nil {
		t.Fatal("uint160 overflow accepted")
	}
}

func TestLaunchBuilderSelectorsAndTargets(t *testing.T) {
	addresses := DefaultAddressBook()
	o1 := O1LaunchParams{TokenName: "Test", TokenSymbol: "TST", Deadline: 1}
	call, err := BuildO1Launch(o1, big.NewInt(7))
	if err != nil {
		t.Fatal(err)
	}
	assertCall(t, call, addresses.O1Factory, "3feab1d8", big.NewInt(7))

	call, err = BuildO1LaunchAndBuy(o1, O1LaunchBuyParams{FundingToken: common.HexToAddress("0x11"), AmountIn: big.NewInt(2), MinAmountOut: big.NewInt(1)}, big.NewInt(7))
	if err != nil {
		t.Fatal(err)
	}
	assertCall(t, call, addresses.O1Factory, "901a3de8", big.NewInt(7))

	bags := BagsLaunchParams{Name: "Test", Symbol: "TST"}
	call, err = BuildBagsCreate(bags, big.NewInt(3))
	if err != nil {
		t.Fatal(err)
	}
	assertCall(t, call, addresses.BagsFactory, "2b24d20e", big.NewInt(3))
	call, err = BuildBagsCreateAndBuy(bags, big.NewInt(4))
	if err != nil {
		t.Fatal(err)
	}
	assertCall(t, call, addresses.BagsFactory, "861a5ece", big.NewInt(4))

	call, err = BuildPoolsCreateToken(common.HexToAddress("0x11"), "Test", "TST", 18, big.NewInt(1), common.HexToAddress("0x12"), nil, big.NewInt(5))
	if err != nil {
		t.Fatal(err)
	}
	assertCall(t, call, addresses.PoolsEntry, "dec14be1", big.NewInt(5))
}

func TestBuilderValueAndMulticallBoundaries(t *testing.T) {
	o1 := O1LaunchParams{TokenName: "Test", TokenSymbol: "TST", Deadline: 1}
	if _, err := BuildO1LaunchAndBuy(o1, O1LaunchBuyParams{AmountIn: big.NewInt(2), MinAmountOut: new(big.Int)}, big.NewInt(1)); err == nil {
		t.Fatal("underfunded native o1 launch-and-buy accepted")
	}
	if _, err := BuildPoolsMulticall([][]byte{{0x01}, nil}, new(big.Int)); err == nil {
		t.Fatal("empty multicall element accepted")
	}

	value := big.NewInt(9)
	call, err := BuildBagsCreate(BagsLaunchParams{Name: "Test", Symbol: "TST"}, value)
	if err != nil {
		t.Fatal(err)
	}
	value.SetInt64(1)
	if call.Value.Int64() != 9 {
		t.Fatal("returned value aliases caller input")
	}

	payload := []byte{1, 2, 3}
	call, err = BuildPoolsMulticall([][]byte{payload}, new(big.Int))
	if err != nil {
		t.Fatal(err)
	}
	payload[0] = 9
	if bytes.Contains(call.Data, []byte{9, 2, 3}) {
		t.Fatal("encoded calldata aliases multicall input")
	}
}

func TestBuildersRejectOversizedDynamicInputs(t *testing.T) {
	token := common.HexToAddress("0x11")
	key := PoolKey{Currency0: common.Address{}, Currency1: token, Fee: 3000, TickSpacing: 60}
	if _, err := BuildV4ExactInputSingle(ExactInputRequest{PoolKey: key, CurrencyIn: common.Address{}, CurrencyOut: token, AmountIn: big.NewInt(1), AmountOutMinimum: new(big.Int), HookData: make([]byte, MaxHookDataBytes+1), Deadline: 1}); err == nil {
		t.Fatal("oversized V4 hook data was accepted")
	}
	if _, err := BuildPoolsCreateToken(token, "Test", "T", 18, big.NewInt(1), common.HexToAddress("0x12"), make([]byte, MaxDynamicPayloadBytes+1), new(big.Int)); err == nil {
		t.Fatal("oversized Pools token data was accepted")
	}
	if _, err := BuildBagsCreate(BagsLaunchParams{Name: string(bytes.Repeat([]byte{'n'}, MaxTokenNameBytes+1)), Symbol: "T"}, new(big.Int)); err == nil {
		t.Fatal("oversized token name was accepted")
	}
}

func TestKnownMethodSelectors(t *testing.T) {
	tests := []struct {
		name string
		got  []byte
		want string
	}{
		{"ERC20 approve", erc20ABI.Methods["approve"].ID, "095ea7b3"},
		{"Permit2 approve", permit2ABI.Methods["approve"].ID, "87517c45"},
		{"PAIR launchTokenMulti", pairABI.Methods["launchTokenMulti"].ID, "07489e0a"},
		{"Pools distributeToken", poolsABI.Methods["distributeToken"].ID, "b6982b48"},
		{"Pools distributeWithNative", poolsABI.Methods["distributeWithNative"].ID, "0ef847b6"},
		{"Pools multicall", poolsABI.Methods["multicall"].ID, "ac9650d8"},
		{"Bags buy", bagsCurveABI.Methods["buy"].ID, "d96a094a"},
		{"Bags buyFor", bagsCurveABI.Methods["buyFor"].ID, "06501a6a"},
		{"Bags sell", bagsCurveABI.Methods["sell"].ID, "d79875eb"},
		{"Bags sellFor", bagsCurveABI.Methods["sellFor"].ID, "5f6108ff"},
		{"v4 quote", v4QuoterABI.Methods["quoteExactInputSingle"].ID, "aa9d21cb"},
		{"StateView getSlot0", stateViewABI.Methods["getSlot0"].ID, "c815641c"},
	}
	for _, test := range tests {
		if got := common.Bytes2Hex(test.got); got != test.want {
			t.Errorf("%s selector = %s, want %s", test.name, got, test.want)
		}
	}
}
