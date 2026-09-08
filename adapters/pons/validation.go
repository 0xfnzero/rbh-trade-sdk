package pons

import (
	"errors"
	"fmt"
	"math/big"
	"reflect"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

var (
	ErrNilBackend                 = errors.New("nil contract backend")
	ErrNilAuth                    = errors.New("nil transaction options")
	ErrZeroAddress                = errors.New("zero contract address")
	ErrWrongChain                 = errors.New("unexpected chain ID")
	ErrUnexpectedTransactionValue = errors.New("unexpected transaction value for non-payable method")
)

func validateChainID(chainID *big.Int) error {
	if chainID == nil || !chainID.IsInt64() || chainID.Int64() != ChainID {
		return fmt.Errorf("%w: got %v, want %d", ErrWrongChain, chainID, ChainID)
	}
	return nil
}

func validateNonPayableAuth(auth *bind.TransactOpts) error {
	if err := validateAuth(auth); err != nil {
		return err
	}
	if auth.Value != nil && auth.Value.Sign() != 0 {
		return ErrUnexpectedTransactionValue
	}
	return nil
}

func validateAddress(address common.Address, name string) error {
	if address == (common.Address{}) {
		return fmt.Errorf("%w: %s", ErrZeroAddress, name)
	}
	return nil
}

func isNilInterface(value interface{}) bool {
	if value == nil {
		return true
	}
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return rv.IsNil()
	default:
		return false
	}
}

// Validate verifies every contract address used directly by Client methods.
func (a Addresses) Validate() error {
	checks := []struct {
		address common.Address
		name    string
	}{
		{a.Factory, "factory"},
		{a.MemeHook, "meme hook"},
		{a.FeeEscrow, "fee escrow"},
		{a.BuybackVault, "buyback vault"},
		{a.LaunchLocker, "launch locker"},
		{a.LaunchAndBuy, "launch-and-buy router"},
	}
	for _, check := range checks {
		if err := validateAddress(check.address, check.name); err != nil {
			return err
		}
	}
	return nil
}

func validateUint256(value *big.Int, name string) error {
	if value == nil {
		return fmt.Errorf("%w: %s must not be nil", ErrInvalidAmount, name)
	}
	if value.Sign() < 0 || value.BitLen() > 256 {
		return fmt.Errorf("%w: %s must fit uint256", ErrInvalidAmount, name)
	}
	return nil
}

func validatePositiveUint256(value *big.Int, name string) error {
	if err := validateUint256(value, name); err != nil {
		return err
	}
	if value.Sign() == 0 {
		return fmt.Errorf("%w: %s must be positive", ErrInvalidAmount, name)
	}
	return nil
}

func validateAuth(auth *bind.TransactOpts) error {
	if auth == nil {
		return ErrNilAuth
	}
	if auth.Value != nil && (auth.Value.Sign() < 0 || auth.Value.BitLen() > 256) {
		return fmt.Errorf("%w: transaction value must fit uint256", ErrInvalidAmount)
	}
	return nil
}

func validateLaunch(params TokenParams, exemptions []common.Address) error {
	if params.Name == "" || params.Symbol == "" {
		return errors.New("token name and symbol must not be empty")
	}
	metadata := []struct {
		name  string
		value string
		limit int
	}{
		{"token name", params.Name, MaxTokenNameBytes},
		{"token symbol", params.Symbol, MaxTokenSymbolBytes},
		{"token logo", params.Logo, MaxTokenLogoBytes},
		{"token description", params.Description, MaxTokenDescriptionBytes},
		{"twitter", params.Socials.Twitter, MaxTokenSocialBytes},
		{"telegram", params.Socials.Telegram, MaxTokenSocialBytes},
		{"discord", params.Socials.Discord, MaxTokenSocialBytes},
		{"website", params.Socials.Website, MaxTokenSocialBytes},
		{"farcaster", params.Socials.Farcaster, MaxTokenSocialBytes},
	}
	for _, field := range metadata {
		if len(field.value) > field.limit {
			return fmt.Errorf("%s exceeds %d bytes", field.name, field.limit)
		}
	}
	if len(exemptions) > MaxSnipeTaxExemptions {
		return fmt.Errorf("too many snipe tax exemptions: %d exceeds %d", len(exemptions), MaxSnipeTaxExemptions)
	}
	return nil
}
