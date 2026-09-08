package main

import (
	"fmt"
	"math/big"
	"time"

	sdk "github.com/0xfnzero/rbh-trade-sdk/rbhtrade"
	"github.com/ethereum/go-ethereum/common"
)

func main() {
	token := common.HexToAddress("0x0000000000000000000000000000000000000010")
	deadlineUnix := time.Now().Add(30 * time.Second).Unix()
	if deadlineUnix <= 0 {
		panic("deadline is before Unix epoch")
	}
	call, err := sdk.BuildV4ExactInputSingle(sdk.ExactInputRequest{
		PoolKey: sdk.PoolKey{
			Currency0:   common.Address{},
			Currency1:   token,
			Fee:         3000,
			TickSpacing: 60,
		},
		CurrencyIn:       common.Address{},
		CurrencyOut:      token,
		AmountIn:         big.NewInt(1_000_000_000_000_000),
		AmountOutMinimum: big.NewInt(1),
		Deadline:         uint64(deadlineUnix), // #nosec G115 -- positive value checked above.
	})
	if err != nil {
		panic(err)
	}
	fmt.Printf("to=%s value=%s calldata=0x%x\n", call.To, call.Value, call.Data)
}
