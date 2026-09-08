package pons

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"reflect"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

func RandomSalt() ([32]byte, error) {
	var salt [32]byte
	_, err := rand.Read(salt[:])
	return salt, err
}

func HashToBytes32(hash common.Hash) [32]byte {
	return [32]byte(hash)
}

func IsNativeQuote(pairToken common.Address) bool {
	return pairToken == (common.Address{})
}

func LaunchTokenValue(launchFee *big.Int) *big.Int {
	return cloneBig(launchFee)
}

func BuyValue(pairToken common.Address, quoteIn *big.Int) *big.Int {
	if IsNativeQuote(pairToken) {
		return cloneBig(quoteIn)
	}
	return new(big.Int)
}

func LaunchAndBuyValue(pairToken common.Address, launchFee *big.Int, quoteIn *big.Int) *big.Int {
	value := cloneBig(launchFee)
	if IsNativeQuote(pairToken) {
		value.Add(value, cloneBig(quoteIn))
	}
	return value
}

func mapABIValue(value interface{}, out interface{}) (err error) {
	if out == nil || reflect.ValueOf(out).Kind() != reflect.Ptr {
		return errors.New("output must be a pointer")
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("convert ABI value: %v", recovered)
		}
	}()
	converted := abi.ConvertType(value, out)
	rv := reflect.ValueOf(converted)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return errors.New("converted ABI value is not a pointer")
	}
	reflect.ValueOf(out).Elem().Set(rv.Elem())
	return nil
}

func uint24ToUint32(value *big.Int, name string) (uint32, error) {
	if value == nil || value.Sign() < 0 || value.BitLen() > 24 {
		return 0, fmt.Errorf("invalid ABI %s: expected uint24", name)
	}
	var encoded [4]byte
	value.FillBytes(encoded[:])
	return binary.BigEndian.Uint32(encoded[:]), nil
}

func int24ToInt32(value *big.Int, name string) (int32, error) {
	if value == nil || !value.IsInt64() {
		return 0, fmt.Errorf("invalid ABI %s: expected int24", name)
	}
	n := value.Int64()
	if n < -(1<<23) || n > (1<<23)-1 {
		return 0, fmt.Errorf("invalid ABI %s: expected int24", name)
	}
	return int32(n), nil
}

func convert[T any](value interface{}) (T, error) {
	var out T
	if err := mapABIValue(value, &out); err != nil {
		return out, err
	}
	return out, nil
}
