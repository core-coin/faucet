package chain

import (
	"math/big"

	"github.com/core-coin/go-core/common"
	// "github.com/ethereum/go-ethereum/common"
)

func IsValidAddress(address string, checksummed bool) bool {
	if !common.IsHexAddress(address) {
		return false
	}
	addr, err := common.HexToAddress(address)
	if err != nil {
		panic(err)
	}
	return !checksummed || addr.Hex() == address
}

func EtherToWei(amount int64) *big.Int {
	ether := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	return new(big.Int).Mul(big.NewInt(amount), ether)
}
