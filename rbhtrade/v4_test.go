package rbhtrade

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestBuildV4ExactInputSingle(t *testing.T) {
	token := common.HexToAddress("0x0000000000000000000000000000000000000010")
	key := PoolKey{Currency0: common.Address{}, Currency1: token, Fee: 3000, TickSpacing: 60, Hooks: common.HexToAddress("0x0000000000000000000000000000000000000020")}
	req := ExactInputRequest{PoolKey: key, CurrencyIn: common.Address{}, CurrencyOut: token, AmountIn: big.NewInt(1_000_000), AmountOutMinimum: big.NewInt(99), Deadline: 12345}
	call, err := BuildV4ExactInputSingle(req)
	if err != nil {
		t.Fatal(err)
	}
	if call.To != DefaultAddressBook().UniversalRouter || call.Value.Cmp(req.AmountIn) != 0 {
		t.Fatalf("unexpected call destination/value: %#v", call)
	}
	if got := common.Bytes2Hex(call.Data[:4]); got != "3593564c" {
		t.Fatalf("execute selector = %s", got)
	}
	outer, err := universalRouterABI.Methods["execute"].Inputs.Unpack(call.Data[4:])
	if err != nil {
		t.Fatal(err)
	}
	commands := outer[0].([]byte)
	inputs := outer[1].([][]byte)
	deadline := outer[2].(*big.Int)
	if !bytes.Equal(commands, []byte{CommandV4Swap}) || len(inputs) != 1 || deadline.Uint64() != req.Deadline {
		t.Fatalf("unexpected outer payload: %#v", outer)
	}
	plan, err := v4PlanArgs.Unpack(inputs[0])
	if err != nil {
		t.Fatal(err)
	}
	actions := plan[0].([]byte)
	params := plan[1].([][]byte)
	if !bytes.Equal(actions, []byte{ActionSwapExactInSingle, ActionSettleAll, ActionTakeAll}) || len(params) != 3 {
		t.Fatalf("unexpected v4 plan")
	}
	expectedSwap, err := v4ActionArgs.Pack(exactInputSingleABI{PoolKey: toPoolKeyABI(key), ZeroForOne: true, AmountIn: big.NewInt(1_000_000), AmountOutMinimum: big.NewInt(99), MinHopPriceX36: new(big.Int), HookData: []byte{}})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(params[0], expectedSwap) {
		t.Fatalf("swap action does not use Robinhood six-field encoding")
	}
}

func TestFastV4EncodingMatchesReferenceABI(t *testing.T) {
	token := common.HexToAddress("0x0000000000000000000000000000000000000010")
	key := PoolKey{Currency0: common.Address{}, Currency1: token, Fee: 0x800000, TickSpacing: 8, Hooks: common.HexToAddress("0x20")}
	for _, req := range []ExactInputRequest{
		{PoolKey: key, CurrencyIn: common.Address{}, CurrencyOut: token, AmountIn: big.NewInt(123), AmountOutMinimum: big.NewInt(99), MinHopPriceX36: big.NewInt(77), HookData: []byte{1, 2, 3}, Deadline: 456},
		{PoolKey: key, CurrencyIn: token, CurrencyOut: common.Address{}, AmountIn: big.NewInt(789), AmountOutMinimum: new(big.Int), HookData: make([]byte, 33), Deadline: 999},
	} {
		call, err := BuildV4ExactInputSingle(req)
		if err != nil {
			t.Fatal(err)
		}
		zeroForOne := req.CurrencyIn == key.Currency0
		minHop := cloneBig(req.MinHopPriceX36)
		swap, err := v4ActionArgs.Pack(exactInputSingleABI{PoolKey: toPoolKeyABI(key), ZeroForOne: zeroForOne, AmountIn: cloneBig(req.AmountIn), AmountOutMinimum: cloneBig(req.AmountOutMinimum), MinHopPriceX36: minHop, HookData: append([]byte(nil), req.HookData...)})
		if err != nil {
			t.Fatal(err)
		}
		settle, _ := addressUint256Args.Pack(req.CurrencyIn, new(big.Int).Set(maxUint256))
		take, _ := addressUint256Args.Pack(req.CurrencyOut, cloneBig(req.AmountOutMinimum))
		plan, _ := v4PlanArgs.Pack([]byte{ActionSwapExactInSingle, ActionSettleAll, ActionTakeAll}, [][]byte{swap, settle, take})
		reference, _ := universalRouterABI.Pack("execute", []byte{CommandV4Swap}, [][]byte{plan}, new(big.Int).SetUint64(req.Deadline))
		if !bytes.Equal(call.Data, reference) {
			t.Fatal("fast v4 encoding differs from canonical ABI encoding")
		}
	}
}

func TestBuildV4ExactInputSingleRejectsUnsafeInputs(t *testing.T) {
	token := common.HexToAddress("0x10")
	key := PoolKey{Currency0: common.Address{}, Currency1: token, Fee: 3000, TickSpacing: 60}
	tests := []ExactInputRequest{
		{PoolKey: key, CurrencyIn: common.Address{}, CurrencyOut: token, AmountIn: nil, AmountOutMinimum: big.NewInt(1), Deadline: 1},
		{PoolKey: key, CurrencyIn: common.Address{}, CurrencyOut: token, AmountIn: new(big.Int).Lsh(big.NewInt(1), 128), AmountOutMinimum: big.NewInt(1), Deadline: 1},
		{PoolKey: key, CurrencyIn: common.Address{}, CurrencyOut: token, AmountIn: big.NewInt(1), AmountOutMinimum: big.NewInt(-1), Deadline: 1},
		{PoolKey: key, CurrencyIn: common.Address{}, CurrencyOut: token, AmountIn: big.NewInt(1), AmountOutMinimum: big.NewInt(1), Deadline: 0},
		{PoolKey: key, CurrencyIn: common.HexToAddress("0x99"), CurrencyOut: token, AmountIn: big.NewInt(1), AmountOutMinimum: big.NewInt(1), Deadline: 1},
	}
	for i, test := range tests {
		if _, err := BuildV4ExactInputSingle(test); err == nil {
			t.Fatalf("case %d unexpectedly succeeded", i)
		}
	}
}

func TestDefaultAddressMutationCannotRedirectBuilder(t *testing.T) {
	original := RobinhoodChain
	defer func() { RobinhoodChain = original }()
	RobinhoodChain.UniversalRouter = common.HexToAddress("0xdead")
	token := common.HexToAddress("0x10")
	call, err := BuildV4ExactInputSingle(ExactInputRequest{PoolKey: PoolKey{Currency0: common.Address{}, Currency1: token, Fee: 3000, TickSpacing: 60}, CurrencyIn: common.Address{}, CurrencyOut: token, AmountIn: big.NewInt(1), AmountOutMinimum: new(big.Int), Deadline: 1})
	if err != nil {
		t.Fatal(err)
	}
	if call.To != DefaultAddressBook().UniversalRouter {
		t.Fatalf("builder destination was redirected to %s", call.To)
	}
}

func BenchmarkBuildV4ExactInputSingle(b *testing.B) {
	token := common.HexToAddress("0x10")
	req := ExactInputRequest{PoolKey: PoolKey{Currency0: common.Address{}, Currency1: token, Fee: 3000, TickSpacing: 60}, CurrencyIn: common.Address{}, CurrencyOut: token, AmountIn: big.NewInt(1_000_000), AmountOutMinimum: big.NewInt(1), Deadline: 1}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := BuildV4ExactInputSingle(req); err != nil {
			b.Fatal(err)
		}
	}
}
