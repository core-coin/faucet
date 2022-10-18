package chain

import (
	"context"
	"errors"
	"math/big"

	"github.com/core-coin/faucet/internal/contracts"

	"github.com/core-coin/go-core/accounts/abi/bind"
	"github.com/core-coin/go-core/common"
	"github.com/core-coin/go-core/core/types"
	"github.com/core-coin/go-core/crypto"
	"github.com/core-coin/go-core/xcbclient"
	eddsa "github.com/core-coin/go-goldilocks"
)

type TxBuilder interface {
	Sender() common.Address
	Transfer(ctx context.Context, to string, value *big.Int) (common.Hash, error)
	TransferTokens(ctx context.Context, to string, value *big.Int) (common.Hash, error)
}

type TxBuild struct {
	client      bind.ContractTransactor
	networkID   *big.Int
	privateKey  *eddsa.PrivateKey
	signer      types.Signer
	fromAddress common.Address

	coreToken *contracts.CoreToken
}

func NewTxBuilder(provider string, privateKey *eddsa.PrivateKey, registryAddr common.Address) (TxBuilder, error) {
	client, err := xcbclient.Dial(provider)
	if err != nil {
		return nil, err
	}

	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		return nil, err
	}

	coreToken, err := contracts.GetCoreToken(context.Background(), registryAddr, client, crypto.PubkeyToAddress(eddsa.Ed448DerivePublicKey(*privateKey)))
	if err != nil {
		return nil, errors.New("cannot create core token instance: " + err.Error())
	}

	return &TxBuild{
		coreToken:   coreToken,
		networkID:   chainID,
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

	energyLimit := uint64(21000)
	energyPrice, err := b.client.SuggestEnergyPrice(ctx)
	if err != nil {
		return common.Hash{}, err
	}

	toAddress, err := common.HexToAddress(to)
	if err != nil {
		panic(err)
	}
	unsignedTx := types.NewTransaction(nonce, toAddress, value, energyLimit, energyPrice, []byte(""))

	signedTx, err := types.SignTx(unsignedTx, b.signer, b.privateKey)
	if err != nil {
		return common.Hash{}, err
	}

	return signedTx.Hash(), b.client.SendTransaction(ctx, signedTx)
}

func (b *TxBuild) TransferTokens(ctx context.Context, to string, value *big.Int) (common.Hash, error) {
	toAddress, err := common.HexToAddress(to)
	if err != nil {
		return common.Hash{}, err
	}
	opts, err := bind.NewKeyedTransactorWithNetworkID(b.privateKey, b.networkID)
	if err != nil {
		return common.Hash{}, err
	}
	opts.Context = ctx
	opts.From = b.fromAddress

	tx, err := b.coreToken.Transfer(opts, toAddress, value)
	if err != nil {
		return common.Hash{}, err
	}

	return tx.Hash(), nil
}
