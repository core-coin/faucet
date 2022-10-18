package cmd

import (
	//"crypto/ecdsa"
	"errors"
	"flag"
	"fmt"
	"git.energy/corepass/corepass-http-helpers/logger"
	"github.com/core-coin/faucet/internal/storage/postgres"
	"github.com/core-coin/go-core/common"
	"github.com/core-coin/go-core/crypto"
	ecdsa "github.com/core-coin/go-goldilocks"
	post "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
	"os"
	"os/signal"

	"github.com/core-coin/faucet/internal/chain"
	"github.com/core-coin/faucet/internal/server"
)

var (
	appVersion = "v1.1.0"

	httpPortFlag = flag.Int("httpport", 8080, "Listener port to serve HTTP connection")
	proxyCntFlag = flag.Int("proxycount", 0, "Count of reverse proxies in front of the server")
	queueCapFlag = flag.Int("queuecap", 100, "Maximum transactions waiting to be sent")
	versionFlag  = flag.Bool("version", false, "Print version number")

	payoutFlag       = flag.Int("faucet.amount", 1, "Number of Сore Coins to transfer per user request")
	tokensPayoutFlag = flag.Int("faucet.tokens.amount", 200, "Number of Core Tokens to transfer per user request")
	intervalFlag     = flag.Int("faucet.minutes", 1440, "Number of minutes to wait between funding rounds")
	loggerLevelFlag  = flag.Int("logger.level", 3, "Logger level")

	privKeyFlag  = flag.String("wallet.privkey", os.Getenv("PRIVATE_KEY"), "Private key hex to fund user requests with")
	providerFlag = flag.String("wallet.provider", os.Getenv("WEB3_PROVIDER"), "Endpoint for Gocore JSON-RPC connection")

	postgresURLFlag   = flag.String("postgres.url", os.Getenv("POSTGRES_URL"), "Postgress URL")
	kycRequestURLFlag = flag.String("kyc.request.url", os.Getenv("KYC_REQUEST_URL"), "Hydra KYC Request URL")
	callbacksURLFlag  = flag.String("callback.url", os.Getenv("CALLBACK_URL"), "Hydra KYC Request Callback URL")

	registryAddressFlag = flag.String("registry.address", os.Getenv("REGISTRY_ADDRESS"), "Corepass registry address")
)

func init() {
	flag.Parse()
	if *versionFlag {
		fmt.Println(appVersion)
		os.Exit(0)
	}
}

func Execute() {
	common.DefaultNetworkID = common.NetworkID(3)

	httpLogger := logger.NewLogger(logger.LogLevel(*loggerLevelFlag))
	httpLogger.Debug("====================")
	flag.VisitAll(func(f *flag.Flag) {
		httpLogger.Debug(fmt.Sprintf("%s: %s\n", f.Name, f.Value))
	})
	httpLogger.Debug("====================")

	privateKey, err := getPrivateKeyFromFlags()
	if err != nil {
		panic(fmt.Errorf("failed to read private key: %w", err))
	}

	registryAddress, err := common.HexToAddress(*registryAddressFlag)
	if err != nil {
		panic(fmt.Errorf("cannot parse core token address: %w", err))
	}

	client, err := gorm.Open(post.Open(*postgresURLFlag), &gorm.Config{})
	if err != nil {
		log.Fatalln("could not connect to postgres: ", err)
	}
	if d, err := client.DB(); err == nil {
		if err = d.Ping(); err != nil {
			log.Fatalln("could not ping postgresql server: ", err)
		}
	}
	postgresClient := postgres.NewPostgresStorage(httpLogger, client)
	err = postgresClient.Migrate()
	if err != nil {
		log.Fatalln("could not migrate tables to postgres: ", err)
	}
	httpLogger.Info("connected to the postgresql server successfully")

	txBuilder, err := chain.NewTxBuilder(*providerFlag, privateKey, registryAddress)
	if err != nil {
		panic(fmt.Errorf("cannot connect to web3 provider: %w", err))
	}

	config := server.NewConfig(*httpPortFlag, *intervalFlag, *payoutFlag, *tokensPayoutFlag, *proxyCntFlag, *queueCapFlag)
	go server.NewServer(txBuilder, config, postgresClient, httpLogger, *kycRequestURLFlag, *callbacksURLFlag).Run()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
}

func getPrivateKeyFromFlags() (*ecdsa.PrivateKey, error) {
	if *privKeyFlag != "" {
		return crypto.HexToEDDSA(*privKeyFlag)
	}
	return nil, errors.New("missing private key")
}
