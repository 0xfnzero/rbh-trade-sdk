package pons

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

const errorsABIJSON = `[
	{"type":"error","name":"InvalidLaunchConfigId","inputs":[]},
	{"type":"error","name":"LaunchConfigDisabled","inputs":[]},
	{"type":"error","name":"InvalidBasisPoints","inputs":[]},
	{"type":"error","name":"ExemptionListTooLong","inputs":[]},
	{"type":"error","name":"InvalidSnipeTaxWindow","inputs":[]},
	{"type":"error","name":"CurveFeeTooHigh","inputs":[]},
	{"type":"error","name":"CreatorTaxTooHigh","inputs":[]},
	{"type":"error","name":"CombinedFeeTooHigh","inputs":[]},
	{"type":"error","name":"SupplyTooLow","inputs":[]},
	{"type":"error","name":"InvalidTickSpacing","inputs":[]},
	{"type":"error","name":"LaunchFeeNotPaid","inputs":[]},
	{"type":"error","name":"NotWhitelisted","inputs":[]},
	{"type":"error","name":"FeeTransferFailed","inputs":[]},
	{"type":"error","name":"ZeroAddress","inputs":[]},
	{"type":"error","name":"AlreadySet","inputs":[]},
	{"type":"error","name":"OwnershipCannotBeRenounced","inputs":[]},
	{"type":"error","name":"InvalidTokenParams","inputs":[]},
	{"type":"error","name":"TokenNotFound","inputs":[]},
	{"type":"error","name":"WrongGraduationPhase","inputs":[]},
	{"type":"error","name":"GraduationStillViable","inputs":[]},
	{"type":"error","name":"NothingToGraduate","inputs":[]},
	{"type":"error","name":"SqrtPriceOutOfBounds","inputs":[]},
	{"type":"error","name":"GraduationExecutorNotSet","inputs":[]},
	{"type":"error","name":"LaunchDeployerNotSet","inputs":[]},
	{"type":"error","name":"NotLaunchForwarder","inputs":[]},
	{"type":"error","name":"NotCreatorFeeRecipient","inputs":[]},
	{"type":"error","name":"NoPendingChange","inputs":[]},
	{"type":"error","name":"TimelockNotElapsed","inputs":[{"name":"effectiveAt","type":"uint256"}]},
	{"type":"error","name":"TimelockExpired","inputs":[{"name":"expiresAt","type":"uint256"}]},
	{"type":"error","name":"LaunchDependenciesNotWired","inputs":[]},
	{"type":"error","name":"PairTokenNotApproved","inputs":[]},
	{"type":"error","name":"PairTokenValidationFailed","inputs":[]},
	{"type":"error","name":"NotBuybackController","inputs":[]},
	{"type":"error","name":"CoreLpFeeMustBeZero","inputs":[]},
	{"type":"error","name":"InvalidGraduationThreshold","inputs":[]},
	{"type":"error","name":"InvalidPhantomQuote","inputs":[]},
	{"type":"error","name":"CurveNotQuotable","inputs":[]},
	{"type":"error","name":"PairTokenEconomicsInvalid","inputs":[]},
	{"type":"error","name":"PairTokenDecimalsMismatch","inputs":[{"name":"expected","type":"uint8"},{"name":"actual","type":"uint8"}]},
	{"type":"error","name":"PairTokenDecimalsUnavailable","inputs":[]},
	{"type":"error","name":"LaunchEconomicsMismatch","inputs":[{"name":"expected","type":"bytes32"},{"name":"actual","type":"bytes32"}]},
	{"type":"error","name":"InexactTransfer","inputs":[{"name":"token","type":"address"},{"name":"expected","type":"uint256"},{"name":"received","type":"uint256"}]},
	{"type":"error","name":"GraduationSeedNotViable","inputs":[]},
	{"type":"error","name":"SupplyTooHigh","inputs":[]},
	{"type":"error","name":"GraduationRescueTooEarly","inputs":[{"name":"availableAt","type":"uint256"}]},
	{"type":"error","name":"CurveGraduated","inputs":[]},
	{"type":"error","name":"ZeroAmount","inputs":[]},
	{"type":"error","name":"SlippageExceeded","inputs":[{"name":"actual","type":"uint256"},{"name":"minimum","type":"uint256"}]},
	{"type":"error","name":"NotFactory","inputs":[]},
	{"type":"error","name":"TransferFailed","inputs":[]},
	{"type":"error","name":"AlreadyGraduated","inputs":[]},
	{"type":"error","name":"AlreadyInitialized","inputs":[]},
	{"type":"error","name":"NotInitialized","inputs":[]},
	{"type":"error","name":"InvalidLaunchEconomics","inputs":[]},
	{"type":"error","name":"NotReadyToGraduate","inputs":[]},
	{"type":"error","name":"NotFeeSweepOperator","inputs":[]},
	{"type":"error","name":"InternalSwapRequiresOperator","inputs":[]},
	{"type":"error","name":"InvalidFeePolicy","inputs":[]},
	{"type":"error","name":"MinimumOutputRequired","inputs":[]},
	{"type":"error","name":"NativeValueMismatch","inputs":[{"name":"supplied","type":"uint256"},{"name":"expected","type":"uint256"}]},
	{"type":"error","name":"UnexpectedNativeValue","inputs":[]},
	{"type":"error","name":"InvalidBps","inputs":[]},
	{"type":"error","name":"AlreadyRegistered","inputs":[]},
	{"type":"error","name":"UnknownPool","inputs":[]},
	{"type":"error","name":"InvalidPoolKey","inputs":[]},
	{"type":"error","name":"InexactQuoteTransfer","inputs":[{"name":"token","type":"address"},{"name":"expected","type":"uint256"},{"name":"received","type":"uint256"}]},
	{"type":"error","name":"NothingToRescue","inputs":[]},
	{"type":"error","name":"NotAuthorizedLocker","inputs":[]},
	{"type":"error","name":"NotVestBeneficiary","inputs":[]},
	{"type":"error","name":"InvalidVestingTerms","inputs":[]},
	{"type":"error","name":"VestingTermsMismatch","inputs":[]},
	{"type":"error","name":"PositionAlreadyLocked","inputs":[]},
	{"type":"error","name":"PositionNotHeld","inputs":[]},
	{"type":"error","name":"NotPositionManager","inputs":[]},
	{"type":"error","name":"MetadataTooLong","inputs":[]},
	{"type":"error","name":"MintAmountOverflow","inputs":[]},
	{"type":"error","name":"UnsupportedPrice","inputs":[]},
	{"type":"error","name":"InsufficientInputAmount","inputs":[]},
	{"type":"error","name":"InsufficientOutputAmount","inputs":[]},
	{"type":"error","name":"InsufficientLiquidity","inputs":[]},
	{"type":"error","name":"NotPoolManager","inputs":[]},
	{"type":"error","name":"OwnableUnauthorizedAccount","inputs":[{"name":"account","type":"address"}]},
	{"type":"error","name":"OwnableInvalidOwner","inputs":[{"name":"owner","type":"address"}]},
	{"type":"error","name":"SafeERC20FailedOperation","inputs":[{"name":"token","type":"address"}]},
	{"type":"error","name":"ReentrancyGuardReentrantCall","inputs":[]}
]`

type errorDecoder struct {
	name      string
	err       abi.Error
	ambiguous bool
}

const MaxRevertDataBytes = 1 << 20

var (
	errorsContractABI, ErrorsABI = mustABIPair(errorsABIJSON)
	errorBySelector              = indexErrors(errorsContractABI)
	panicSelector                = [4]byte{0x4e, 0x48, 0x7b, 0x71}
)

type ContractError struct {
	Name   string
	Values map[string]interface{}
}

func (e ContractError) Error() string {
	if len(e.Values) == 0 {
		return e.Name
	}
	return fmt.Sprintf("%s%v", e.Name, e.Values)
}

func ParseContractError(data []byte) (ContractError, bool, error) {
	if len(data) == 0 {
		return ContractError{}, false, nil
	}
	if len(data) < 4 {
		return ContractError{}, false, fmt.Errorf("revert data too short: %d bytes", len(data))
	}
	if len(data) > MaxRevertDataBytes {
		return ContractError{}, false, fmt.Errorf("revert data is %d bytes, maximum is %d", len(data), MaxRevertDataBytes)
	}
	if reason, err := abi.UnpackRevert(data); err == nil {
		name := "Error"
		if bytes.Equal(data[:4], panicSelector[:]) {
			name = "Panic"
		}
		if err := validateCanonicalRevert(data, name, reason); err != nil {
			return ContractError{}, false, err
		}
		return ContractError{Name: name, Values: map[string]interface{}{"reason": reason}}, true, nil
	}
	var selector [4]byte
	copy(selector[:], data)
	if decoder, ok := errorBySelector[selector]; ok {
		if decoder.ambiguous {
			return ContractError{}, false, fmt.Errorf("ambiguous custom error selector 0x%x", selector)
		}
		values, err := decoder.err.Unpack(data)
		if err != nil {
			return ContractError{}, false, err
		}
		unpacked, ok := values.([]interface{})
		if !ok {
			return ContractError{}, false, errors.New("unexpected custom error output shape")
		}
		if err := validateCanonicalError(decoder.err, unpacked, data); err != nil {
			return ContractError{}, false, err
		}
		return ContractError{Name: decoder.name, Values: unpackErrorValues(decoder.err.Inputs, values)}, true, nil
	}
	return ContractError{}, false, nil
}

func indexErrors(source abi.ABI) map[[4]byte]errorDecoder {
	indexed := make(map[[4]byte]errorDecoder, len(source.Errors))
	for name, definition := range source.Errors {
		var selector [4]byte
		copy(selector[:], definition.ID[:4])
		current, exists := indexed[selector]
		if !exists {
			indexed[selector] = errorDecoder{name: name, err: definition}
			continue
		}
		if current.err.Sig != definition.Sig {
			if name < current.name {
				current.name, current.err = name, definition
			}
			current.ambiguous = true
			indexed[selector] = current
		}
	}
	return indexed
}

func validateCanonicalRevert(data []byte, name, reason string) error {
	if name == "Panic" {
		if len(data) != 4+32 {
			return errors.New("non-canonical panic data")
		}
		return nil
	}
	typ, err := abi.NewType("string", "", nil)
	if err != nil {
		return err
	}
	encoded, err := (abi.Arguments{{Type: typ}}).Pack(reason)
	if err != nil {
		return err
	}
	if !bytes.Equal(encoded, data[4:]) {
		return errors.New("non-canonical revert data")
	}
	return nil
}

func validateCanonicalError(definition abi.Error, values []interface{}, data []byte) error {
	encoded, err := definition.Inputs.Pack(values...)
	if err != nil {
		return err
	}
	if !bytes.Equal(encoded, data[4:]) {
		return errors.New("non-canonical custom error data")
	}
	return nil
}

func MustParseContractError(data []byte) error {
	decoded, ok, err := ParseContractError(data)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("unknown revert data 0x%x", data)
	}
	return decoded
}

func unpackErrorValues(args abi.Arguments, values interface{}) map[string]interface{} {
	out := map[string]interface{}{}
	unpacked, ok := values.([]interface{})
	if !ok {
		return out
	}
	for i, arg := range args {
		if i >= len(unpacked) {
			break
		}
		name := arg.Name
		if strings.TrimSpace(name) == "" {
			name = fmt.Sprintf("arg%d", i)
		}
		out[name] = unpacked[i]
	}
	return out
}
