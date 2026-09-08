package pons

import (
	"bytes"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestPackBuy(t *testing.T) {
	data, err := PackBuy(big.NewInt(1), big.NewInt(2), common.HexToAddress("0x0000000000000000000000000000000000000001"))
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 4+32*3 {
		t.Fatalf("unexpected calldata length %d", len(data))
	}
}

func TestPackLaunchToken(t *testing.T) {
	params := TokenParams{
		Name:        "Example",
		Symbol:      "EXMPL",
		Logo:        "ipfs://example",
		Description: "example launch",
		Socials: Socials{
			Twitter:   "x",
			Telegram:  "tg",
			Discord:   "dc",
			Website:   "https://example.com",
			Farcaster: "fc",
		},
		CreatorFeeRecipient: common.HexToAddress("0x0000000000000000000000000000000000000001"),
		CreatorTaxBps:       100,
		BuybackEnabled:      true,
	}
	data, err := PackLaunchToken(params, big.NewInt(0), NativeQuote, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) <= 4 {
		t.Fatalf("unexpected calldata length %d", len(data))
	}
}

func TestPackLaunchTokenSimple(t *testing.T) {
	params := TokenParams{Name: "Example", Symbol: "EXMPL"}
	data, err := PackLaunchTokenSimple(params, big.NewInt(0), NativeQuote)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) <= 4 {
		t.Fatalf("unexpected calldata length %d", len(data))
	}
}

func TestPackLaunchRejectsOversizedMetadata(t *testing.T) {
	tests := []TokenParams{
		{Name: string(make([]byte, MaxTokenNameBytes+1)), Symbol: "OK"},
		{Name: "OK", Symbol: string(make([]byte, MaxTokenSymbolBytes+1))},
		{Name: "OK", Symbol: "OK", Logo: string(make([]byte, MaxTokenLogoBytes+1))},
		{Name: "OK", Symbol: "OK", Description: string(make([]byte, MaxTokenDescriptionBytes+1))},
		{Name: "OK", Symbol: "OK", Socials: Socials{Website: string(make([]byte, MaxTokenSocialBytes+1))}},
	}
	for i, params := range tests {
		if _, err := PackLaunchTokenSimple(params, big.NewInt(0), NativeQuote); err == nil {
			t.Fatalf("case %d: expected metadata length error", i)
		}
	}
}

func TestInternalABIsAreIndependentFromExports(t *testing.T) {
	original := CurveABI
	CurveABI = abi.ABI{}
	t.Cleanup(func() { CurveABI = original })
	if _, err := PackBuy(big.NewInt(1), big.NewInt(0), common.HexToAddress("0x1")); err != nil {
		t.Fatalf("exported ABI mutation affected internal calldata: %v", err)
	}
}

func TestPackOperationalCalls(t *testing.T) {
	token := common.HexToAddress("0x0000000000000000000000000000000000000001")
	poolID := common.HexToHash("0x010203")
	cases := []struct {
		name string
		fn   func() ([]byte, error)
	}{
		{"graduate", func() ([]byte, error) { return PackGraduate(token) }},
		{"createGraduatedPool", func() ([]byte, error) { return PackCreateGraduatedPool(token) }},
		{"sweepCurveFees", func() ([]byte, error) { return PackSweepCurveFees(big.NewInt(1)) }},
		{"sweepPoolFees", func() ([]byte, error) { return PackSweepPoolFees(poolID, big.NewInt(1), big.NewInt(2)) }},
		{"claimNativeAmount", func() ([]byte, error) { return PackClaimNativeAmount(big.NewInt(3)) }},
		{"claimTokenAmount", func() ([]byte, error) { return PackClaimTokenAmount(token, big.NewInt(4)) }},
		{"releaseVested", func() ([]byte, error) { return PackReleaseVested(token) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := tc.fn()
			if err != nil {
				t.Fatal(err)
			}
			if len(data) < 4 {
				t.Fatalf("unexpected calldata length %d", len(data))
			}
		})
	}
}

func TestParseContractError(t *testing.T) {
	abiErr := ErrorsABI.Errors["SlippageExceeded"]
	args, err := abiErr.Inputs.Pack(big.NewInt(10), big.NewInt(11))
	if err != nil {
		t.Fatal(err)
	}
	data := append(abiErr.ID[:4], args...)
	decoded, ok, err := ParseContractError(data)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected custom error")
	}
	if decoded.Name != "SlippageExceeded" {
		t.Fatalf("unexpected error name %s", decoded.Name)
	}
	if decoded.Values["actual"].(*big.Int).Cmp(big.NewInt(10)) != 0 {
		t.Fatalf("unexpected actual value %#v", decoded.Values)
	}
}

func TestParseContractPanicAndIgnoreExportedABIMutation(t *testing.T) {
	original := ErrorsABI
	ErrorsABI = abi.ABI{}
	t.Cleanup(func() { ErrorsABI = original })

	data := append([]byte{0x4e, 0x48, 0x7b, 0x71}, common.LeftPadBytes([]byte{0x11}, 32)...)
	decoded, ok, err := ParseContractError(data)
	if err != nil || !ok {
		t.Fatalf("panic parse failed: decoded=%#v ok=%v err=%v", decoded, ok, err)
	}
	if decoded.Name != "Panic" || decoded.Values["reason"] != "arithmetic underflow or overflow" {
		t.Fatalf("unexpected panic: %#v", decoded)
	}

	custom := errorsContractABI.Errors["ZeroAmount"]
	decoded, ok, err = ParseContractError(custom.ID[:4])
	if err != nil || !ok || decoded.Name != "ZeroAmount" {
		t.Fatalf("internal error ABI was affected by export mutation: decoded=%#v ok=%v err=%v", decoded, ok, err)
	}
}

func TestParseContractErrorRejectsTrailingData(t *testing.T) {
	custom := errorsContractABI.Errors["ZeroAmount"]
	if _, _, err := ParseContractError(append(custom.ID[:4], make([]byte, 32)...)); err == nil {
		t.Fatal("expected trailing custom error data to be rejected")
	}
	panicData := append([]byte{0x4e, 0x48, 0x7b, 0x71}, common.LeftPadBytes([]byte{0x11}, 32)...)
	if _, _, err := ParseContractError(append(panicData, make([]byte, 32)...)); err == nil {
		t.Fatal("expected trailing panic data to be rejected")
	}
}

func TestParseContractErrorStringCanonicalAndBounded(t *testing.T) {
	typ, err := abi.NewType("string", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	args, err := (abi.Arguments{{Type: typ}}).Pack("denied")
	if err != nil {
		t.Fatal(err)
	}
	data := append([]byte{0x08, 0xc3, 0x79, 0xa0}, args...)
	decoded, ok, err := ParseContractError(data)
	if err != nil || !ok || decoded.Name != "Error" || decoded.Values["reason"] != "denied" {
		t.Fatalf("decoded=%#v ok=%v err=%v", decoded, ok, err)
	}
	if _, _, err := ParseContractError(append(data, make([]byte, 32)...)); err == nil {
		t.Fatal("expected trailing revert data to be rejected")
	}
	oversized := make([]byte, MaxRevertDataBytes+1)
	copy(oversized, data[:4])
	if _, _, err := ParseContractError(oversized); err == nil {
		t.Fatal("expected oversized revert data to be rejected")
	}
}

func TestContractErrorIndexMarksSelectorCollisionsAmbiguous(t *testing.T) {
	first := abi.NewError("First", nil)
	second := abi.NewError("Second", nil)
	copy(second.ID[:4], first.ID[:4])
	indexed := indexErrors(abi.ABI{Errors: map[string]abi.Error{"First": first, "Second": second}})
	var selector [4]byte
	copy(selector[:], first.ID[:4])
	if !indexed[selector].ambiguous {
		t.Fatal("selector collision was not marked ambiguous")
	}
}

func TestOfficialContractErrorSelectorsAreUnique(t *testing.T) {
	seen := make(map[[4]byte]string, len(errorsContractABI.Errors))
	for name, definition := range errorsContractABI.Errors {
		var selector [4]byte
		copy(selector[:], definition.ID[:4])
		if previous, ok := seen[selector]; ok {
			t.Fatalf("selector collision between %s and %s", previous, name)
		}
		seen[selector] = name
	}
}

func TestValueHelpersFollowDocs(t *testing.T) {
	launchFee := big.NewInt(100)
	quoteIn := big.NewInt(25)
	erc20 := common.HexToAddress("0x0000000000000000000000000000000000000001")

	if got := BuyValue(NativeQuote, quoteIn); got.Cmp(quoteIn) != 0 {
		t.Fatalf("native buy value = %s", got)
	}
	if got := BuyValue(erc20, quoteIn); got.Sign() != 0 {
		t.Fatalf("erc20 buy value = %s", got)
	}
	if got := LaunchAndBuyValue(NativeQuote, launchFee, quoteIn); got.Cmp(big.NewInt(125)) != 0 {
		t.Fatalf("native launch-and-buy value = %s", got)
	}
	if got := LaunchAndBuyValue(erc20, launchFee, quoteIn); got.Cmp(launchFee) != 0 {
		t.Fatalf("erc20 launch-and-buy value = %s", got)
	}
	if got := LaunchAndBuyValue(NativeQuote, launchFee, nil); got.Cmp(launchFee) != 0 {
		t.Fatalf("native launch-and-buy value with nil quote = %s", got)
	}
}

func TestNativeQuoteDetectionDoesNotDependOnExportedVariable(t *testing.T) {
	original := NativeQuote
	NativeQuote = common.HexToAddress("0x0000000000000000000000000000000000000001")
	t.Cleanup(func() { NativeQuote = original })
	if !IsNativeQuote(common.Address{}) {
		t.Fatal("zero address must remain the native quote sentinel")
	}
	if IsNativeQuote(NativeQuote) {
		t.Fatal("mutating the compatibility variable must not change native quote detection")
	}
}

func TestPoolID(t *testing.T) {
	key := PoolKey{
		Currency0:   common.Address{},
		Currency1:   common.HexToAddress("0x0000000000000000000000000000000000000001"),
		Fee:         0,
		TickSpacing: 60,
		Hooks:       PonsV2Addresses.MemeHook,
	}
	id, err := PoolID(key)
	if err != nil {
		t.Fatal(err)
	}
	if id == (common.Hash{}) {
		t.Fatal("expected non-zero pool id")
	}
}

func TestPoolIDRejectsInvalidKeys(t *testing.T) {
	address := common.HexToAddress("0x0000000000000000000000000000000000000001")
	tests := []PoolKey{
		{Currency0: address, Currency1: address},
		{Currency0: address, Currency1: common.Address{}},
		{Currency0: common.Address{}, Currency1: address, Fee: 1 << 24},
		{Currency0: common.Address{}, Currency1: address, TickSpacing: 1 << 23},
	}
	for _, key := range tests {
		if _, err := PoolID(key); !errors.Is(err, ErrInvalidPoolKey) {
			t.Fatalf("PoolID(%#v) error = %v", key, err)
		}
	}
}

func TestOfficialMethodSignatures(t *testing.T) {
	tests := []struct {
		name      string
		contract  abi.ABI
		method    string
		signature string
	}{
		{"buy", CurveABI, "buy", "buy(uint256,uint256,address)"},
		{"sell", CurveABI, "sell", "sell(uint256,uint256,address)"},
		{"launchTokenSimple", FactoryLaunchSimpleABI, "launchToken", "launchToken((string,string,string,string,(string,string,string,string,string),address,uint16,bool,bytes32,bytes32),uint256,address)"},
		{"launchTokenWithExemptions", FactoryLaunchABI, "launchToken", "launchToken((string,string,string,string,(string,string,string,string,string),address,uint16,bool,bytes32,bytes32),uint256,address,address[])"},
		{"launchAndBuy", RouterABI, "launchAndBuy", "launchAndBuy((string,string,string,string,(string,string,string,string,string),address,uint16,bool,bytes32,bytes32),uint256,address,uint256,uint256,address,address[])"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			method := tc.contract.Methods[tc.method]
			if method.Sig != tc.signature {
				t.Fatalf("signature = %s, want %s", method.Sig, tc.signature)
			}
			wantID := crypto.Keccak256([]byte(tc.signature))[:4]
			if !bytes.Equal(method.ID, wantID) {
				t.Fatalf("selector = 0x%x, want 0x%x", method.ID, wantID)
			}
		})
	}
}
