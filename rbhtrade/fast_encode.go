package rbhtrade

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

const abiWordSize = 32

func encodeV4ExactInputSingle(req ExactInputRequest, zeroForOne bool, minHopPrice *big.Int) ([]byte, error) {
	hookPadded, err := paddedABIBytes(len(req.HookData))
	if err != nil {
		return nil, err
	}
	swapLength := 12*abiWordSize + hookPadded
	swap := make([]byte, swapLength)
	putUint64Word(swap, 0, abiWordSize)
	tuple := abiWordSize
	putAddressWord(swap, tuple, req.PoolKey.Currency0)
	putAddressWord(swap, tuple+abiWordSize, req.PoolKey.Currency1)
	putUint64Word(swap, tuple+2*abiWordSize, uint64(req.PoolKey.Fee))
	putUint64Word(swap, tuple+3*abiWordSize, uint64(req.PoolKey.TickSpacing))
	putAddressWord(swap, tuple+4*abiWordSize, req.PoolKey.Hooks)
	if zeroForOne {
		putUint64Word(swap, tuple+5*abiWordSize, 1)
	}
	putBigWord(swap, tuple+6*abiWordSize, req.AmountIn)
	putBigWord(swap, tuple+7*abiWordSize, req.AmountOutMinimum)
	putBigWord(swap, tuple+8*abiWordSize, minHopPrice)
	putUint64Word(swap, tuple+9*abiWordSize, 10*abiWordSize)
	putUint64Word(swap, tuple+10*abiWordSize, uint64(len(req.HookData)))
	copy(swap[tuple+11*abiWordSize:], req.HookData)

	const (
		planHeadLength   = 2 * abiWordSize
		actionsLength    = 2 * abiWordSize
		paramsArrayStart = planHeadLength + actionsLength
		paramsTableStart = paramsArrayStart + abiWordSize
		paramsHeadLength = 3 * abiWordSize
	)
	firstStart := paramsTableStart + paramsHeadLength
	secondStart := firstStart + abiWordSize + swapLength
	thirdStart := secondStart + 3*abiWordSize
	planLength := thirdStart + 3*abiWordSize
	plan := make([]byte, planLength)
	putUint64Word(plan, 0, planHeadLength)
	putUint64Word(plan, abiWordSize, paramsArrayStart)
	putUint64Word(plan, planHeadLength, 3)
	plan[planHeadLength+abiWordSize] = ActionSwapExactInSingle
	plan[planHeadLength+abiWordSize+1] = ActionSettleAll
	plan[planHeadLength+abiWordSize+2] = ActionTakeAll
	putUint64Word(plan, paramsArrayStart, 3)
	putUint64Word(plan, paramsTableStart, uint64(firstStart-paramsTableStart))
	putUint64Word(plan, paramsTableStart+abiWordSize, uint64(secondStart-paramsTableStart))
	putUint64Word(plan, paramsTableStart+2*abiWordSize, uint64(thirdStart-paramsTableStart))
	putUint64Word(plan, firstStart, uint64(swapLength))
	copy(plan[firstStart+abiWordSize:], swap)
	putUint64Word(plan, secondStart, 2*abiWordSize)
	putAddressWord(plan, secondStart+abiWordSize, req.CurrencyIn)
	putBigWord(plan, secondStart+2*abiWordSize, maxUint256)
	putUint64Word(plan, thirdStart, 2*abiWordSize)
	putAddressWord(plan, thirdStart+abiWordSize, req.CurrencyOut)
	putBigWord(plan, thirdStart+2*abiWordSize, req.AmountOutMinimum)

	const (
		outerHeadLength   = 3 * abiWordSize
		commandsStart     = outerHeadLength
		inputsStart       = commandsStart + 2*abiWordSize
		inputTableStart   = inputsStart + abiWordSize
		firstInputStart   = inputTableStart + abiWordSize
		firstInputPayload = firstInputStart + abiWordSize
	)
	data := make([]byte, 4+firstInputPayload+planLength)
	copy(data[:4], universalRouterABI.Methods["execute"].ID)
	args := data[4:]
	putUint64Word(args, 0, commandsStart)
	putUint64Word(args, abiWordSize, inputsStart)
	putUint64Word(args, 2*abiWordSize, req.Deadline)
	putUint64Word(args, commandsStart, 1)
	args[commandsStart+abiWordSize] = CommandV4Swap
	putUint64Word(args, inputsStart, 1)
	putUint64Word(args, inputTableStart, abiWordSize)
	putUint64Word(args, firstInputStart, uint64(planLength))
	copy(args[firstInputPayload:], plan)
	return data, nil
}

func paddedABIBytes(length int) (int, error) {
	if length < 0 || length > int(^uint(0)>>1)-(abiWordSize-1) {
		return 0, fmt.Errorf("hook data is too large")
	}
	return (length + abiWordSize - 1) &^ (abiWordSize - 1), nil
}

func putAddressWord(dst []byte, offset int, address common.Address) {
	copy(dst[offset+12:offset+abiWordSize], address[:])
}

func putUint64Word(dst []byte, offset int, value uint64) {
	for i := 0; i < 8; i++ {
		dst[offset+abiWordSize-1-i] = byte(value)
		value >>= 8
	}
}

func putBigWord(dst []byte, offset int, value *big.Int) {
	value.FillBytes(dst[offset : offset+abiWordSize])
}
