package pons

import (
	"math/big"

	root "github.com/0xfnzero/rbh-trade-sdk/rbhtrade"
	"github.com/ethereum/go-ethereum/common"
)

func BuildLaunch(params TokenParams, launchConfigID *big.Int, pairToken common.Address, exemptions []common.Address, launchFee *big.Int) (root.Call, error) {
	if !validUint256(launchFee) {
		return root.Call{}, root.ErrInvalidAmount
	}
	data, err := PackLaunchToken(params, launchConfigID, pairToken, exemptions)
	if err != nil {
		return root.Call{}, err
	}
	return root.Call{To: root.DefaultAddressBook().PonsFactory, Data: data, Value: new(big.Int).Set(launchFee)}, nil
}

func BuildBuy(curve common.Address, quoteIn, minTokensOut *big.Int, recipient common.Address, nativeQuote bool) (root.Call, error) {
	if curve == (common.Address{}) {
		return root.Call{}, root.ErrInvalidAddress
	}
	data, err := PackBuy(quoteIn, minTokensOut, recipient)
	if err != nil {
		return root.Call{}, err
	}
	value := new(big.Int)
	if nativeQuote {
		value.Set(quoteIn)
	}
	return root.Call{To: curve, Data: data, Value: value}, nil
}

func BuildSell(curve common.Address, tokensIn, minQuoteOut *big.Int, recipient common.Address) (root.Call, error) {
	if curve == (common.Address{}) {
		return root.Call{}, root.ErrInvalidAddress
	}
	data, err := PackSell(tokensIn, minQuoteOut, recipient)
	if err != nil {
		return root.Call{}, err
	}
	return root.Call{To: curve, Data: data, Value: new(big.Int)}, nil
}

func BuildLaunchAndBuy(params TokenParams, launchConfigID *big.Int, pairToken common.Address, quoteIn, minTokensOut *big.Int, recipient common.Address, exemptions []common.Address, value *big.Int) (root.Call, error) {
	if !validUint256(value) {
		return root.Call{}, root.ErrInvalidAmount
	}
	if pairToken == (common.Address{}) && (quoteIn == nil || value.Cmp(quoteIn) < 0) {
		return root.Call{}, root.ErrInvalidAmount
	}
	data, err := PackLaunchAndBuy(params, launchConfigID, pairToken, quoteIn, minTokensOut, recipient, exemptions)
	if err != nil {
		return root.Call{}, err
	}
	return root.Call{To: root.DefaultAddressBook().PonsLaunchAndBuy, Data: data, Value: new(big.Int).Set(value)}, nil
}

func validUint256(value *big.Int) bool {
	return value != nil && value.Sign() >= 0 && value.BitLen() <= 256
}
