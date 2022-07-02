package chain

import (
	"context"
	"math/big"

	"github.com/core-coin/go-core/accounts/abi/bind"
	"github.com/core-coin/go-core/common"
	"github.com/core-coin/go-core/core/types"
	"github.com/core-coin/go-core/crypto"
	ethclient "github.com/core-coin/go-core/xcbclient"
	eddsa "github.com/core-coin/go-goldilocks"
)

type TxBuilder interface {
	Sender() common.Address
	Transfer(ctx context.Context, to string, value *big.Int) (common.Hash, error)
}

type TxBuild struct {
	client      bind.ContractTransactor
	privateKey  *eddsa.PrivateKey
	signer      types.Signer
	fromAddress common.Address
}

func NewTxBuilder(provider string, privateKey *eddsa.PrivateKey, chainID *big.Int) (TxBuilder, error) {
	client, err := ethclient.Dial(provider)
	if err != nil {
		return nil, err
	}

	if chainID == nil {
		chainID, err = client.NetworkID(context.Background())
		if err != nil {
			return nil, err
		}
	}

	return &TxBuild{
		client:      client,
		privateKey:  privateKey,
		signer:      types.MakeSigner(chainID),
		fromAddress: crypto.PubkeyToAddress(eddsa.PrivateToPublic(*privateKey)),
	}, nil
}

func (b *TxBuild) Sender() common.Address {
	return b.fromAddress
}

func (b *TxBuild) Transfer(ctx context.Context, to string, value *big.Int) (common.Hash, error) {
	nonce, err := b.client.PendingNonceAt(ctx, b.Sender())
	if err != nil {
		return common.Hash{}, err
	}

	gasLimit := uint64(21000)
	gasPrice, err := b.client.SuggestEnergyPrice(ctx)
	if err != nil {
		return common.Hash{}, err
	}

	toAddress, err := common.HexToAddress(to)
	if err != nil {
		panic(err)
	}
	unsignedTx := types.NewTransaction(nonce, toAddress, value, gasLimit, gasPrice, []byte(""))
	/*
		unsignedTx := types.NewTx(&types.LegacyTx{
			Nonce:    nonce,
			To:       &toAddress,
			Value:    value,
			Gas:      gasLimit,
			GasPrice: gasPrice,
		})
	*/

	signedTx, err := types.SignTx(unsignedTx, b.signer, b.privateKey)
	if err != nil {
		return common.Hash{}, err
	}

	return signedTx.Hash(), b.client.SendTransaction(ctx, signedTx)
}
