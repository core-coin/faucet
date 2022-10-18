//go:generate sh generate.sh

package contracts

import (
	"context"
	"github.com/core-coin/go-core/accounts/abi/bind"
	"github.com/core-coin/go-core/common"
	"golang.org/x/crypto/sha3"
)

func GetCoreToken(
	ctx context.Context,
	registryAddress common.Address,
	provider bind.ContractBackend,
	owner common.Address,
) (*CoreToken, error) {
	registry, err := NewRegistry(registryAddress, provider)
	if err != nil {
		return nil, err
	}

	key := sha3.Sum256([]byte("CTN"))
	coreToken, err := registry.Get(&bind.CallOpts{
		From:    owner,
		Context: ctx,
	}, key)
	if err != nil {
		return nil, err
	}

	coreTokenAddress, err := common.HexToAddress(string(coreToken))
	if err != nil {
		return nil, err
	}

	return NewCoreToken(coreTokenAddress, provider)
}
