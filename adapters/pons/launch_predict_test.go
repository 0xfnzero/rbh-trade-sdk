package pons

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestPredictLaunchMatchesVerifiedHistoricalDeployment(t *testing.T) {
	supply, ok := new(big.Int).SetString("1000000000000000000000000000", 10)
	if !ok {
		t.Fatal("invalid fixture supply")
	}
	deployment := LaunchDeployment{
		PairToken:           common.HexToAddress("0x5fc5360D0400a0Fd4f2af552ADD042D716F1d168"),
		CreatorFeeRecipient: common.HexToAddress("0x8258639CD5C2A642977d5BD8851CA931f7096389"),
		OriginalDeployer:    common.HexToAddress("0x8258639CD5C2A642977d5BD8851CA931f7096389"),
		FeePolicy:           PonsV2Addresses.MemeHook,
		Policy:              FeePolicy{ProtocolFeeRecipient: common.HexToAddress("0x263ed295dAFaE1d9AAdD6E56c4B6F9f38eE019Dd"), ProtocolFeeShareBps: 3000, BuybackBurnBps: 5000, HookFeeBps: 100, MaxInternalPriceImpactBps: 300},
		FeeEscrow:           PonsV2Addresses.FeeEscrow, BuybackVault: PonsV2Addresses.BuybackVault,
		PhantomQuote: big.NewInt(3_236_000_000), CurveFeeBps: big.NewInt(100), CreatorTaxBps: big.NewInt(200),
		GraduationThreshold: big.NewInt(8_090_000_000), Supply: supply,
		Salt: [32]byte(common.HexToHash("0x8a74587d996179285feae6546c627f11d5db71ac5ac2edcf98d74bf65dd2b948")),
		Name: "RetardsOutperformBanksInvestorsNormies", Symbol: "ROBIN",
		Logo:        "https://gateway.pinata.cloud/ipfs/bafkreifbntqxjgs64bqgyrjdekb5igy6jxmqawx23lwfu6u2bbnxso7j6e",
		Description: "robin", Socials: Socials{Twitter: "https://x.com/THE8MIX8/status/2097148948997697545"},
	}
	predicted, err := PredictLaunch(deployment, PonsV2Addresses)
	if err != nil {
		t.Fatal(err)
	}
	if want := common.HexToAddress("0xb0cfae1576fcd5bcfdac6bf5a2df6eb57a439db6"); predicted.Token != want {
		t.Fatalf("token = %s, want %s", predicted.Token, want)
	}
	if want := common.HexToAddress("0x572fa9743954b12ae9773b66303f888b953576b4"); predicted.Curve != want {
		t.Fatalf("curve = %s, want %s", predicted.Curve, want)
	}
	if predicted.State.Curve != predicted.Curve || predicted.State.Token != predicted.Token || predicted.State.Reserves.QuoteReserve.Cmp(deployment.PhantomQuote) != 0 || predicted.State.TrackedTokens.Cmp(supply) != 0 || predicted.State.SellableTokens.Sign() <= 0 {
		t.Fatalf("invalid initial state: %#v", predicted.State)
	}
}

func TestPredictLaunchRejectsIncompleteSnapshot(t *testing.T) {
	_, err := PredictLaunch(LaunchDeployment{}, PonsV2Addresses)
	if err == nil {
		t.Fatal("incomplete deployment accepted")
	}
}

func TestNewLaunchDeploymentRejectsEconomicsMismatch(t *testing.T) {
	params := TokenParams{Name: "Token", Symbol: "TKN", ExpectedEconomics: [32]byte{1}}
	config := LaunchConfig{Supply: big.NewInt(1_000_000), CurveFeeBps: big.NewInt(100), PhantomQuote: big.NewInt(10_000), GraduationThreshold: big.NewInt(100_000), Enabled: true}
	policy := FeePolicy{ProtocolFeeRecipient: common.HexToAddress("0x1000000000000000000000000000000000000001")}
	_, err := NewLaunchDeployment(params, config, PairTokenEconomics{}, policy, common.Address{}, common.HexToAddress("0x2000000000000000000000000000000000000002"), PonsV2Addresses)
	if !errors.Is(err, ErrInvalidLaunchDeployment) {
		t.Fatalf("economics mismatch error = %v", err)
	}
}

func TestVerifiedCreationCodeLengths(t *testing.T) {
	if len(verifiedCurveCreationCode) != 0x2cd8 {
		t.Fatalf("curve creation code length = %d", len(verifiedCurveCreationCode))
	}
	if len(verifiedTokenCreationCode) != 0x1c0a {
		t.Fatalf("token creation code length = %d", len(verifiedTokenCreationCode))
	}
}

func TestVerifyLaunchDeployerRuntimeCodeRejectsEmptyAndMismatch(t *testing.T) {
	if err := VerifyLaunchDeployerRuntimeCode(nil); !errors.Is(err, ErrInvalidLaunchDeployment) {
		t.Fatalf("empty code error = %v", err)
	}
	if err := VerifyLaunchDeployerRuntimeCode([]byte{1}); !errors.Is(err, ErrInvalidLaunchDeployment) {
		t.Fatalf("mismatched code error = %v", err)
	}
}
