package pons

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type Phase uint8

const (
	PhaseNotGraduated Phase = 0
	PhaseSwept        Phase = 1
	PhasePoolCreated  Phase = 2
	PhaseRescued      Phase = 3
)

type Socials struct {
	Twitter   string `abi:"twitter"`
	Telegram  string `abi:"telegram"`
	Discord   string `abi:"discord"`
	Website   string `abi:"website"`
	Farcaster string `abi:"farcaster"`
}

type TokenParams struct {
	Name                string         `abi:"name"`
	Symbol              string         `abi:"symbol"`
	Logo                string         `abi:"logo"`
	Description         string         `abi:"description"`
	Socials             Socials        `abi:"socials"`
	CreatorFeeRecipient common.Address `abi:"creatorFeeRecipient"`
	CreatorTaxBps       uint16         `abi:"creatorTaxBps"`
	BuybackEnabled      bool           `abi:"buybackEnabled"`
	ExpectedEconomics   [32]byte       `abi:"expectedEconomics"`
	Salt                [32]byte       `abi:"salt"`
}

type LaunchConfig struct {
	Supply              *big.Int `abi:"supply"`
	CurveFeeBps         *big.Int `abi:"curveFeeBps"`
	PhantomQuote        *big.Int `abi:"phantomQuote"`
	GraduationThreshold *big.Int `abi:"graduationThreshold"`
	PoolFee             uint32   `abi:"poolFee"`
	TickSpacing         int32    `abi:"tickSpacing"`
	Enabled             bool     `abi:"enabled"`
}

type launchConfigABI struct {
	Supply              *big.Int `abi:"supply"`
	CurveFeeBps         *big.Int `abi:"curveFeeBps"`
	PhantomQuote        *big.Int `abi:"phantomQuote"`
	GraduationThreshold *big.Int `abi:"graduationThreshold"`
	PoolFee             *big.Int `abi:"poolFee"`
	TickSpacing         *big.Int `abi:"tickSpacing"`
	Enabled             bool     `abi:"enabled"`
}

type OpenLaunchConfig struct {
	ID *big.Int
	LaunchConfig
}

type PairTokenEconomics struct {
	PhantomQuote        *big.Int
	GraduationThreshold *big.Int
	Decimals            uint8
}

type LaunchedToken struct {
	Token               common.Address `abi:"token"`
	Curve               common.Address `abi:"curve"`
	Deployer            common.Address `abi:"deployer"`
	CreatorFeeRecipient common.Address `abi:"creatorFeeRecipient"`
	PairToken           common.Address `abi:"pairToken"`
	GraduationThreshold *big.Int       `abi:"graduationThreshold"`
	PoolFee             uint32         `abi:"poolFee"`
	TickSpacing         int32          `abi:"tickSpacing"`
	CreatorTaxBps       uint16         `abi:"creatorTaxBps"`
	BuybackEnabled      bool           `abi:"buybackEnabled"`
	Phase               uint8          `abi:"phase"`
	SweptQuote          *big.Int       `abi:"sweptQuote"`
	SweptTokens         *big.Int       `abi:"sweptTokens"`
	SweptAt             *big.Int       `abi:"sweptAt"`
	Exists              bool           `abi:"exists"`
}

type launchedTokenABI struct {
	Token               common.Address `abi:"token"`
	Curve               common.Address `abi:"curve"`
	Deployer            common.Address `abi:"deployer"`
	CreatorFeeRecipient common.Address `abi:"creatorFeeRecipient"`
	PairToken           common.Address `abi:"pairToken"`
	GraduationThreshold *big.Int       `abi:"graduationThreshold"`
	PoolFee             *big.Int       `abi:"poolFee"`
	TickSpacing         *big.Int       `abi:"tickSpacing"`
	CreatorTaxBps       uint16         `abi:"creatorTaxBps"`
	BuybackEnabled      bool           `abi:"buybackEnabled"`
	Phase               uint8          `abi:"phase"`
	SweptQuote          *big.Int       `abi:"sweptQuote"`
	SweptTokens         *big.Int       `abi:"sweptTokens"`
	SweptAt             *big.Int       `abi:"sweptAt"`
	Exists              bool           `abi:"exists"`
}

type FeePolicy struct {
	ProtocolFeeRecipient      common.Address `abi:"protocolFeeRecipient"`
	ProtocolFeeShareBps       uint16         `abi:"protocolFeeShareBps"`
	BuybackBurnBps            uint16         `abi:"buybackBurnBps"`
	HookFeeBps                uint16         `abi:"hookFeeBps"`
	MaxInternalPriceImpactBps uint16         `abi:"maxInternalPriceImpactBps"`
}

// LaunchDeployment mirrors the exact tuple consumed by the verified
// PonsV2LaunchDeployer deployed on Robinhood Chain.
type LaunchDeployment struct {
	PairToken           common.Address `abi:"pairToken"`
	CreatorFeeRecipient common.Address `abi:"creatorFeeRecipient"`
	OriginalDeployer    common.Address `abi:"originalDeployer"`
	FeePolicy           common.Address `abi:"feePolicy"`
	Policy              FeePolicy      `abi:"policy"`
	FeeEscrow           common.Address `abi:"feeEscrow"`
	BuybackVault        common.Address `abi:"buybackVault"`
	PhantomQuote        *big.Int       `abi:"phantomQuote"`
	CurveFeeBps         *big.Int       `abi:"curveFeeBps"`
	CreatorTaxBps       *big.Int       `abi:"creatorTaxBps"`
	BuybackEnabled      bool           `abi:"buybackEnabled"`
	GraduationThreshold *big.Int       `abi:"graduationThreshold"`
	Supply              *big.Int       `abi:"supply"`
	PoolFee             uint32
	TickSpacing         int32
	ExpectedEconomics   [32]byte
	Salt                [32]byte `abi:"salt"`
	Name                string   `abi:"name"`
	Symbol              string   `abi:"symbol"`
	Logo                string   `abi:"logo"`
	Description         string   `abi:"description"`
	Socials             Socials  `abi:"socials"`
}

type PredictedLaunch struct {
	Token common.Address
	Curve common.Address
	State CurveState
}

type PendingCreatorFeeRecipient struct {
	NewRecipient common.Address
	EffectiveAt  *big.Int
	ExpiresAt    *big.Int
}

type TokenInfo struct {
	Deployer    common.Address
	Logo        string
	Description string
	Socials     Socials
}

type TokenMetadata struct {
	Name          string
	Symbol        string
	Decimals      uint8
	TotalSupply   *big.Int
	Deployer      common.Address
	LaunchFactory common.Address
	Curve         common.Address
	Logo          string
	Description   string
	Socials       Socials
}

type CurveReserves struct {
	QuoteReserve *big.Int
	TokenReserve *big.Int
}

type CurveState struct {
	Curve               common.Address
	Token               common.Address
	PairToken           common.Address
	Deployer            common.Address
	Factory             common.Address
	Reserves            CurveReserves
	PhantomQuote        *big.Int
	RealQuoteReserve    *big.Int
	GraduationThreshold *big.Int
	SellableTokens      *big.Int
	ReservedTokens      *big.Int
	TrackedQuote        *big.Int
	TrackedTokens       *big.Int
	QuoteFeeBalance     *big.Int
	BuybackQuoteBalance *big.Int
	CreatorTaxBalance   *big.Int
	ReadyToGraduate     bool
	Graduated           bool
	FeeBps              *big.Int
	CreatorTaxBps       *big.Int
	BuybackEnabled      bool
}

type QuoteBuyResult struct {
	TokensOut     *big.Int
	Spent         *big.Int
	Refund        *big.Int
	Fee           *big.Int
	Tax           *big.Int
	SnipeTax      *big.Int
	FeeBps        *big.Int
	CreatorTaxBps *big.Int
	SnipeTaxBps   *big.Int
}

type QuoteSellResult struct {
	QuoteOut      *big.Int
	GrossQuoteOut *big.Int
	Fee           *big.Int
	Tax           *big.Int
	FeeBps        *big.Int
	CreatorTaxBps *big.Int
}

type PoolKey struct {
	Currency0   common.Address
	Currency1   common.Address
	Fee         uint32
	TickSpacing int32
	Hooks       common.Address
}

type VestState struct {
	TotalLocked     *big.Int
	TotalReleased   *big.Int
	VestedAmount    *big.Int
	Releasable      *big.Int
	VestingStart    *big.Int
	VestingDuration *big.Int
	Terms           VestingTerms
}

type VestingTerms struct {
	CreatorRecipient    common.Address
	ProtocolRecipient   common.Address
	ProtocolFeeShareBps uint16
}

type HookLaunch struct {
	Registered                bool
	MemecoinIsCurrency0       bool
	Memecoin                  common.Address
	QuoteToken                common.Address
	Creator                   common.Address
	BuybackCreatorRecipient   common.Address
	ProtocolFeeRecipient      common.Address
	CreatorTaxBps             uint16
	ProtocolFeeShareBps       uint16
	BuybackBurnBps            uint16
	HookFeeBps                uint16
	MaxInternalPriceImpactBps uint16
	BuybackEnabled            bool
}
