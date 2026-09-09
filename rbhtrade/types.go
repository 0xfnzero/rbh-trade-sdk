package rbhtrade

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type Call struct {
	To    common.Address
	Data  []byte
	Value *big.Int
}

type PoolKey struct {
	Currency0   common.Address `abi:"currency0"`
	Currency1   common.Address `abi:"currency1"`
	Fee         uint32         `abi:"fee"`
	TickSpacing int32          `abi:"tickSpacing"`
	Hooks       common.Address `abi:"hooks"`
}

type poolKeyABI struct {
	Currency0   common.Address `abi:"currency0"`
	Currency1   common.Address `abi:"currency1"`
	Fee         *big.Int       `abi:"fee"`
	TickSpacing *big.Int       `abi:"tickSpacing"`
	Hooks       common.Address `abi:"hooks"`
}

type exactInputSingleABI struct {
	PoolKey          poolKeyABI `abi:"poolKey"`
	ZeroForOne       bool       `abi:"zeroForOne"`
	AmountIn         *big.Int   `abi:"amountIn"`
	AmountOutMinimum *big.Int   `abi:"amountOutMinimum"`
	MinHopPriceX36   *big.Int   `abi:"minHopPriceX36"`
	HookData         []byte     `abi:"hookData"`
}

type ExactInputRequest struct {
	PoolKey          PoolKey
	CurrencyIn       common.Address
	CurrencyOut      common.Address
	AmountIn         *big.Int
	AmountOutMinimum *big.Int
	MinHopPriceX36   *big.Int
	HookData         []byte
	Deadline         uint64
}

type QuoteExactInputRequest struct {
	PoolKey    PoolKey
	CurrencyIn common.Address
	AmountIn   *big.Int
	HookData   []byte
}

type QuoteResult struct {
	AmountOut   *big.Int
	GasEstimate *big.Int
}

type Slot0 struct {
	SqrtPriceX96 *big.Int
	Tick         int32
	ProtocolFee  uint32
	LPFee        uint32
}

// BagsTokenState is the exact state exposed by the verified Bags Lens for one
// token. Reserve and price values use the contracts' native integer units.
type BagsTokenState struct {
	Exists               bool
	Migrated             bool
	Curve                common.Address
	FeeShare             common.Address
	PoolID               common.Hash
	ThresholdQuote       *big.Int
	RealQuoteReserves    *big.Int
	RealTokenReserves    *big.Int
	VirtualTokenReserves *big.Int
	VirtualQuoteReserves *big.Int
	PriceQuotePerToken   *big.Int
	BondingProgressPct   *big.Int
	TotalRaised          *big.Int
}

type BagsBuyQuote struct {
	TokensOut   *big.Int
	FeeQuote    *big.Int
	NetQuoteIn  *big.Int
	GrossUsed   *big.Int
	RefundQuote *big.Int
}

type BagsSellQuote struct {
	QuoteToSeller *big.Int
	FeeQuote      *big.Int
	GrossQuoteOut *big.Int
}

type LongCreateParams struct {
	InitialSupply         *big.Int       `abi:"initialSupply"`
	NumTokensToSell       *big.Int       `abi:"numTokensToSell"`
	Numeraire             common.Address `abi:"numeraire"`
	TokenFactory          common.Address `abi:"tokenFactory"`
	TokenFactoryData      []byte         `abi:"tokenFactoryData"`
	GovernanceFactory     common.Address `abi:"governanceFactory"`
	GovernanceFactoryData []byte         `abi:"governanceFactoryData"`
	PoolInitializer       common.Address `abi:"poolInitializer"`
	PoolInitializerData   []byte         `abi:"poolInitializerData"`
	LiquidityMigrator     common.Address `abi:"liquidityMigrator"`
	LiquidityMigratorData []byte         `abi:"liquidityMigratorData"`
	Integrator            common.Address `abi:"integrator"`
	Salt                  [32]byte       `abi:"salt"`
}

type O1LaunchParams struct {
	TokenName             string         `abi:"tokenName"`
	TokenSymbol           string         `abi:"tokenSymbol"`
	TokenContractURI      string         `abi:"tokenContractURI"`
	CreatorSalt           [32]byte       `abi:"creatorSalt"`
	QuoteToken            common.Address `abi:"quoteToken"`
	ExpectedConfigVersion uint64         `abi:"expectedConfigVersion"`
	Deadline              uint64         `abi:"deadline"`
	MetadataEditable      bool           `abi:"metadataEditable"`
	MetadataKeys          []string       `abi:"metadataKeys"`
	MetadataValues        []string       `abi:"metadataValues"`
}

type O1LaunchBuyParams struct {
	FundingToken common.Address `abi:"fundingToken"`
	AmountIn     *big.Int       `abi:"amountIn"`
	MinAmountOut *big.Int       `abi:"minAmountOut"`
	RouteData    []byte         `abi:"routeData"`
}

type BagsLaunchParams struct {
	Name        string
	Symbol      string
	MetadataURI string
	Partner     common.Address
	Claimers    []common.Address
	BPS         []uint16
}

type PAIRAllocation struct {
	QuoteToken common.Address `abi:"quoteToken"`
	WeightBPS  uint16         `abi:"weightBps"`
}

type PAIRLaunchParams struct {
	Name                    string           `abi:"name"`
	Symbol                  string           `abi:"symbol"`
	MetadataURI             string           `abi:"metadataURI"`
	MetadataHash            [32]byte         `abi:"metadataHash"`
	Allocations             []PAIRAllocation `abi:"allocations"`
	CreatorFeeRecipient     common.Address   `abi:"creatorFeeRecipient"`
	DeveloperBuyRecipient   common.Address   `abi:"developerBuyRecipient"`
	DeveloperBuyPairIndex   uint8            `abi:"developerBuyPairIndex"`
	DeveloperTokenAmountOut *big.Int         `abi:"developerTokenAmountOut"`
	MaxQuoteAmountIn        *big.Int         `abi:"maxQuoteAmountIn"`
	Deadline                *big.Int         `abi:"deadline"`
}

type PoolsDistribution struct {
	Strategy   common.Address `abi:"strategy"`
	Amount     *big.Int       `abi:"amount"`
	ConfigData []byte         `abi:"configData"`
}
