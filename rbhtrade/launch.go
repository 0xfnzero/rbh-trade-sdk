package rbhtrade

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

const PAIRNoDeveloperBuy uint8 = 255

const (
	PAIRMinMarkets = 1
	PAIRMaxMarkets = 5
	BPSTotal       = 10_000
)

func BuildLongCreate(params LongCreateParams) (Call, error) {
	if err := validateDynamicPayload("Long factory data", 4, len(params.TokenFactoryData), len(params.GovernanceFactoryData), len(params.PoolInitializerData), len(params.LiquidityMigratorData)); err != nil {
		return Call{}, err
	}
	if err := validateUint(params.InitialSupply, 256, true, "initial supply"); err != nil {
		return Call{}, err
	}
	if err := validateUint(params.NumTokensToSell, 256, true, "tokens to sell"); err != nil {
		return Call{}, err
	}
	if params.NumTokensToSell.Cmp(params.InitialSupply) > 0 {
		return Call{}, fmt.Errorf("tokens to sell exceed initial supply")
	}
	if err := validateAddress(params.Numeraire, "numeraire"); err != nil {
		return Call{}, err
	}
	if params.TokenFactory != defaultAddresses.LongTokenFactory {
		return Call{}, fmt.Errorf("long token factory %s is not trusted", params.TokenFactory)
	}
	if err := validateAddress(params.PoolInitializer, "pool initializer"); err != nil {
		return Call{}, err
	}
	if err := validateAddress(params.GovernanceFactory, "governance factory"); err != nil {
		return Call{}, err
	}
	if err := validateAddress(params.LiquidityMigrator, "liquidity migrator"); err != nil {
		return Call{}, err
	}
	data, err := longABI.Pack("create", params)
	if err != nil {
		return Call{}, fmt.Errorf("encode Long create: %w", err)
	}
	return Call{To: defaultAddresses.LongLauncher, Data: data, Value: new(big.Int)}, nil
}

func validateO1Launch(params O1LaunchParams) error {
	if err := validateMetadata(params.TokenName, params.TokenSymbol); err != nil {
		return err
	}
	if params.Deadline == 0 {
		return ErrExpiredDeadline
	}
	if len(params.MetadataKeys) != len(params.MetadataValues) {
		return ErrMismatchedLengths
	}
	if len(params.MetadataKeys) > MaxDynamicItems {
		return fmt.Errorf("o1 metadata contains too many items")
	}
	sizes := make([]int, 0, 3+len(params.MetadataKeys)+len(params.MetadataValues))
	sizes = append(sizes, len(params.TokenName), len(params.TokenSymbol), len(params.TokenContractURI))
	for _, value := range params.MetadataKeys {
		sizes = append(sizes, len(value))
	}
	for _, value := range params.MetadataValues {
		sizes = append(sizes, len(value))
	}
	if err := validateDynamicPayload("o1 metadata", len(params.MetadataKeys), sizes...); err != nil {
		return err
	}
	return nil
}

func BuildO1Launch(params O1LaunchParams, nativeLaunchFee *big.Int) (Call, error) {
	if err := validateO1Launch(params); err != nil {
		return Call{}, err
	}
	if err := validateUint(nativeLaunchFee, 256, false, "native launch fee"); err != nil {
		return Call{}, err
	}
	data, err := o1ABI.Pack("createLaunch", params)
	if err != nil {
		return Call{}, fmt.Errorf("encode o1 launch: %w", err)
	}
	return Call{To: defaultAddresses.O1Factory, Data: data, Value: cloneBig(nativeLaunchFee)}, nil
}

func BuildO1LaunchAndBuy(params O1LaunchParams, buy O1LaunchBuyParams, value *big.Int) (Call, error) {
	if err := validateO1Launch(params); err != nil {
		return Call{}, err
	}
	if err := validateUint(buy.AmountIn, 256, true, "amount in"); err != nil {
		return Call{}, err
	}
	if err := validateUint(buy.MinAmountOut, 256, false, "minimum amount out"); err != nil {
		return Call{}, err
	}
	if err := validateUint(value, 256, false, "transaction value"); err != nil {
		return Call{}, err
	}
	if err := validateDynamicPayload("o1 route data", 1, len(buy.RouteData)); err != nil {
		return Call{}, err
	}
	if buy.FundingToken == (common.Address{}) && value.Cmp(buy.AmountIn) < 0 {
		return Call{}, fmt.Errorf("transaction value is below native funding amount")
	}
	data, err := o1ABI.Pack("createLaunchAndBuy", params, buy)
	if err != nil {
		return Call{}, fmt.Errorf("encode o1 launch-and-buy: %w", err)
	}
	return Call{To: defaultAddresses.O1Factory, Data: data, Value: cloneBig(value)}, nil
}

func validateBagsLaunch(params BagsLaunchParams) error {
	if err := validateMetadata(params.Name, params.Symbol); err != nil {
		return err
	}
	if len(params.Claimers) != len(params.BPS) {
		return ErrMismatchedLengths
	}
	if err := validateDynamicPayload("Bags launch metadata", len(params.Claimers), len(params.Name), len(params.Symbol), len(params.MetadataURI)); err != nil {
		return err
	}
	total := 0
	for i, bps := range params.BPS {
		if params.Claimers[i] == (common.Address{}) {
			return fmt.Errorf("claimer %d: %w", i, ErrInvalidAddress)
		}
		total += int(bps)
	}
	if len(params.BPS) > 0 && total != BPSTotal {
		return fmt.Errorf("claimer BPS total %d: %w", total, ErrInvalidBPS)
	}
	return nil
}

func BuildBagsCreate(params BagsLaunchParams, value *big.Int) (Call, error) {
	return buildBagsCreate("create", params, value)
}

func BuildBagsCreateAndBuy(params BagsLaunchParams, value *big.Int) (Call, error) {
	return buildBagsCreate("createAndBuy", params, value)
}

func buildBagsCreate(method string, params BagsLaunchParams, value *big.Int) (Call, error) {
	if err := validateBagsLaunch(params); err != nil {
		return Call{}, err
	}
	if err := validateUint(value, 256, false, "transaction value"); err != nil {
		return Call{}, err
	}
	data, err := bagsABI.Pack(method, params.Name, params.Symbol, params.MetadataURI, params.Partner, params.Claimers, params.BPS)
	if err != nil {
		return Call{}, fmt.Errorf("encode Bags %s: %w", method, err)
	}
	return Call{To: defaultAddresses.BagsFactory, Data: data, Value: cloneBig(value)}, nil
}

func BuildBagsBuy(curve common.Address, minAmountOut, value *big.Int) (Call, error) {
	return buildBagsCurveBuy("buy", curve, common.Address{}, minAmountOut, value)
}

func BuildBagsBuyFor(curve, recipient common.Address, minAmountOut, value *big.Int) (Call, error) {
	return buildBagsCurveBuy("buyFor", curve, recipient, minAmountOut, value)
}

func buildBagsCurveBuy(method string, curve, recipient common.Address, minAmountOut, value *big.Int) (Call, error) {
	if err := validateAddress(curve, "curve"); err != nil {
		return Call{}, err
	}
	if method == "buyFor" {
		if err := validateAddress(recipient, "recipient"); err != nil {
			return Call{}, err
		}
	}
	if err := validateUint(minAmountOut, 256, false, "minimum amount out"); err != nil {
		return Call{}, err
	}
	if err := validateUint(value, 256, true, "buy value"); err != nil {
		return Call{}, err
	}
	var data []byte
	var err error
	if method == "buyFor" {
		data, err = bagsCurveABI.Pack(method, recipient, cloneBig(minAmountOut))
	} else {
		data, err = bagsCurveABI.Pack(method, cloneBig(minAmountOut))
	}
	if err != nil {
		return Call{}, fmt.Errorf("encode Bags %s: %w", method, err)
	}
	return Call{To: curve, Data: data, Value: cloneBig(value)}, nil
}

func BuildBagsSell(curve common.Address, amountIn, minAmountOut *big.Int) (Call, error) {
	return buildBagsCurveSell("sell", curve, common.Address{}, amountIn, minAmountOut)
}

func BuildBagsSellFor(curve, recipient common.Address, amountIn, minAmountOut *big.Int) (Call, error) {
	return buildBagsCurveSell("sellFor", curve, recipient, amountIn, minAmountOut)
}

func buildBagsCurveSell(method string, curve, recipient common.Address, amountIn, minAmountOut *big.Int) (Call, error) {
	if err := validateAddress(curve, "curve"); err != nil {
		return Call{}, err
	}
	if method == "sellFor" {
		if err := validateAddress(recipient, "recipient"); err != nil {
			return Call{}, err
		}
	}
	if err := validateUint(amountIn, 256, true, "amount in"); err != nil {
		return Call{}, err
	}
	if err := validateUint(minAmountOut, 256, false, "minimum amount out"); err != nil {
		return Call{}, err
	}
	var data []byte
	var err error
	if method == "sellFor" {
		data, err = bagsCurveABI.Pack(method, recipient, cloneBig(amountIn), cloneBig(minAmountOut))
	} else {
		data, err = bagsCurveABI.Pack(method, cloneBig(amountIn), cloneBig(minAmountOut))
	}
	if err != nil {
		return Call{}, fmt.Errorf("encode Bags %s: %w", method, err)
	}
	return Call{To: curve, Data: data, Value: new(big.Int)}, nil
}

func BuildPAIRLaunch(params PAIRLaunchParams, value *big.Int) (Call, error) {
	if err := validateMetadata(params.Name, params.Symbol); err != nil {
		return Call{}, err
	}
	if len(params.Allocations) < PAIRMinMarkets || len(params.Allocations) > PAIRMaxMarkets {
		return Call{}, fmt.Errorf("PAIR allocations must contain %d to %d markets", PAIRMinMarkets, PAIRMaxMarkets)
	}
	if err := validateDynamicPayload("PAIR metadata", len(params.Allocations), len(params.Name), len(params.Symbol), len(params.MetadataURI)); err != nil {
		return Call{}, err
	}
	total := 0
	for i, allocation := range params.Allocations {
		if err := validateAddress(allocation.QuoteToken, fmt.Sprintf("allocation %d quote token", i)); err != nil {
			return Call{}, err
		}
		total += int(allocation.WeightBPS)
	}
	if total != BPSTotal {
		return Call{}, fmt.Errorf("allocation BPS total %d: %w", total, ErrInvalidBPS)
	}
	if err := validateAddress(params.CreatorFeeRecipient, "creator fee recipient"); err != nil {
		return Call{}, err
	}
	if err := validateAddress(params.DeveloperBuyRecipient, "developer buy recipient"); err != nil {
		return Call{}, err
	}
	if params.DeveloperBuyPairIndex != PAIRNoDeveloperBuy && int(params.DeveloperBuyPairIndex) >= len(params.Allocations) {
		return Call{}, fmt.Errorf("developer buy pair index out of range")
	}
	if err := validateUint(params.DeveloperTokenAmountOut, 256, false, "developer token amount out"); err != nil {
		return Call{}, err
	}
	if err := validateUint(params.MaxQuoteAmountIn, 256, false, "maximum quote amount in"); err != nil {
		return Call{}, err
	}
	if params.DeveloperBuyPairIndex == PAIRNoDeveloperBuy && (params.DeveloperTokenAmountOut.Sign() != 0 || params.MaxQuoteAmountIn.Sign() != 0) {
		return Call{}, fmt.Errorf("no-developer-buy sentinel requires zero buy amounts")
	}
	if params.DeveloperBuyPairIndex != PAIRNoDeveloperBuy && (params.DeveloperTokenAmountOut.Sign() == 0 || params.MaxQuoteAmountIn.Sign() == 0) {
		return Call{}, fmt.Errorf("developer buy requires positive buy amounts")
	}
	if err := validateUint(params.Deadline, 256, true, "deadline"); err != nil {
		return Call{}, err
	}
	if err := validateUint(value, 256, false, "transaction value"); err != nil {
		return Call{}, err
	}
	data, err := pairABI.Pack("launchTokenMulti", params)
	if err != nil {
		return Call{}, fmt.Errorf("encode PAIR launch: %w", err)
	}
	return Call{To: defaultAddresses.PAIRLaunchpad, Data: data, Value: cloneBig(value)}, nil
}

func BuildPoolsCreateToken(factory common.Address, name, symbol string, decimals uint8, initialSupply *big.Int, recipient common.Address, tokenData []byte, value *big.Int) (Call, error) {
	if err := validateAddress(factory, "token factory"); err != nil {
		return Call{}, err
	}
	if err := validateMetadata(name, symbol); err != nil {
		return Call{}, err
	}
	if err := validateDynamicPayload("Pools token data", 1, len(name), len(symbol), len(tokenData)); err != nil {
		return Call{}, err
	}
	if err := validateUint(initialSupply, 128, true, "initial supply"); err != nil {
		return Call{}, err
	}
	if err := validateAddress(recipient, "recipient"); err != nil {
		return Call{}, err
	}
	if err := validateUint(value, 256, false, "transaction value"); err != nil {
		return Call{}, err
	}
	data, err := poolsABI.Pack("createToken", factory, name, symbol, decimals, cloneBig(initialSupply), recipient, tokenData)
	if err != nil {
		return Call{}, fmt.Errorf("encode Pools token creation: %w", err)
	}
	return Call{To: defaultAddresses.PoolsEntry, Data: data, Value: cloneBig(value)}, nil
}

func BuildPoolsDistributeToken(token common.Address, distribution PoolsDistribution, salt [32]byte, value *big.Int) (Call, error) {
	if err := validateAddress(token, "token"); err != nil {
		return Call{}, err
	}
	if err := validateAddress(distribution.Strategy, "strategy"); err != nil {
		return Call{}, err
	}
	if err := validateUint(distribution.Amount, 128, true, "distribution amount"); err != nil {
		return Call{}, err
	}
	if err := validateUint(value, 256, false, "transaction value"); err != nil {
		return Call{}, err
	}
	data, err := poolsABI.Pack("distributeToken", token, distribution, salt)
	if err != nil {
		return Call{}, fmt.Errorf("encode Pools distribution: %w", err)
	}
	return Call{To: defaultAddresses.PoolsEntry, Data: data, Value: cloneBig(value)}, nil
}

func BuildPoolsDistributeWithNative(strategy common.Address, configData []byte, salt [32]byte, nativeAmount, value *big.Int) (Call, error) {
	if err := validateAddress(strategy, "strategy"); err != nil {
		return Call{}, err
	}
	if err := validateDynamicPayload("Pools configuration", 1, len(configData)); err != nil {
		return Call{}, err
	}
	if err := validateUint(nativeAmount, 256, true, "native distribution amount"); err != nil {
		return Call{}, err
	}
	if err := validateUint(value, 256, true, "transaction value"); err != nil {
		return Call{}, err
	}
	if value.Cmp(nativeAmount) < 0 {
		return Call{}, fmt.Errorf("transaction value is below native distribution amount")
	}
	data, err := poolsABI.Pack("distributeWithNative", strategy, configData, salt, cloneBig(nativeAmount))
	if err != nil {
		return Call{}, fmt.Errorf("encode Pools native distribution: %w", err)
	}
	return Call{To: defaultAddresses.PoolsEntry, Data: data, Value: cloneBig(value)}, nil
}

func BuildPoolsMulticall(calls [][]byte, value *big.Int) (Call, error) {
	if len(calls) == 0 {
		return Call{}, fmt.Errorf("multicall requires at least one call")
	}
	if len(calls) > MaxDynamicItems {
		return Call{}, fmt.Errorf("pools multicall contains too many items")
	}
	sizes := make([]int, 0, len(calls))
	for i, call := range calls {
		if len(call) == 0 {
			return Call{}, fmt.Errorf("multicall element %d is empty", i)
		}
		sizes = append(sizes, len(call))
	}
	if err := validateDynamicPayload("Pools multicall", len(calls), sizes...); err != nil {
		return Call{}, err
	}
	if err := validateUint(value, 256, false, "transaction value"); err != nil {
		return Call{}, err
	}
	data, err := poolsABI.Pack("multicall", calls)
	if err != nil {
		return Call{}, fmt.Errorf("encode Pools multicall: %w", err)
	}
	return Call{To: defaultAddresses.PoolsEntry, Data: data, Value: cloneBig(value)}, nil
}
