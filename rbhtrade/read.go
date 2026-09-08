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
	if !ok0 || !ok1 {
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
	if !ok0 || !ok1 || !ok2 || !ok3 || !tick.IsInt64() || tick.Int64() < -8388608 || tick.Int64() > 8388607 || protocolFee.Sign() < 0 || protocolFee.BitLen() > 24 || lpFee.Sign() < 0 || lpFee.BitLen() > 24 {
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
	if !ok0 || !ok1 || !ok2 || !ok3 || !ok4 || !ok5 || !ok6 || !ok7 || !launchTime.IsUint64() || launchTime.BitLen() > 48 {
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
