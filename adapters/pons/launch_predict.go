package pons

import (
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var ErrInvalidLaunchDeployment = errors.New("invalid Pons launch deployment")

// LaunchDeployerRuntimeCodeHash pins the exact deployed bytecode whose
// embedded creation code is used by PredictLaunch.
var launchDeployerRuntimeCodeHash = common.HexToHash("0xeade22566c766377f6adfb99534f2772251efad9568642c0704a7051418e624c")

func ExpectedLaunchDeployerRuntimeCodeHash() common.Hash { return launchDeployerRuntimeCodeHash }

func VerifyLaunchDeployerRuntimeCode(code []byte) error {
	if len(code) == 0 {
		return fmt.Errorf("%w: launch deployer has no code", ErrInvalidLaunchDeployment)
	}
	actual := crypto.Keccak256Hash(code)
	if actual != launchDeployerRuntimeCodeHash {
		return fmt.Errorf("%w: launch deployer code hash %s, expected %s", ErrInvalidLaunchDeployment, actual, launchDeployerRuntimeCodeHash)
	}
	return nil
}

var (
	addressABI, _ = abi.NewType("address", "", nil)
	uint256ABI, _ = abi.NewType("uint256", "", nil)
	boolABI, _    = abi.NewType("bool", "", nil)
	bytes32ABI, _ = abi.NewType("bytes32", "", nil)
	uint24ABI, _  = abi.NewType("uint24", "", nil)
	int24ABI, _   = abi.NewType("int24", "", nil)
	stringABI, _  = abi.NewType("string", "", nil)
	feePolicyABI  = mustLaunchABIType("tuple", []abi.ArgumentMarshaling{
		{Name: "protocolFeeRecipient", Type: "address"}, {Name: "protocolFeeShareBps", Type: "uint16"},
		{Name: "buybackBurnBps", Type: "uint16"}, {Name: "hookFeeBps", Type: "uint16"},
		{Name: "maxInternalPriceImpactBps", Type: "uint16"},
	})
	socialsABI = mustLaunchABIType("tuple", []abi.ArgumentMarshaling{
		{Name: "twitter", Type: "string"}, {Name: "telegram", Type: "string"}, {Name: "discord", Type: "string"},
		{Name: "website", Type: "string"}, {Name: "farcaster", Type: "string"},
	})
)

func mustLaunchABIType(kind string, components []abi.ArgumentMarshaling) abi.Type {
	typ, err := abi.NewType(kind, "", components)
	if err != nil {
		panic(err)
	}
	return typ
}

// NewLaunchDeployment resolves the same defaults and quote economics as
// PonsV2LaunchFactory._launchToken. Callers must supply a confirmed/pending
// local snapshot of config, policy and pair economics.
func NewLaunchDeployment(params TokenParams, config LaunchConfig, pairEconomics PairTokenEconomics, policy FeePolicy, pairToken, originalDeployer common.Address, addresses Addresses) (LaunchDeployment, error) {
	if !config.Enabled {
		return LaunchDeployment{}, fmt.Errorf("%w: launch config is disabled", ErrInvalidLaunchDeployment)
	}
	creator := params.CreatorFeeRecipient
	if creator == (common.Address{}) {
		creator = originalDeployer
	}
	phantom, threshold := config.PhantomQuote, config.GraduationThreshold
	if pairToken != (common.Address{}) {
		phantom, threshold = pairEconomics.PhantomQuote, pairEconomics.GraduationThreshold
	}
	deployment := LaunchDeployment{
		PairToken: pairToken, CreatorFeeRecipient: creator, OriginalDeployer: originalDeployer,
		FeePolicy: addresses.MemeHook, Policy: policy, FeeEscrow: addresses.FeeEscrow, BuybackVault: addresses.BuybackVault,
		PhantomQuote: cloneBig(phantom), CurveFeeBps: cloneBig(config.CurveFeeBps), CreatorTaxBps: new(big.Int).SetUint64(uint64(params.CreatorTaxBps)),
		BuybackEnabled: params.BuybackEnabled, GraduationThreshold: cloneBig(threshold), Supply: cloneBig(config.Supply),
		PoolFee: config.PoolFee, TickSpacing: config.TickSpacing, ExpectedEconomics: params.ExpectedEconomics, Salt: params.Salt,
		Name: params.Name, Symbol: params.Symbol, Logo: params.Logo, Description: params.Description, Socials: params.Socials,
	}
	if err := validateLaunchDeployment(deployment, addresses); err != nil {
		return LaunchDeployment{}, err
	}
	if params.ExpectedEconomics != ([32]byte{}) {
		actual, err := LaunchEconomicsDigest(deployment)
		if err != nil {
			return LaunchDeployment{}, err
		}
		if actual != common.Hash(params.ExpectedEconomics) {
			return LaunchDeployment{}, fmt.Errorf("%w: economics expected %s, actual %s", ErrInvalidLaunchDeployment, common.Hash(params.ExpectedEconomics), actual)
		}
	}
	return deployment, nil
}

// LaunchEconomicsDigest mirrors PonsV2LaunchFactory's economics commitment.
// It is deterministic and performs no network I/O.
func LaunchEconomicsDigest(d LaunchDeployment) (common.Hash, error) {
	if d.PhantomQuote == nil || d.GraduationThreshold == nil || d.Supply == nil || d.CurveFeeBps == nil {
		return common.Hash{}, fmt.Errorf("%w: incomplete economics", ErrInvalidLaunchDeployment)
	}
	encoded, err := (abi.Arguments{
		{Type: uint256ABI}, {Type: uint256ABI}, {Type: uint256ABI}, {Type: uint256ABI},
		{Type: uint24ABI}, {Type: int24ABI}, {Type: uint256ABI}, {Type: uint256ABI},
		{Type: uint256ABI}, {Type: uint256ABI},
	}).Pack(d.PhantomQuote, d.GraduationThreshold, d.Supply, d.CurveFeeBps,
		new(big.Int).SetUint64(uint64(d.PoolFee)), big.NewInt(int64(d.TickSpacing)),
		new(big.Int).SetUint64(uint64(d.Policy.ProtocolFeeShareBps)), new(big.Int).SetUint64(uint64(d.Policy.BuybackBurnBps)),
		new(big.Int).SetUint64(uint64(d.Policy.HookFeeBps)), new(big.Int).SetUint64(uint64(d.Policy.MaxInternalPriceImpactBps)))
	if err != nil {
		return common.Hash{}, fmt.Errorf("encode launch economics: %w", err)
	}
	return crypto.Keccak256Hash(encoded), nil
}

// PredictLaunch computes both CREATE2 addresses locally. It performs no RPC
// calls and uses creation code extracted from Sourcify's exact-match build of
// the deployed launch deployer.
func PredictLaunch(deployment LaunchDeployment, addresses Addresses) (PredictedLaunch, error) {
	if err := validateLaunchDeployment(deployment, addresses); err != nil {
		return PredictedLaunch{}, err
	}
	namespace, err := (abi.Arguments{{Type: addressABI}, {Type: bytes32ABI}}).Pack(deployment.OriginalDeployer, deployment.Salt)
	if err != nil {
		return PredictedLaunch{}, err
	}
	salt := crypto.Keccak256Hash(namespace)
	curveArgs, err := (abi.Arguments{
		{Type: addressABI}, {Type: addressABI}, {Type: addressABI}, {Type: addressABI}, {Type: feePolicyABI}, {Type: addressABI},
		{Type: addressABI}, {Type: uint256ABI}, {Type: uint256ABI}, {Type: uint256ABI}, {Type: boolABI}, {Type: uint256ABI},
	}).Pack(deployment.PairToken, deployment.CreatorFeeRecipient, addresses.Factory, deployment.FeePolicy, deployment.Policy,
		deployment.FeeEscrow, deployment.BuybackVault, deployment.PhantomQuote, deployment.CurveFeeBps, deployment.CreatorTaxBps,
		deployment.BuybackEnabled, deployment.GraduationThreshold)
	if err != nil {
		return PredictedLaunch{}, fmt.Errorf("encode curve constructor: %w", err)
	}
	curve := crypto.CreateAddress2(addresses.LaunchDeployer, salt, crypto.Keccak256(appendCode(verifiedCurveCreationCode[:verifiedCurveCreationCodeLength], curveArgs)))
	tokenArgs, err := (abi.Arguments{
		{Type: stringABI}, {Type: stringABI}, {Type: stringABI}, {Type: stringABI}, {Type: socialsABI},
		{Type: addressABI}, {Type: addressABI}, {Type: addressABI}, {Type: uint256ABI},
	}).Pack(deployment.Name, deployment.Symbol, deployment.Logo, deployment.Description, deployment.Socials,
		deployment.OriginalDeployer, curve, addresses.Factory, deployment.Supply)
	if err != nil {
		return PredictedLaunch{}, fmt.Errorf("encode token constructor: %w", err)
	}
	token := crypto.CreateAddress2(addresses.LaunchDeployer, salt, crypto.Keccak256(appendCode(verifiedTokenCreationCode, tokenArgs)))
	reserved, err := ReservedTokensForPool(deployment.Supply, deployment.PhantomQuote, deployment.GraduationThreshold)
	if err != nil || reserved.Sign() == 0 || reserved.Cmp(deployment.Supply) >= 0 {
		return PredictedLaunch{}, fmt.Errorf("%w: invalid reserved token allocation", ErrInvalidLaunchDeployment)
	}
	state := CurveState{
		Curve: curve, Token: token, PairToken: deployment.PairToken, Deployer: deployment.CreatorFeeRecipient, Factory: addresses.Factory,
		Reserves:     CurveReserves{QuoteReserve: cloneBig(deployment.PhantomQuote), TokenReserve: cloneBig(deployment.Supply)},
		PhantomQuote: cloneBig(deployment.PhantomQuote), RealQuoteReserve: new(big.Int), GraduationThreshold: cloneBig(deployment.GraduationThreshold),
		SellableTokens: new(big.Int).Sub(cloneBig(deployment.Supply), reserved), ReservedTokens: reserved,
		TrackedQuote: new(big.Int), TrackedTokens: cloneBig(deployment.Supply), QuoteFeeBalance: new(big.Int),
		BuybackQuoteBalance: new(big.Int), CreatorTaxBalance: new(big.Int), FeeBps: cloneBig(deployment.CurveFeeBps),
		CreatorTaxBps: cloneBig(deployment.CreatorTaxBps), BuybackEnabled: deployment.BuybackEnabled,
	}
	return PredictedLaunch{Token: token, Curve: curve, State: state}, nil
}

func validateLaunchDeployment(d LaunchDeployment, addresses Addresses) error {
	if err := addresses.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidLaunchDeployment, err)
	}
	if addresses.Factory == (common.Address{}) || addresses.LaunchDeployer == (common.Address{}) || d.OriginalDeployer == (common.Address{}) ||
		d.CreatorFeeRecipient == (common.Address{}) || d.FeePolicy == (common.Address{}) || d.FeeEscrow == (common.Address{}) || d.BuybackVault == (common.Address{}) ||
		d.Policy.ProtocolFeeRecipient == (common.Address{}) || d.Name == "" || d.Symbol == "" {
		return fmt.Errorf("%w: required address or metadata is empty", ErrInvalidLaunchDeployment)
	}
	if deploymentAddressMismatch(d, addresses) {
		return fmt.Errorf("%w: deployment address snapshot mismatch", ErrInvalidLaunchDeployment)
	}
	for name, value := range map[string]*big.Int{"phantom quote": d.PhantomQuote, "curve fee": d.CurveFeeBps, "creator tax": d.CreatorTaxBps, "threshold": d.GraduationThreshold, "supply": d.Supply} {
		if value == nil || value.Sign() < 0 || value.BitLen() > 256 {
			return fmt.Errorf("%w: %s", ErrInvalidLaunchDeployment, name)
		}
	}
	if d.PhantomQuote.Sign() == 0 || d.GraduationThreshold.Sign() == 0 || d.Supply.Sign() == 0 {
		return fmt.Errorf("%w: economics must be positive", ErrInvalidLaunchDeployment)
	}
	if d.CurveFeeBps.Cmp(bpsInt) > 0 || d.CreatorTaxBps.Cmp(bpsInt) > 0 || d.Policy.ProtocolFeeShareBps > uint16(BPS) ||
		d.Policy.BuybackBurnBps > uint16(BPS) || d.Policy.HookFeeBps > uint16(BPS) || d.Policy.MaxInternalPriceImpactBps > uint16(BPS) ||
		d.PoolFee > 1<<24-1 || d.TickSpacing < -(1<<23) || d.TickSpacing > 1<<23-1 {
		return fmt.Errorf("%w: economics field exceeds contract range", ErrInvalidLaunchDeployment)
	}
	if len(d.Name) > MaxTokenNameBytes || len(d.Symbol) > MaxTokenSymbolBytes || len(d.Logo) > MaxTokenLogoBytes || len(d.Description) > MaxTokenDescriptionBytes ||
		len(d.Socials.Twitter) > MaxTokenSocialBytes || len(d.Socials.Telegram) > MaxTokenSocialBytes || len(d.Socials.Discord) > MaxTokenSocialBytes || len(d.Socials.Website) > MaxTokenSocialBytes || len(d.Socials.Farcaster) > MaxTokenSocialBytes {
		return fmt.Errorf("%w: metadata exceeds contract limit", ErrInvalidLaunchDeployment)
	}
	return nil
}

func deploymentAddressMismatch(d LaunchDeployment, addresses Addresses) bool {
	return d.FeePolicy != addresses.MemeHook || d.FeeEscrow != addresses.FeeEscrow || d.BuybackVault != addresses.BuybackVault
}

func appendCode(code, args []byte) []byte {
	result := make([]byte, 0, len(code)+len(args))
	result = append(result, code...)
	return append(result, args...)
}

func decodeVerifiedCreationCode(value string) []byte {
	decoded := common.FromHex(strings.TrimSpace(value))
	if len(decoded) == 0 {
		panic("empty verified Pons creation code")
	}
	return decoded
}
