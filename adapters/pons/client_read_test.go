package pons

import (
	"context"
	"errors"
	"math/big"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

type readResponseKey struct {
	address  common.Address
	selector [4]byte
}

type readCaller struct {
	mu        sync.Mutex
	responses map[readResponseKey][]byte
	blocks    []*big.Int
}

func (*readCaller) CodeAt(context.Context, common.Address, *big.Int) ([]byte, error) {
	return []byte{1}, nil
}

func (c *readCaller) CallContract(_ context.Context, call ethereum.CallMsg, block *big.Int) ([]byte, error) {
	if call.To == nil || len(call.Data) < 4 {
		return nil, errors.New("invalid contract call")
	}
	var selector [4]byte
	copy(selector[:], call.Data[:4])

	c.mu.Lock()
	if block == nil {
		c.blocks = append(c.blocks, nil)
	} else {
		c.blocks = append(c.blocks, new(big.Int).Set(block))
	}
	response, ok := c.responses[readResponseKey{address: *call.To, selector: selector}]
	c.mu.Unlock()
	if !ok {
		return nil, errors.New("unexpected contract call")
	}
	return append([]byte(nil), response...), nil
}

func (*readCaller) BlockNumber(context.Context) (uint64, error) {
	return 777, nil
}

func addReadResponse(t *testing.T, caller *readCaller, address common.Address, contractABI abi.ABI, method string, values ...interface{}) {
	t.Helper()
	abiMethod, ok := contractABI.Methods[method]
	if !ok {
		t.Fatalf("ABI method %s not found", method)
	}
	encoded, err := abiMethod.Outputs.Pack(values...)
	if err != nil {
		t.Fatalf("pack %s response: %v", method, err)
	}
	var selector [4]byte
	copy(selector[:], abiMethod.ID)
	caller.responses[readResponseKey{address: address, selector: selector}] = encoded
}

func TestClientFactoryReadMappings(t *testing.T) {
	factory := common.HexToAddress("0x00000000000000000000000000000000000000f1")
	token := common.HexToAddress("0x00000000000000000000000000000000000000a1")
	curve := common.HexToAddress("0x00000000000000000000000000000000000000a2")
	deployer := common.HexToAddress("0x00000000000000000000000000000000000000a3")
	recipient := common.HexToAddress("0x00000000000000000000000000000000000000a4")
	pairToken := common.HexToAddress("0x00000000000000000000000000000000000000a5")
	caller := &readCaller{responses: make(map[readResponseKey][]byte)}
	client := &Client{caller: caller, addresses: Addresses{Factory: factory}}

	config := launchConfigABI{
		Supply: big.NewInt(1_000_000), CurveFeeBps: big.NewInt(100), PhantomQuote: big.NewInt(10_000),
		GraduationThreshold: big.NewInt(50_000), PoolFee: big.NewInt(3_000), TickSpacing: big.NewInt(60), Enabled: true,
	}
	launched := launchedTokenABI{
		Token: token, Curve: curve, Deployer: deployer, CreatorFeeRecipient: recipient, PairToken: pairToken,
		GraduationThreshold: big.NewInt(50_000), PoolFee: big.NewInt(3_000), TickSpacing: big.NewInt(60), CreatorTaxBps: 100,
		BuybackEnabled: true, Phase: uint8(PhaseSwept), SweptQuote: big.NewInt(1), SweptTokens: big.NewInt(2),
		SweptAt: big.NewInt(3), Exists: true,
	}
	policy := FeePolicy{
		ProtocolFeeRecipient: recipient, ProtocolFeeShareBps: 2_000, BuybackBurnBps: 5_000,
		HookFeeBps: 100, MaxInternalPriceImpactBps: 300,
	}
	factoryBigValues := map[string]int64{
		"launchFee":                              11,
		"maxCreatorTaxBps":                       12,
		"snipeTaxStartBps":                       13,
		"snipeTaxSeconds":                        14,
		"CREATOR_FEE_RECIPIENT_TIMELOCK":         15,
		"CREATOR_FEE_RECIPIENT_EXECUTION_WINDOW": 16,
		"GRADUATION_RESCUE_DELAY":                17,
	}
	for method, value := range factoryBigValues {
		addReadResponse(t, caller, factory, FactoryABI, method, big.NewInt(value))
	}
	addReadResponse(t, caller, factory, FactoryABI, "launchEnabled", true)
	addReadResponse(t, caller, factory, FactoryABI, "canLaunch", true)
	addReadResponse(t, caller, factory, FactoryABI, "whitelistedLaunchers", true)
	addReadResponse(t, caller, factory, FactoryABI, "approvedPairTokens", true)
	addReadResponse(t, caller, factory, FactoryABI, "owner", deployer)
	addReadResponse(t, caller, factory, FactoryABI, "pendingOwner", recipient)
	addReadResponse(t, caller, factory, FactoryABI, "launchConfigCount", big.NewInt(2))
	addReadResponse(t, caller, factory, FactoryABI, "getLaunchConfig", config)
	addReadResponse(t, caller, factory, FactoryABI, "previewLaunchEconomics", [32]byte{1, 2, 3})
	addReadResponse(t, caller, factory, FactoryABI, "pairTokenEconomics", big.NewInt(10_000), big.NewInt(50_000), uint8(18))
	addReadResponse(t, caller, factory, FactoryABI, "pendingCreatorFeeRecipient", recipient, big.NewInt(100), big.NewInt(200))
	addReadResponse(t, caller, factory, FactoryABI, "getLaunchedToken", launched)
	addReadResponse(t, caller, factory, FactoryABI, "getLaunchFeePolicy", policy)

	ctx := context.Background()
	configs, err := client.OpenLaunchConfigsWithOptions(ctx, LaunchConfigQueryOptions{MaxConfigs: 2, Concurrency: 2}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(configs) != 2 || configs[0].ID.Sign() != 0 || configs[1].ID.Cmp(big.NewInt(1)) != 0 || configs[0].Supply.Cmp(config.Supply) != 0 {
		t.Fatalf("unexpected open configs: %#v", configs)
	}
	if defaults, err := client.OpenLaunchConfigs(ctx, nil); err != nil || len(defaults) != 2 {
		t.Fatalf("default open configs: count=%d err=%v", len(defaults), err)
	}
	gotLaunch, err := client.GetLaunchedToken(ctx, token, nil)
	if err != nil {
		t.Fatal(err)
	}
	if gotLaunch.Token != token || gotLaunch.Curve != curve || gotLaunch.TickSpacing != 60 || !gotLaunch.Exists {
		t.Fatalf("unexpected launched token: %#v", gotLaunch)
	}
	bigChecks := []struct {
		name string
		want int64
		call func() (*big.Int, error)
	}{
		{"LaunchFee", 11, func() (*big.Int, error) { return client.LaunchFee(ctx, nil) }},
		{"MaxCreatorTaxBps", 12, func() (*big.Int, error) { return client.MaxCreatorTaxBps(ctx, nil) }},
		{"SnipeTaxStartBps", 13, func() (*big.Int, error) { return client.SnipeTaxStartBps(ctx, nil) }},
		{"SnipeTaxSeconds", 14, func() (*big.Int, error) { return client.SnipeTaxSeconds(ctx, nil) }},
		{"CreatorFeeRecipientTimelock", 15, func() (*big.Int, error) { return client.CreatorFeeRecipientTimelock(ctx, nil) }},
		{"CreatorFeeRecipientExecutionWindow", 16, func() (*big.Int, error) { return client.CreatorFeeRecipientExecutionWindow(ctx, nil) }},
		{"GraduationRescueDelay", 17, func() (*big.Int, error) { return client.GraduationRescueDelay(ctx, nil) }},
	}
	for _, check := range bigChecks {
		t.Run(check.name, func(t *testing.T) {
			got, err := check.call()
			if err != nil || got.Cmp(big.NewInt(check.want)) != 0 {
				t.Fatalf("value = %v, err = %v, want %d", got, err, check.want)
			}
		})
	}

	checks := []struct {
		name string
		call func() error
	}{
		{"LaunchEnabled", func() error { _, err := client.LaunchEnabled(ctx, nil); return err }},
		{"CanLaunch", func() error { _, err := client.CanLaunch(ctx, deployer, nil); return err }},
		{"WhitelistedLauncher", func() error { _, err := client.WhitelistedLauncher(ctx, deployer, nil); return err }},
		{"FactoryOwner", func() error { _, err := client.FactoryOwner(ctx, nil); return err }},
		{"FactoryPendingOwner", func() error { _, err := client.FactoryPendingOwner(ctx, nil); return err }},
		{"GetLaunchConfig", func() error { _, err := client.GetLaunchConfig(ctx, big.NewInt(0), nil); return err }},
		{"PreviewLaunchEconomics", func() error { _, err := client.PreviewLaunchEconomics(ctx, big.NewInt(0), pairToken, nil); return err }},
		{"ApprovedPairToken", func() error { _, err := client.ApprovedPairToken(ctx, pairToken, nil); return err }},
		{"PairTokenEconomics", func() error { _, err := client.PairTokenEconomics(ctx, pairToken, nil); return err }},
		{"PendingCreatorFeeRecipient", func() error { _, err := client.PendingCreatorFeeRecipient(ctx, token, nil); return err }},
		{"GetLaunchFeePolicy", func() error { _, err := client.GetLaunchFeePolicy(ctx, token, nil); return err }},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			if err := check.call(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestNewClientOptions(t *testing.T) {
	addresses := Addresses{Factory: common.HexToAddress("0x00000000000000000000000000000000000000f1")}
	client := NewClient(nil, WithAddresses(addresses))
	if client.Addresses() != addresses {
		t.Fatalf("addresses = %#v, want %#v", client.Addresses(), addresses)
	}
}

func TestNewClientCheckedValidatesConfiguration(t *testing.T) {
	if _, err := NewClientChecked(nil); !errors.Is(err, ErrNilBackend) {
		t.Fatalf("nil backend error = %v", err)
	}
	backend := &struct{ bind.ContractBackend }{}
	if _, err := NewClientChecked(backend, WithAddresses(Addresses{})); !errors.Is(err, ErrZeroAddress) {
		t.Fatalf("zero address error = %v", err)
	}
	client, err := NewClientChecked(backend)
	if err != nil {
		t.Fatal(err)
	}
	if client.Addresses() != defaultPonsV2Addresses {
		t.Fatalf("unexpected default addresses: %#v", client.Addresses())
	}
}

func TestClientCompositeReadMappings(t *testing.T) {
	addresses := Addresses{
		Factory:      common.HexToAddress("0x00000000000000000000000000000000000000f1"),
		MemeHook:     common.HexToAddress("0x00000000000000000000000000000000000000f2"),
		FeeEscrow:    common.HexToAddress("0x00000000000000000000000000000000000000f3"),
		BuybackVault: common.HexToAddress("0x00000000000000000000000000000000000000f4"),
		LaunchLocker: common.HexToAddress("0x00000000000000000000000000000000000000f5"),
	}
	curve := common.HexToAddress("0x00000000000000000000000000000000000000c1")
	token := common.HexToAddress("0x00000000000000000000000000000000000000c2")
	pairToken := common.HexToAddress("0x00000000000000000000000000000000000000c3")
	deployer := common.HexToAddress("0x00000000000000000000000000000000000000c4")
	recipient := common.HexToAddress("0x00000000000000000000000000000000000000c5")
	poolID := common.HexToHash("0x1234")
	caller := &readCaller{responses: make(map[readResponseKey][]byte)}
	client := &Client{caller: caller, addresses: addresses}

	addReadResponse(t, caller, addresses.MemeHook, HookABI, "launches", true, false, token, pairToken, deployer, recipient, addresses.FeeEscrow, uint16(100), uint16(2_000), uint16(5_000), uint16(30), uint16(300), true)
	addReadResponse(t, caller, addresses.MemeHook, HookABI, "currentFeePolicy", FeePolicy{ProtocolFeeRecipient: recipient, ProtocolFeeShareBps: 2_000, BuybackBurnBps: 5_000, HookFeeBps: 30, MaxInternalPriceImpactBps: 300})
	for _, method := range []string{"pendingFees", "pendingCreatorTax", "pendingBuyback"} {
		addReadResponse(t, caller, addresses.MemeHook, HookABI, method, big.NewInt(9))
	}

	for method, value := range map[string]common.Address{"token": token, "pairToken": pairToken, "deployer": deployer, "factory": addresses.Factory} {
		addReadResponse(t, caller, curve, CurveABI, method, value)
	}
	addReadResponse(t, caller, curve, CurveABI, "getReserves", big.NewInt(1_000), big.NewInt(10_000))
	bigMethods := []string{"phantomQuote", "realQuoteReserve", "graduationThreshold", "sellableTokens", "reservedTokens", "trackedQuote", "trackedTokens", "quoteFeeBalance", "buybackQuoteBalance", "creatorTaxBalance", "feeBps", "creatorTaxBps"}
	for i, method := range bigMethods {
		addReadResponse(t, caller, curve, CurveABI, method, big.NewInt(int64(i+1)))
	}
	addReadResponse(t, caller, curve, CurveABI, "readyToGraduate", false)
	addReadResponse(t, caller, curve, CurveABI, "graduated", false)
	addReadResponse(t, caller, curve, CurveABI, "buybackEnabled", true)
	addReadResponse(t, caller, curve, CurveABI, "isNativeQuote", false)

	socials := Socials{Twitter: "x", Telegram: "tg", Discord: "discord", Website: "web", Farcaster: "fc"}
	addReadResponse(t, caller, token, TokenABI, "name", "Pons")
	addReadResponse(t, caller, token, TokenABI, "symbol", "PONS")
	addReadResponse(t, caller, token, TokenABI, "decimals", uint8(18))
	addReadResponse(t, caller, token, TokenABI, "totalSupply", big.NewInt(1_000_000))
	addReadResponse(t, caller, token, TokenABI, "launchFactory", addresses.Factory)
	addReadResponse(t, caller, token, TokenABI, "curve", curve)
	addReadResponse(t, caller, token, TokenABI, "getTokenInfo", deployer, "logo", "description", socials)
	addReadResponse(t, caller, token, TokenABI, "allowance", big.NewInt(99))

	for i, method := range []string{"totalLocked", "totalReleased", "vestedAmount", "releasable", "vestingStart", "VESTING_DURATION"} {
		addReadResponse(t, caller, addresses.BuybackVault, VaultABI, method, big.NewInt(int64(i+10)))
	}
	addReadResponse(t, caller, addresses.BuybackVault, VaultABI, "vestingTerms", recipient, addresses.FeeEscrow, uint16(2_000))
	addReadResponse(t, caller, addresses.FeeEscrow, EscrowABI, "balanceOf", big.NewInt(7))
	addReadResponse(t, caller, addresses.FeeEscrow, EscrowABI, "balanceOfToken", big.NewInt(8))
	addReadResponse(t, caller, addresses.LaunchLocker, LockerABI, "isLocked", true)

	ctx := context.Background()
	hookLaunch, err := client.HookLaunch(ctx, poolID, nil)
	if err != nil || !hookLaunch.Registered || hookLaunch.Memecoin != token || hookLaunch.HookFeeBps != 30 {
		t.Fatalf("unexpected hook launch: value=%#v err=%v", hookLaunch, err)
	}
	state, err := client.CurveState(ctx, curve, nil)
	if err != nil || state.Token != token || state.Reserves.QuoteReserve.Cmp(big.NewInt(1_000)) != 0 || !state.BuybackEnabled {
		t.Fatalf("unexpected curve state: value=%#v err=%v", state, err)
	}
	metadata, err := client.TokenMetadata(ctx, token, nil)
	if err != nil || metadata.Name != "Pons" || metadata.Deployer != deployer || metadata.Socials.Telegram != "tg" {
		t.Fatalf("unexpected token metadata: value=%#v err=%v", metadata, err)
	}
	vest, err := client.VestState(ctx, token, nil)
	if err != nil || vest.TotalLocked.Cmp(big.NewInt(10)) != 0 || vest.Terms.CreatorRecipient != recipient {
		t.Fatalf("unexpected vest state: value=%#v err=%v", vest, err)
	}

	checks := []struct {
		name string
		call func() error
	}{
		{"CurrentFeePolicy", func() error { _, err := client.CurrentFeePolicy(ctx, nil); return err }},
		{"GetReserves", func() error { _, err := client.GetReserves(ctx, curve, nil); return err }},
		{"IsNativeQuote", func() error { _, err := client.IsNativeQuote(ctx, curve, nil); return err }},
		{"TokenInfo", func() error { _, err := client.TokenInfo(ctx, token, nil); return err }},
		{"TokenAllowance", func() error { _, err := client.TokenAllowance(ctx, token, deployer, recipient, nil); return err }},
		{"EscrowNativeBalance", func() error { _, err := client.EscrowNativeBalance(ctx, recipient, nil); return err }},
		{"EscrowTokenBalance", func() error { _, err := client.EscrowTokenBalance(ctx, recipient, token, nil); return err }},
		{"VestingTerms", func() error { _, err := client.VestingTerms(ctx, token, nil); return err }},
		{"PendingFees", func() error { _, err := client.PendingFees(ctx, poolID, pairToken, nil); return err }},
		{"PendingCreatorTax", func() error { _, err := client.PendingCreatorTax(ctx, poolID, pairToken, nil); return err }},
		{"PendingBuyback", func() error { _, err := client.PendingBuyback(ctx, poolID, pairToken, nil); return err }},
		{"LockerIsLocked", func() error { _, err := client.LockerIsLocked(ctx, token, nil); return err }},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			if err := check.call(); err != nil {
				t.Fatal(err)
			}
		})
	}

	caller.mu.Lock()
	defer caller.mu.Unlock()
	for _, block := range caller.blocks {
		if block != nil && block.Cmp(big.NewInt(777)) != 0 {
			t.Fatalf("composite read used block %v, want 777", block)
		}
	}
}

func TestABIIntegerConversionsRejectOutOfRangeValues(t *testing.T) {
	if _, err := uint24ToUint32(new(big.Int).Lsh(big.NewInt(1), 24), "value"); err == nil {
		t.Fatal("expected uint24 overflow error")
	}
	if _, err := int24ToInt32(big.NewInt(1<<23), "value"); err == nil {
		t.Fatal("expected positive int24 overflow error")
	}
	if _, err := int24ToInt32(big.NewInt(-(1<<23)-1), "value"); err == nil {
		t.Fatal("expected negative int24 overflow error")
	}
}
