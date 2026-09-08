package rbhtrade

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

func BuildERC20Approve(token, spender common.Address, amount *big.Int) (Call, error) {
	if err := validateAddress(token, "token"); err != nil {
		return Call{}, err
	}
	if err := validateAddress(spender, "spender"); err != nil {
		return Call{}, err
	}
	if err := validateUint(amount, 256, false, "approval amount"); err != nil {
		return Call{}, err
	}
	data, err := erc20ABI.Pack("approve", spender, cloneBig(amount))
	if err != nil {
		return Call{}, fmt.Errorf("encode ERC-20 approval: %w", err)
	}
	return Call{To: token, Data: data, Value: new(big.Int)}, nil
}

func BuildPermit2Approve(token, spender common.Address, amount *big.Int, expiration uint64) (Call, error) {
	if err := validateAddress(token, "token"); err != nil {
		return Call{}, err
	}
	if err := validateAddress(spender, "spender"); err != nil {
		return Call{}, err
	}
	if err := validateUint(amount, 160, false, "Permit2 approval amount"); err != nil {
		return Call{}, err
	}
	if expiration > (1<<48)-1 {
		return Call{}, fmt.Errorf("Permit2 expiration exceeds uint48")
	}
	data, err := permit2ABI.Pack("approve", token, spender, cloneBig(amount), new(big.Int).SetUint64(expiration))
	if err != nil {
		return Call{}, fmt.Errorf("encode Permit2 approval: %w", err)
	}
	return Call{To: defaultAddresses.Permit2, Data: data, Value: new(big.Int)}, nil
}

func BuildUniversalRouterApprovals(token common.Address, amount *big.Int, expiration uint64) ([2]Call, error) {
	var calls [2]Call
	if err := validateUint(amount, 160, true, "approval amount"); err != nil {
		return calls, err
	}
	erc20Call, err := BuildERC20Approve(token, defaultAddresses.Permit2, amount)
	if err != nil {
		return calls, err
	}
	permit2Call, err := BuildPermit2Approve(token, defaultAddresses.UniversalRouter, amount, expiration)
	if err != nil {
		return calls, err
	}
	calls[0], calls[1] = erc20Call, permit2Call
	return calls, nil
}
