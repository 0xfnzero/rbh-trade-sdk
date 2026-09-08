package pons

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

func PackLaunchToken(params TokenParams, launchConfigID *big.Int, pairToken common.Address, snipeTaxExemptions []common.Address) ([]byte, error) {
	if err := validateLaunch(params, snipeTaxExemptions); err != nil {
		return nil, err
	}
	if err := validateUint256(launchConfigID, "launch config ID"); err != nil {
		return nil, err
	}
	return factoryLaunchContractABI.Pack("launchToken", params, launchConfigID, pairToken, snipeTaxExemptions)
}

func PackLaunchTokenSimple(params TokenParams, launchConfigID *big.Int, pairToken common.Address) ([]byte, error) {
	if err := validateLaunch(params, nil); err != nil {
		return nil, err
	}
	if err := validateUint256(launchConfigID, "launch config ID"); err != nil {
		return nil, err
	}
	return factorySimpleContractABI.Pack("launchToken", params, launchConfigID, pairToken)
}

func PackLaunchTokenFor(params TokenParams, launchConfigID *big.Int, pairToken common.Address, originalDeployer common.Address, snipeTaxExemptions []common.Address) ([]byte, error) {
	if err := validateLaunch(params, snipeTaxExemptions); err != nil {
		return nil, err
	}
	if err := validateUint256(launchConfigID, "launch config ID"); err != nil {
		return nil, err
	}
	if err := validateAddress(originalDeployer, "original deployer"); err != nil {
		return nil, err
	}
	return factoryWriteContractABI.Pack("launchTokenFor", params, launchConfigID, pairToken, originalDeployer, snipeTaxExemptions)
}

func PackLaunchAndBuy(params TokenParams, launchConfigID *big.Int, pairToken common.Address, quoteIn *big.Int, minTokensOut *big.Int, recipient common.Address, snipeTaxExemptions []common.Address) ([]byte, error) {
	if err := validateLaunch(params, snipeTaxExemptions); err != nil {
		return nil, err
	}
	if err := validateUint256(launchConfigID, "launch config ID"); err != nil {
		return nil, err
	}
	if err := validatePositiveUint256(quoteIn, "quote input"); err != nil {
		return nil, err
	}
	if err := validateUint256(minTokensOut, "minimum token output"); err != nil {
		return nil, err
	}
	if err := validateAddress(recipient, "recipient"); err != nil {
		return nil, err
	}
	return routerContractABI.Pack("launchAndBuy", params, launchConfigID, pairToken, quoteIn, minTokensOut, recipient, snipeTaxExemptions)
}

func PackBuy(quoteIn *big.Int, minTokensOut *big.Int, recipient common.Address) ([]byte, error) {
	if err := validatePositiveUint256(quoteIn, "quote input"); err != nil {
		return nil, err
	}
	if err := validateUint256(minTokensOut, "minimum token output"); err != nil {
		return nil, err
	}
	if err := validateAddress(recipient, "recipient"); err != nil {
		return nil, err
	}
	return curveContractABI.Pack("buy", quoteIn, minTokensOut, recipient)
}

func PackSell(tokensIn *big.Int, minQuoteOut *big.Int, recipient common.Address) ([]byte, error) {
	if err := validatePositiveUint256(tokensIn, "token input"); err != nil {
		return nil, err
	}
	if err := validateUint256(minQuoteOut, "minimum quote output"); err != nil {
		return nil, err
	}
	if err := validateAddress(recipient, "recipient"); err != nil {
		return nil, err
	}
	return curveContractABI.Pack("sell", tokensIn, minQuoteOut, recipient)
}

func PackSweepCurveFees(minBuybackTokensOut *big.Int) ([]byte, error) {
	if err := validateUint256(minBuybackTokensOut, "minimum buyback token output"); err != nil {
		return nil, err
	}
	return curveContractABI.Pack("sweepFees", minBuybackTokensOut)
}

func PackGraduate(token common.Address) ([]byte, error) {
	if err := validateAddress(token, "token"); err != nil {
		return nil, err
	}
	return factoryWriteContractABI.Pack("graduate", token)
}

func PackCreateGraduatedPool(token common.Address) ([]byte, error) {
	if err := validateAddress(token, "token"); err != nil {
		return nil, err
	}
	return factoryWriteContractABI.Pack("createGraduatedPool", token)
}

func PackSweepPoolFees(poolID common.Hash, minConversionQuoteOut *big.Int, minBuybackTokensOut *big.Int) ([]byte, error) {
	if err := validateUint256(minConversionQuoteOut, "minimum conversion quote output"); err != nil {
		return nil, err
	}
	if err := validateUint256(minBuybackTokensOut, "minimum buyback token output"); err != nil {
		return nil, err
	}
	return hookContractABI.Pack("sweepPoolFees", [32]byte(poolID), minConversionQuoteOut, minBuybackTokensOut)
}

func PackTransferCreatorFeeRecipient(token common.Address, newRecipient common.Address) ([]byte, error) {
	if err := validateAddress(token, "token"); err != nil {
		return nil, err
	}
	if err := validateAddress(newRecipient, "new recipient"); err != nil {
		return nil, err
	}
	return factoryWriteContractABI.Pack("transferCreatorFeeRecipient", token, newRecipient)
}

func PackSetBuybackEnabled(token common.Address, enabled bool) ([]byte, error) {
	if err := validateAddress(token, "token"); err != nil {
		return nil, err
	}
	return factoryWriteContractABI.Pack("setBuybackEnabled", token, enabled)
}

func PackApprove(spender common.Address, amount *big.Int) ([]byte, error) {
	if err := validateAddress(spender, "spender"); err != nil {
		return nil, err
	}
	if err := validateUint256(amount, "approval amount"); err != nil {
		return nil, err
	}
	return tokenContractABI.Pack("approve", spender, amount)
}

func PackClaimNative() ([]byte, error) {
	return escrowContractABI.Pack("claim")
}

func PackClaimToken(token common.Address) ([]byte, error) {
	if err := validateAddress(token, "token"); err != nil {
		return nil, err
	}
	return escrowContractABI.Pack("claimToken", token)
}

func PackClaimNativeAmount(amount *big.Int) ([]byte, error) {
	if err := validatePositiveUint256(amount, "claim amount"); err != nil {
		return nil, err
	}
	return escrowAmountContractABI.Pack("claim", amount)
}

func PackClaimTokenAmount(token common.Address, amount *big.Int) ([]byte, error) {
	if err := validateAddress(token, "token"); err != nil {
		return nil, err
	}
	if err := validatePositiveUint256(amount, "claim amount"); err != nil {
		return nil, err
	}
	return escrowAmountContractABI.Pack("claimToken", token, amount)
}

func PackReleaseVested(token common.Address) ([]byte, error) {
	if err := validateAddress(token, "token"); err != nil {
		return nil, err
	}
	return vaultContractABI.Pack("release", token)
}
