package rbhtrade

import "github.com/ethereum/go-ethereum/common"

const ChainID int64 = 4663

var (
	NativeCurrency   = common.Address{} // Compatibility snapshot; builders use an internal zero value.
	RobinhoodChain   = newAddressBook() // Compatibility snapshot; mutating it does not affect builders.
	defaultAddresses = newAddressBook()
)

type AddressBook struct {
	PoolManager       common.Address
	PositionManager   common.Address
	V4Quoter          common.Address
	StateView         common.Address
	UniversalRouter   common.Address
	Permit2           common.Address
	WETH              common.Address
	USDG              common.Address
	V3Factory         common.Address
	V3SwapRouter02    common.Address
	PonsFactory       common.Address
	PonsLaunchAndBuy  common.Address
	PonsMemeHook      common.Address
	LongLauncher      common.Address
	LongAirlock       common.Address
	LongTokenFactory  common.Address
	LongDopplerHook   common.Address
	O1Factory         common.Address
	O1Hook            common.Address
	O1FeeEscrow       common.Address
	O1TokenDeployer   common.Address
	O1Registry        common.Address
	O1LaunchBuy       common.Address
	PoolsEntry        common.Address
	PoolsTokenFactory common.Address
	PoolsInstant      common.Address
	PAIRLaunchpad     common.Address
	PAIRLocker        common.Address
	BagsFactory       common.Address
	BagsHook          common.Address
	BagsVault         common.Address
	BagsLens          common.Address
}

func newAddressBook() AddressBook {
	return AddressBook{
		PoolManager:       common.HexToAddress("0x8366a39CC670B4001A1121B8F6A443A643e40951"),
		PositionManager:   common.HexToAddress("0x58daec3116aae6D93017bAAea7749052E8a04fA7"),
		V4Quoter:          common.HexToAddress("0x8dc178efb8111bb0973dd9d722ebeff267c98f94"),
		StateView:         common.HexToAddress("0xf3334192d15450cdd385c8b70e03f9a6bd9e673b"),
		UniversalRouter:   common.HexToAddress("0x8876789976dEcBfCbBbe364623C63652db8C0904"),
		Permit2:           common.HexToAddress("0x000000000022D473030F116dDEE9F6B43aC78BA3"),
		WETH:              common.HexToAddress("0x0Bd7D308f8E1639FAb988df18A8011F41EAcAD73"),
		USDG:              common.HexToAddress("0x5fc5360D0400a0Fd4f2af552ADD042D716F1d168"),
		V3Factory:         common.HexToAddress("0x1f7d7550B1b028f7571E69A784071F0205FD2EfA"),
		V3SwapRouter02:    common.HexToAddress("0xCaf681a66D020601342297493863E78C959E5cb2"),
		PonsFactory:       common.HexToAddress("0x7eD598BcEf8bd9Edd8C97A195C6d13f40801EC7e"),
		PonsLaunchAndBuy:  common.HexToAddress("0xe33E9E479dF8802cb0866d5d05258bEc4cF62948"),
		PonsMemeHook:      common.HexToAddress("0xE5e702641Ea86F4ae6cC3cDaeD2B886f976Be044"),
		LongLauncher:      common.HexToAddress("0x22e99278308B393ea1260859B181AD7E78f5eeED"),
		LongAirlock:       common.HexToAddress("0xeb7C034704eF8Dcd2D32324c1545f62fB4aD0862"),
		LongTokenFactory:  common.HexToAddress("0x1B37D3a72082029c44B35B604Ea473617580b69a"),
		LongDopplerHook:   common.HexToAddress("0x4e3468951D49f2EEa976eD0D6e75fFCb44a9a544"),
		O1Factory:         common.HexToAddress("0xcE9C48cFa068947f77738c81Be406B53338E5B0d"),
		O1Hook:            common.HexToAddress("0x0310cFEbE1D7A69f2414f6595bBe9d17c5342aCc"),
		O1FeeEscrow:       common.HexToAddress("0xc5444b417a04a7E1b9C1E327c7D499803c14E5EF"),
		O1TokenDeployer:   common.HexToAddress("0xf86dfDb678D8E5d932100Ef479A59fa65a82a5Eb"),
		O1Registry:        common.HexToAddress("0x19C4c024Aca11e4A3d47792C69C80c8f4E596b23"),
		O1LaunchBuy:       common.HexToAddress("0xF9804FeAB2F9b16EDE0Cd92E5C6e75C5cf64462f"),
		PoolsEntry:        common.HexToAddress("0x0000FffFBE8efE702c8703aE3477FF5dE3d319C0"),
		PoolsTokenFactory: common.HexToAddress("0x000000e200088D55C39a11F609E5F667729ad49b"),
		PoolsInstant:      common.HexToAddress("0x23f8209572b4a1C2AD88A42749E830791Fb027f1"),
		PAIRLaunchpad:     common.HexToAddress("0x8660A7F019C7943b0b0A91B8E39AFf3b6DB6Ae62"),
		PAIRLocker:        common.HexToAddress("0xeFcF476E8870fB3eb8680f039414fdcCE6C2a117"),
		BagsFactory:       common.HexToAddress("0xe8Cc4431adF8b5A847C113EF0c6af9043219Cb37"),
		BagsHook:          common.HexToAddress("0x2380aBf72C17aABAb76480244759AC7E2932EEcC"),
		BagsVault:         common.HexToAddress("0x4861446aa7fFd9e67a83cBbAcb1A4B70540B83Aa"),
		BagsLens:          common.HexToAddress("0xC82Db941dAf90B754aecb5F7D14c683dc608d595"),
	}
}

func DefaultAddressBook() AddressBook { return defaultAddresses }
