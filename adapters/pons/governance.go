package pons

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	factoryLaunchEconomicsEvents = eventIDs(FactoryABI,
		"LaunchConfigAdded",
		"LaunchConfigUpdated",
		"LaunchFeeUpdated",
		"LaunchEnabledUpdated",
		"WhitelistedLauncherUpdated",
		"MaxCreatorTaxUpdated",
		"SnipeTaxStartBpsUpdated",
		"SnipeTaxSecondsUpdated",
		"GraduationExecutorSet",
		"LaunchDeployerSet",
		"LaunchForwarderSet",
		"PairTokenApprovalUpdated",
		"PairTokenEconomicsUpdated",
	)
	hookLaunchEconomicsEvents = eventIDs(HookABI,
		"FactorySet",
		"BuybackVaultSet",
		"ProtocolFeeShareUpdated",
		"BuybackBurnBpsUpdated",
		"HookFeeBpsUpdated",
		"MaxInternalPriceImpactUpdated",
		"ProtocolFeeRecipientUpdated",
		"FeeSweepOperatorUpdated",
	)
	factoryLaunchEconomicsCalls = methodIDs(FactoryWriteABI,
		"addLaunchConfig", "updateLaunchConfig", "setLaunchFee", "setLaunchEnabled",
		"setWhitelistedLauncher", "setPairTokenEconomics", "setPairTokenApproved",
		"setMaxCreatorTaxBps", "setSnipeTaxStartBps", "setSnipeTaxSeconds",
		"setGraduationExecutor", "setLaunchDeployer", "setLaunchForwarder",
	)
	hookLaunchEconomicsCalls = signatureIDs(
		"setFactory(address)", "setBuybackVault(address)", "setProtocolFeeShareBps(uint256)",
		"setBuybackBurnBps(uint256)", "setHookFeeBps(uint256)", "setMaxInternalPriceImpactBps(uint256)",
		"setProtocolFeeRecipient(address)", "setFeeSweepOperator(address)",
	)
)

var ErrInvalidTokenLaunchedLog = errors.New("invalid Pons TokenLaunched log")

// IsLaunchEconomicsEvent reports whether a Pons event invalidates the
// governance snapshot used for deterministic launch prediction and quoting.
// Operational events such as launches, swaps, fee collection, and sweeps do
// not invalidate that snapshot.
func IsLaunchEconomicsEvent(address common.Address, topic common.Hash) bool {
	var events map[common.Hash]struct{}
	switch address {
	case PonsV2Addresses.Factory:
		events = factoryLaunchEconomicsEvents
	case PonsV2Addresses.MemeHook:
		events = hookLaunchEconomicsEvents
	default:
		return false
	}
	_, ok := events[topic]
	return ok
}

// IsLaunchEconomicsCall reports whether calldata invokes a mutator that can
// change deterministic launch prediction or initial curve economics.
func IsLaunchEconomicsCall(address common.Address, data []byte) bool {
	if len(data) < 4 {
		return false
	}
	var methods map[[4]byte]struct{}
	switch address {
	case PonsV2Addresses.Factory:
		methods = factoryLaunchEconomicsCalls
	case PonsV2Addresses.MemeHook:
		methods = hookLaunchEconomicsCalls
	default:
		return false
	}
	var selector [4]byte
	copy(selector[:], data[:4])
	_, ok := methods[selector]
	return ok
}

func TokenLaunchedEventID() common.Hash { return FactoryABI.Events["TokenLaunched"].ID }

// TokenLaunchedPairToken extracts the non-indexed quote token from an official
// factory TokenLaunched log. It is intended for cold-path pair discovery.
func TokenLaunchedPairToken(eventLog types.Log) (common.Address, error) {
	event := FactoryABI.Events["TokenLaunched"]
	if eventLog.Address != PonsV2Addresses.Factory || len(eventLog.Topics) != 4 || eventLog.Topics[0] != event.ID {
		return common.Address{}, ErrInvalidTokenLaunchedLog
	}
	values, err := event.Inputs.NonIndexed().Unpack(eventLog.Data)
	if err != nil || len(values) != 3 {
		return common.Address{}, ErrInvalidTokenLaunchedLog
	}
	pairToken, ok := values[0].(common.Address)
	if !ok {
		return common.Address{}, ErrInvalidTokenLaunchedLog
	}
	return pairToken, nil
}

func eventIDs(contractABI abi.ABI, names ...string) map[common.Hash]struct{} {
	events := make(map[common.Hash]struct{}, len(names))
	for _, name := range names {
		event, ok := contractABI.Events[name]
		if !ok {
			panic(fmt.Sprintf("Pons ABI event %q is missing", name))
		}
		events[event.ID] = struct{}{}
	}
	return events
}

func methodIDs(contractABI abi.ABI, names ...string) map[[4]byte]struct{} {
	methods := make(map[[4]byte]struct{}, len(names))
	for _, name := range names {
		method, ok := contractABI.Methods[name]
		if !ok {
			panic(fmt.Sprintf("Pons ABI method %q is missing", name))
		}
		var selector [4]byte
		copy(selector[:], method.ID)
		methods[selector] = struct{}{}
	}
	return methods
}

func signatureIDs(signatures ...string) map[[4]byte]struct{} {
	methods := make(map[[4]byte]struct{}, len(signatures))
	for _, signature := range signatures {
		digest := crypto.Keccak256([]byte(signature))
		var selector [4]byte
		copy(selector[:], digest)
		if selector == ([4]byte{}) {
			panic("Pons method selector is zero")
		}
		methods[selector] = struct{}{}
	}
	return methods
}
