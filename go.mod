module github.com/core-coin/faucet

// +heroku goVersion go1.16
go 1.16

require (
	git.energy/corepass/corepass-http-helpers v0.0.0-20221007095702-daa9b0e93fd5
	github.com/LK4D4/trylock v0.0.0-20191027065348-ff7e133a5c54
	github.com/core-coin/go-core v1.1.6-0.20221013165106-d491a447ce1b
	github.com/core-coin/go-goldilocks v1.0.12
	github.com/ethereum/go-ethereum v1.10.17
	github.com/golang-jwt/jwt/v4 v4.3.0
	github.com/jellydator/ttlcache/v2 v2.11.1
	github.com/sirupsen/logrus v1.8.1
	github.com/urfave/negroni v1.0.0
	golang.org/x/crypto v0.0.0-20220722155217-630584e8d5aa
	gorm.io/driver/postgres v1.4.4
	gorm.io/gorm v1.24.0
)
