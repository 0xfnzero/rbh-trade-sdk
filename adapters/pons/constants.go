package pons

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

const (
	ChainID int64 = 4663
	// PonsV2FactoryDeploymentBlock is the first block containing runtime code
	// at the official factory address on Robinhood Chain mainnet.
	PonsV2FactoryDeploymentBlock uint64 = 57_412_596
	MaxSnipeTaxExemptions        int    = 32
	MaxTokenNameBytes                   = 64
	MaxTokenSymbolBytes                 = 16
	MaxTokenLogoBytes                   = 512
	MaxTokenDescriptionBytes            = 2048
	MaxTokenSocialBytes                 = 256
)

var (
	RobinhoodChainID = big.NewInt(ChainID)
	NativeQuote      = common.Address{}
)

type Addresses struct {
	Factory            common.Address
	MemeHook           common.Address
	FeeEscrow          common.Address
	BuybackVault       common.Address
	LaunchLocker       common.Address
	LaunchAndBuy       common.Address
	LaunchDeployer     common.Address
	GraduationExecutor common.Address
	GraduationGuard    common.Address
}

var defaultPonsV2Addresses = Addresses{
	Factory:            common.HexToAddress("0x7eD598BcEf8bd9Edd8C97A195C6d13f40801EC7e"),
	MemeHook:           common.HexToAddress("0xE5e702641Ea86F4ae6cC3cDaeD2B886f976Be044"),
	FeeEscrow:          common.HexToAddress("0xd3AFEB2a57f70eF218Aa82451c51B2fb0416Ac9e"),
	BuybackVault:       common.HexToAddress("0x42df2a798f82289E177311362e8f5ccC45c1219c"),
	LaunchLocker:       common.HexToAddress("0x267444D099b10fB5Ed7c3Cc7B7c767AdcA574952"),
	LaunchAndBuy:       common.HexToAddress("0xe33E9E479dF8802cb0866d5d05258bEc4cF62948"),
	LaunchDeployer:     common.HexToAddress("0x3711ceA4feaDE896C913C68F01Eda97Cb06D1A42"),
	GraduationExecutor: common.HexToAddress("0xC7819B64A1dAECD7eC19856d026cb14EfBd89046"),
	GraduationGuard:    common.HexToAddress("0xf5695117b99B6f6401e67d4195BD653628176C6C"),
}

// PonsV2Addresses is a copy of the official deployment addresses for callers
// that need to inspect or pass them explicitly. New clients use an internal
// immutable copy, so external mutation cannot alter future defaults.
var PonsV2Addresses = defaultPonsV2Addresses
