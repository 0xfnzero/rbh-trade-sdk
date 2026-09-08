package rbhtrade

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

const universalRouterABIJSON = `[{"type":"function","name":"execute","stateMutability":"payable","inputs":[{"name":"commands","type":"bytes"},{"name":"inputs","type":"bytes[]"},{"name":"deadline","type":"uint256"}],"outputs":[]}]`
const erc20ABIJSON = `[{"type":"function","name":"approve","stateMutability":"nonpayable","inputs":[{"name":"spender","type":"address"},{"name":"amount","type":"uint256"}],"outputs":[{"name":"","type":"bool"}]}]`
const permit2ABIJSON = `[{"type":"function","name":"approve","stateMutability":"nonpayable","inputs":[{"name":"token","type":"address"},{"name":"spender","type":"address"},{"name":"amount","type":"uint160"},{"name":"expiration","type":"uint48"}],"outputs":[]}]`
const longABIJSON = `[{"type":"function","name":"create","stateMutability":"nonpayable","inputs":[{"name":"data","type":"tuple","components":[{"name":"initialSupply","type":"uint256"},{"name":"numTokensToSell","type":"uint256"},{"name":"numeraire","type":"address"},{"name":"tokenFactory","type":"address"},{"name":"tokenFactoryData","type":"bytes"},{"name":"governanceFactory","type":"address"},{"name":"governanceFactoryData","type":"bytes"},{"name":"poolInitializer","type":"address"},{"name":"poolInitializerData","type":"bytes"},{"name":"liquidityMigrator","type":"address"},{"name":"liquidityMigratorData","type":"bytes"},{"name":"integrator","type":"address"},{"name":"salt","type":"bytes32"}]}],"outputs":[{"name":"asset","type":"address"},{"name":"poolOrHook","type":"address"}]}]`
const o1ABIJSON = `[{"type":"function","name":"createLaunch","stateMutability":"payable","inputs":[{"name":"launchParams","type":"tuple","components":[{"name":"tokenName","type":"string"},{"name":"tokenSymbol","type":"string"},{"name":"tokenContractURI","type":"string"},{"name":"creatorSalt","type":"bytes32"},{"name":"quoteToken","type":"address"},{"name":"expectedConfigVersion","type":"uint64"},{"name":"deadline","type":"uint64"},{"name":"metadataEditable","type":"bool"},{"name":"metadataKeys","type":"string[]"},{"name":"metadataValues","type":"string[]"}]}],"outputs":[{"name":"token","type":"address"},{"name":"poolId","type":"bytes32"}]},{"type":"function","name":"createLaunchAndBuy","stateMutability":"payable","inputs":[{"name":"launchParams","type":"tuple","components":[{"name":"tokenName","type":"string"},{"name":"tokenSymbol","type":"string"},{"name":"tokenContractURI","type":"string"},{"name":"creatorSalt","type":"bytes32"},{"name":"quoteToken","type":"address"},{"name":"expectedConfigVersion","type":"uint64"},{"name":"deadline","type":"uint64"},{"name":"metadataEditable","type":"bool"},{"name":"metadataKeys","type":"string[]"},{"name":"metadataValues","type":"string[]"}]},{"name":"launchBuyParams","type":"tuple","components":[{"name":"fundingToken","type":"address"},{"name":"amountIn","type":"uint256"},{"name":"minAmountOut","type":"uint256"},{"name":"routeData","type":"bytes"}]}],"outputs":[{"name":"token","type":"address"},{"name":"poolId","type":"bytes32"},{"name":"amountOut","type":"uint256"}]}]`
const bagsABIJSON = `[{"type":"function","name":"create","stateMutability":"payable","inputs":[{"name":"name","type":"string"},{"name":"symbol","type":"string"},{"name":"metadataURI","type":"string"},{"name":"partner","type":"address"},{"name":"claimers","type":"address[]"},{"name":"bps","type":"uint16[]"}],"outputs":[{"name":"token","type":"address"},{"name":"curve","type":"address"}]},{"type":"function","name":"createAndBuy","stateMutability":"payable","inputs":[{"name":"name","type":"string"},{"name":"symbol","type":"string"},{"name":"metadataURI","type":"string"},{"name":"partner","type":"address"},{"name":"claimers","type":"address[]"},{"name":"bps","type":"uint16[]"}],"outputs":[{"name":"token","type":"address"},{"name":"curve","type":"address"}]}]`
const bagsCurveABIJSON = `[{"type":"function","name":"buy","stateMutability":"payable","inputs":[{"name":"minAmountOut","type":"uint256"}],"outputs":[]},{"type":"function","name":"buyFor","stateMutability":"payable","inputs":[{"name":"recipient","type":"address"},{"name":"minAmountOut","type":"uint256"}],"outputs":[]},{"type":"function","name":"sell","stateMutability":"nonpayable","inputs":[{"name":"amountIn","type":"uint256"},{"name":"minAmountOut","type":"uint256"}],"outputs":[]},{"type":"function","name":"sellFor","stateMutability":"nonpayable","inputs":[{"name":"recipient","type":"address"},{"name":"amountIn","type":"uint256"},{"name":"minAmountOut","type":"uint256"}],"outputs":[]}]`
const pairABIJSON = `[{"type":"function","name":"launchTokenMulti","stateMutability":"payable","inputs":[{"name":"p","type":"tuple","components":[{"name":"name","type":"string"},{"name":"symbol","type":"string"},{"name":"metadataURI","type":"string"},{"name":"metadataHash","type":"bytes32"},{"name":"allocations","type":"tuple[]","components":[{"name":"quoteToken","type":"address"},{"name":"weightBps","type":"uint16"}]},{"name":"creatorFeeRecipient","type":"address"},{"name":"developerBuyRecipient","type":"address"},{"name":"developerBuyPairIndex","type":"uint8"},{"name":"developerTokenAmountOut","type":"uint256"},{"name":"maxQuoteAmountIn","type":"uint256"},{"name":"deadline","type":"uint256"}]}],"outputs":[{"name":"projectToken","type":"address"}]}]`
const poolsABIJSON = `[{"type":"function","name":"createToken","stateMutability":"payable","inputs":[{"name":"factory","type":"address"},{"name":"name","type":"string"},{"name":"symbol","type":"string"},{"name":"decimals","type":"uint8"},{"name":"initialSupply","type":"uint128"},{"name":"recipient","type":"address"},{"name":"tokenData","type":"bytes"}],"outputs":[{"name":"tokenAddress","type":"address"}]},{"type":"function","name":"distributeToken","stateMutability":"payable","inputs":[{"name":"token","type":"address"},{"name":"distribution","type":"tuple","components":[{"name":"strategy","type":"address"},{"name":"amount","type":"uint128"},{"name":"configData","type":"bytes"}]},{"name":"salt","type":"bytes32"}],"outputs":[]},{"type":"function","name":"distributeWithNative","stateMutability":"payable","inputs":[{"name":"strategy","type":"address"},{"name":"configData","type":"bytes"},{"name":"salt","type":"bytes32"},{"name":"nativeAmount","type":"uint256"}],"outputs":[]},{"type":"function","name":"multicall","stateMutability":"payable","inputs":[{"name":"data","type":"bytes[]"}],"outputs":[{"name":"results","type":"bytes[]"}]}]`
const v4QuoterABIJSON = `[{"type":"function","name":"quoteExactInputSingle","stateMutability":"nonpayable","inputs":[{"name":"params","type":"tuple","components":[{"name":"poolKey","type":"tuple","components":[{"name":"currency0","type":"address"},{"name":"currency1","type":"address"},{"name":"fee","type":"uint24"},{"name":"tickSpacing","type":"int24"},{"name":"hooks","type":"address"}]},{"name":"zeroForOne","type":"bool"},{"name":"exactAmount","type":"uint128"},{"name":"hookData","type":"bytes"}]}],"outputs":[{"name":"amountOut","type":"uint256"},{"name":"gasEstimate","type":"uint256"}]}]`
const stateViewABIJSON = `[{"type":"function","name":"getSlot0","stateMutability":"view","inputs":[{"name":"poolId","type":"bytes32"}],"outputs":[{"name":"sqrtPriceX96","type":"uint160"},{"name":"tick","type":"int24"},{"name":"protocolFee","type":"uint24"},{"name":"lpFee","type":"uint24"}]}]`
const o1HookReadABIJSON = `[{"type":"function","name":"poolConfig","stateMutability":"view","inputs":[{"name":"poolId","type":"bytes32"}],"outputs":[{"name":"initialized","type":"bool"},{"name":"tokenIsCurrency0","type":"bool"},{"name":"currentCreator","type":"address"},{"name":"creatorFeeRecipient","type":"address"},{"name":"baseFeeBps","type":"uint16"},{"name":"antiSnipeStartTotalBps","type":"uint16"},{"name":"antiSnipeWindowSeconds","type":"uint32"},{"name":"launchTime","type":"uint48"}]}]`

var (
	universalRouterABI = mustParseABI(universalRouterABIJSON)
	erc20ABI           = mustParseABI(erc20ABIJSON)
	permit2ABI         = mustParseABI(permit2ABIJSON)
	longABI            = mustParseABI(longABIJSON)
	o1ABI              = mustParseABI(o1ABIJSON)
	bagsABI            = mustParseABI(bagsABIJSON)
	bagsCurveABI       = mustParseABI(bagsCurveABIJSON)
	pairABI            = mustParseABI(pairABIJSON)
	poolsABI           = mustParseABI(poolsABIJSON)
	v4QuoterABI        = mustParseABI(v4QuoterABIJSON)
	stateViewABI       = mustParseABI(stateViewABIJSON)
	o1HookReadABI      = mustParseABI(o1HookReadABIJSON)
	v4ActionArgs       = mustV4ActionArgs()
	v4PlanArgs         = mustArguments("bytes", "bytes[]")
	addressUint256Args = mustArguments("address", "uint256")
)

func mustParseABI(raw string) abi.ABI {
	parsed, err := abi.JSON(strings.NewReader(raw))
	if err != nil {
		panic(err)
	}
	return parsed
}

func mustArguments(types ...string) abi.Arguments {
	args := make(abi.Arguments, len(types))
	for i, name := range types {
		typ, err := abi.NewType(name, "", nil)
		if err != nil {
			panic(err)
		}
		args[i] = abi.Argument{Type: typ}
	}
	return args
}

func mustV4ActionArgs() abi.Arguments {
	typ, err := abi.NewType("tuple", "ExactInputSingle", []abi.ArgumentMarshaling{
		{Name: "poolKey", Type: "tuple", Components: []abi.ArgumentMarshaling{{Name: "currency0", Type: "address"}, {Name: "currency1", Type: "address"}, {Name: "fee", Type: "uint24"}, {Name: "tickSpacing", Type: "int24"}, {Name: "hooks", Type: "address"}}},
		{Name: "zeroForOne", Type: "bool"},
		{Name: "amountIn", Type: "uint128"},
		{Name: "amountOutMinimum", Type: "uint128"},
		{Name: "minHopPriceX36", Type: "uint256"},
		{Name: "hookData", Type: "bytes"},
	})
	if err != nil {
		panic(err)
	}
	return abi.Arguments{{Type: typ}}
}
