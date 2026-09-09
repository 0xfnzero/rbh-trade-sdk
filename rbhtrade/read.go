package rbhtrade

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type quoteExactSingleABI struct {
	PoolKey     poolKeyABI `abi:"poolKey"`
	ZeroForOne  bool       `abi:"zeroForOne"`
	ExactAmount *big.Int   `abi:"exactAmount"`
	HookData    []byte     `abi:"hookData"`
}

type O1HookPoolConfig struct {
	Initialized         bool
	TokenIsCurrency0    bool
	CurrentCreator      common.Address
	CreatorFeeRecipient common.Address
	FeeSchedule         O1HookFeeSchedule
}

type bagsTokenStateABI struct {
	Exists               bool
	Migrated             bool
	Curve                common.Address
	FeeShare             common.Address
	PoolId               common.Hash
	ThresholdQuote       *big.Int
	RealQuoteReserves    *big.Int
	RealTokenReserves    *big.Int
	VirtualTokenReserves *big.Int
	VirtualQuoteReserves *big.Int
	PriceQuotePerToken   *big.Int
	BondingProgressPct   *big.Int
	TotalRaised          *big.Int
}

func BuildV4QuoteExactInputSingle(req QuoteExactInputRequest) (Call, error) {
	if len(req.HookData) > MaxHookDataBytes {
		return Call{}, fmt.Errorf("hook data exceeds %d bytes", MaxHookDataBytes)
	}
	if err := validatePoolKey(req.PoolKey); err != nil {
		return Call{}, err
	}
	if err := validateUint(req.AmountIn, 128, true, "quote amount in"); err != nil {
		return Call{}, err
	}
	zeroForOne := false
	switch req.CurrencyIn {
	case req.PoolKey.Currency0:
		zeroForOne = true
	case req.PoolKey.Currency1:
	default:
		return Call{}, ErrCurrencyNotInPool
	}
	data, err := v4QuoterABI.Pack("quoteExactInputSingle", quoteExactSingleABI{PoolKey: toPoolKeyABI(req.PoolKey), ZeroForOne: zeroForOne, ExactAmount: cloneBig(req.AmountIn), HookData: append([]byte(nil), req.HookData...)})
	if err != nil {
		return Call{}, fmt.Errorf("encode v4 quote: %w", err)
	}
	return Call{To: defaultAddresses.V4Quoter, Data: data, Value: new(big.Int)}, nil
}

func DecodeV4QuoteExactInputSingle(data []byte) (QuoteResult, error) {
	if len(data) != 64 {
		return QuoteResult{}, fmt.Errorf("quote response length %d", len(data))
	}
	values, err := v4QuoterABI.Methods["quoteExactInputSingle"].Outputs.Unpack(data)
	if err != nil {
		return QuoteResult{}, fmt.Errorf("decode v4 quote response: %w", err)
	}
	if len(values) != 2 {
		return QuoteResult{}, fmt.Errorf("unexpected v4 quote response field count %d", len(values))
	}
	amountOut, ok0 := values[0].(*big.Int)
	gasEstimate, ok1 := values[1].(*big.Int)
	if !ok0 || !ok1 || amountOut == nil || gasEstimate == nil {
		return QuoteResult{}, fmt.Errorf("unexpected v4 quote response types")
	}
	return QuoteResult{AmountOut: new(big.Int).Set(amountOut), GasEstimate: new(big.Int).Set(gasEstimate)}, nil
}

func BuildStateViewGetSlot0(poolID common.Hash) (Call, error) {
	if poolID == (common.Hash{}) {
		return Call{}, fmt.Errorf("zero pool id")
	}
	data, err := stateViewABI.Pack("getSlot0", poolID)
	if err != nil {
		return Call{}, fmt.Errorf("encode StateView getSlot0: %w", err)
	}
	return Call{To: defaultAddresses.StateView, Data: data, Value: new(big.Int)}, nil
}

func DecodeStateViewSlot0(data []byte) (Slot0, error) {
	if len(data) != 128 {
		return Slot0{}, fmt.Errorf("slot0 response length %d", len(data))
	}
	values, err := stateViewABI.Methods["getSlot0"].Outputs.Unpack(data)
	if err != nil {
		return Slot0{}, fmt.Errorf("decode StateView slot0: %w", err)
	}
	if len(values) != 4 {
		return Slot0{}, fmt.Errorf("unexpected StateView slot0 field count %d", len(values))
	}
	sqrtPrice, ok0 := values[0].(*big.Int)
	tick, ok1 := values[1].(*big.Int)
	protocolFee, ok2 := values[2].(*big.Int)
	lpFee, ok3 := values[3].(*big.Int)
	if !ok0 || !ok1 || !ok2 || !ok3 || sqrtPrice == nil || tick == nil || protocolFee == nil || lpFee == nil || sqrtPrice.Sign() < 0 || sqrtPrice.BitLen() > 160 || !tick.IsInt64() || tick.Int64() < -8388608 || tick.Int64() > 8388607 || protocolFee.Sign() < 0 || protocolFee.BitLen() > 24 || lpFee.Sign() < 0 || lpFee.BitLen() > 24 {
		return Slot0{}, fmt.Errorf("unexpected StateView slot0 response types")
	}
	// #nosec G115 -- ABI values are range-checked as int24/uint24 immediately above.
	return Slot0{SqrtPriceX96: new(big.Int).Set(sqrtPrice), Tick: int32(tick.Int64()), ProtocolFee: uint32(protocolFee.Uint64()), LPFee: uint32(lpFee.Uint64())}, nil
}

func BuildO1HookPoolConfig(poolID common.Hash) (Call, error) {
	if poolID == (common.Hash{}) {
		return Call{}, fmt.Errorf("zero pool id")
	}
	data, err := o1HookReadABI.Pack("poolConfig", poolID)
	if err != nil {
		return Call{}, fmt.Errorf("encode o1 poolConfig: %w", err)
	}
	return Call{To: defaultAddresses.O1Hook, Data: data, Value: new(big.Int)}, nil
}

func DecodeO1HookPoolConfig(data []byte) (O1HookPoolConfig, error) {
	if len(data) != 256 {
		return O1HookPoolConfig{}, fmt.Errorf("o1 poolConfig response length %d", len(data))
	}
	for _, wordIndex := range [...]int{2, 3} {
		for _, b := range data[wordIndex*32 : wordIndex*32+12] {
			if b != 0 {
				return O1HookPoolConfig{}, ErrInvalidO1HookState
			}
		}
	}
	values, err := o1HookReadABI.Methods["poolConfig"].Outputs.Unpack(data)
	if err != nil {
		return O1HookPoolConfig{}, fmt.Errorf("decode o1 poolConfig response: %w", err)
	}
	if len(values) != 8 {
		return O1HookPoolConfig{}, fmt.Errorf("unexpected o1 poolConfig field count %d", len(values))
	}
	initialized, ok0 := values[0].(bool)
	tokenIsCurrency0, ok1 := values[1].(bool)
	currentCreator, ok2 := values[2].(common.Address)
	creatorFeeRecipient, ok3 := values[3].(common.Address)
	baseFee, ok4 := values[4].(uint16)
	antiSnipeStart, ok5 := values[5].(uint16)
	antiSnipeWindow, ok6 := values[6].(uint32)
	launchTime, ok7 := values[7].(*big.Int)
	if !ok0 || !ok1 || !ok2 || !ok3 || !ok4 || !ok5 || !ok6 || !ok7 || launchTime == nil || !launchTime.IsUint64() || launchTime.BitLen() > 48 {
		return O1HookPoolConfig{}, fmt.Errorf("unexpected o1 poolConfig response types")
	}
	config := O1HookPoolConfig{
		Initialized: initialized, TokenIsCurrency0: tokenIsCurrency0,
		CurrentCreator: currentCreator, CreatorFeeRecipient: creatorFeeRecipient,
		FeeSchedule: O1HookFeeSchedule{BaseFeeBPS: baseFee, AntiSnipeStartTotalBPS: antiSnipeStart, AntiSnipeWindowSeconds: antiSnipeWindow, LaunchTime: launchTime.Uint64()},
	}
	if !config.Initialized || config.CurrentCreator == (common.Address{}) || config.CreatorFeeRecipient == (common.Address{}) {
		return O1HookPoolConfig{}, ErrInvalidO1HookState
	}
	if _, err := O1HookFeeBPS(config.FeeSchedule, config.FeeSchedule.LaunchTime); err != nil {
		return O1HookPoolConfig{}, err
	}
	return config, nil
}

func BuildBagsGetTokenState(token common.Address) (Call, error) {
	if err := validateAddress(token, "Bags token"); err != nil {
		return Call{}, err
	}
	data, err := bagsLensABI.Pack("getTokenState", token)
	if err != nil {
		return Call{}, fmt.Errorf("encode Bags getTokenState: %w", err)
	}
	return Call{To: defaultAddresses.BagsLens, Data: data, Value: new(big.Int)}, nil
}

func DecodeBagsTokenState(data []byte) (BagsTokenState, error) {
	const encodedBagsTokenStateBytes = 13 * 32
	if len(data) != encodedBagsTokenStateBytes {
		return BagsTokenState{}, fmt.Errorf("bags token state response length %d", len(data))
	}
	for _, word := range [...]int{2, 3} {
		for _, padding := range data[word*32 : word*32+12] {
			if padding != 0 {
				return BagsTokenState{}, fmt.Errorf("bags token state address field %d: %w", word, ErrInvalidBagsState)
			}
		}
	}
	values, err := bagsLensABI.Methods["getTokenState"].Outputs.Unpack(data)
	if err != nil {
		return BagsTokenState{}, fmt.Errorf("decode Bags token state response: %w", err)
	}
	var decoded struct {
		State bagsTokenStateABI
	}
	if err := bagsLensABI.Methods["getTokenState"].Outputs.Copy(&decoded, values); err != nil {
		return BagsTokenState{}, fmt.Errorf("copy Bags token state response: %w", err)
	}
	state := BagsTokenState{
		Exists:               decoded.State.Exists,
		Migrated:             decoded.State.Migrated,
		Curve:                decoded.State.Curve,
		FeeShare:             decoded.State.FeeShare,
		PoolID:               decoded.State.PoolId,
		ThresholdQuote:       decoded.State.ThresholdQuote,
		RealQuoteReserves:    decoded.State.RealQuoteReserves,
		RealTokenReserves:    decoded.State.RealTokenReserves,
		VirtualTokenReserves: decoded.State.VirtualTokenReserves,
		VirtualQuoteReserves: decoded.State.VirtualQuoteReserves,
		PriceQuotePerToken:   decoded.State.PriceQuotePerToken,
		BondingProgressPct:   decoded.State.BondingProgressPct,
		TotalRaised:          decoded.State.TotalRaised,
	}
	amounts := [...]*big.Int{
		state.ThresholdQuote,
		state.RealQuoteReserves,
		state.RealTokenReserves,
		state.VirtualTokenReserves,
		state.VirtualQuoteReserves,
		state.PriceQuotePerToken,
		state.BondingProgressPct,
		state.TotalRaised,
	}
	for i, amount := range amounts {
		if amount == nil || amount.Sign() < 0 || amount.BitLen() > 256 {
			return BagsTokenState{}, fmt.Errorf("bags token state field %d: %w", i, ErrInvalidBagsState)
		}
	}
	if state.Migrated && !state.Exists {
		return BagsTokenState{}, ErrInvalidBagsState
	}
	if state.Exists && (state.Curve == (common.Address{}) || state.PoolID == (common.Hash{})) {
		return BagsTokenState{}, ErrInvalidBagsState
	}
	state.ThresholdQuote = cloneBig(state.ThresholdQuote)
	state.RealQuoteReserves = cloneBig(state.RealQuoteReserves)
	state.RealTokenReserves = cloneBig(state.RealTokenReserves)
	state.VirtualTokenReserves = cloneBig(state.VirtualTokenReserves)
	state.VirtualQuoteReserves = cloneBig(state.VirtualQuoteReserves)
	state.PriceQuotePerToken = cloneBig(state.PriceQuotePerToken)
	state.BondingProgressPct = cloneBig(state.BondingProgressPct)
	state.TotalRaised = cloneBig(state.TotalRaised)
	return state, nil
}

func BuildBagsClaimableOf(token, user common.Address) (Call, error) {
	if err := validateAddress(token, "Bags token"); err != nil {
		return Call{}, err
	}
	if err := validateAddress(user, "Bags claimer"); err != nil {
		return Call{}, err
	}
	data, err := bagsLensABI.Pack("claimableOf", token, user)
	if err != nil {
		return Call{}, fmt.Errorf("encode Bags claimableOf: %w", err)
	}
	return Call{To: defaultAddresses.BagsLens, Data: data, Value: new(big.Int)}, nil
}

func DecodeBagsClaimableOf(data []byte) (*big.Int, error) {
	if len(data) != 32 {
		return nil, fmt.Errorf("bags claimableOf response length %d", len(data))
	}
	values, err := bagsLensABI.Methods["claimableOf"].Outputs.Unpack(data)
	if err != nil {
		return nil, fmt.Errorf("decode Bags claimableOf response: %w", err)
	}
	if len(values) != 1 {
		return nil, fmt.Errorf("unexpected Bags claimableOf response field count %d", len(values))
	}
	amount, ok := values[0].(*big.Int)
	if !ok || amount == nil || amount.Sign() < 0 || amount.BitLen() > 256 {
		return nil, fmt.Errorf("unexpected Bags claimableOf response type")
	}
	return cloneBig(amount), nil
}

func BuildBagsQuoteBuy(curve common.Address, quoteIn *big.Int) (Call, error) {
	return buildBagsCurveQuote("quoteBuy", curve, quoteIn)
}

func BuildBagsQuoteSell(curve common.Address, tokensIn *big.Int) (Call, error) {
	return buildBagsCurveQuote("quoteSell", curve, tokensIn)
}

func buildBagsCurveQuote(method string, curve common.Address, amountIn *big.Int) (Call, error) {
	if err := validateAddress(curve, "Bags curve"); err != nil {
		return Call{}, err
	}
	if err := validateUint(amountIn, 256, true, "Bags quote amount in"); err != nil {
		return Call{}, err
	}
	data, err := bagsCurveABI.Pack(method, cloneBig(amountIn))
	if err != nil {
		return Call{}, fmt.Errorf("encode Bags %s: %w", method, err)
	}
	return Call{To: curve, Data: data, Value: new(big.Int)}, nil
}

func DecodeBagsQuoteBuy(data []byte) (BagsBuyQuote, error) {
	values, err := decodeBagsQuote("quoteBuy", data, 5)
	if err != nil {
		return BagsBuyQuote{}, err
	}
	if new(big.Int).Add(values[1], values[2]).Cmp(values[3]) != 0 {
		return BagsBuyQuote{}, ErrInvalidBagsQuote
	}
	return BagsBuyQuote{TokensOut: values[0], FeeQuote: values[1], NetQuoteIn: values[2], GrossUsed: values[3], RefundQuote: values[4]}, nil
}

func DecodeBagsQuoteSell(data []byte) (BagsSellQuote, error) {
	values, err := decodeBagsQuote("quoteSell", data, 3)
	if err != nil {
		return BagsSellQuote{}, err
	}
	if new(big.Int).Add(values[0], values[1]).Cmp(values[2]) != 0 {
		return BagsSellQuote{}, ErrInvalidBagsQuote
	}
	return BagsSellQuote{QuoteToSeller: values[0], FeeQuote: values[1], GrossQuoteOut: values[2]}, nil
}

func decodeBagsQuote(method string, data []byte, words int) ([]*big.Int, error) {
	if len(data) != words*32 {
		return nil, fmt.Errorf("bags %s response length %d", method, len(data))
	}
	decoded, err := bagsCurveABI.Methods[method].Outputs.Unpack(data)
	if err != nil {
		return nil, fmt.Errorf("decode Bags %s response: %w", method, err)
	}
	if len(decoded) != words {
		return nil, fmt.Errorf("unexpected Bags %s response field count %d", method, len(decoded))
	}
	values := make([]*big.Int, words)
	for i, value := range decoded {
		amount, ok := value.(*big.Int)
		if !ok || amount == nil || amount.Sign() < 0 || amount.BitLen() > 256 {
			return nil, fmt.Errorf("unexpected Bags %s response field %d", method, i)
		}
		values[i] = cloneBig(amount)
	}
	return values, nil
}
