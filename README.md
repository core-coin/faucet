# XAB Faucet

The faucet is a web application with the goal of distributing small amounts of Core in private and test networks.

## Features

* Allow to configure the funding account via private key or keystore.
* Asynchronous processing Txs to achieve parallel execution of user requests.
* Rate limiting by XAB address and IP address as a precaution against spam.
* Prevent X-Forwarded-For spoofing by specifying the count of reverse proxies.

## Get started

### Prerequisites

* Go (1.16 or later)
* Node.js
* Yarn

### Installation

1. Clone the repository and navigate to the app’s directory
```bash
git clone https://github.com/core-coin/faucet.git
cd faucet
```

2. Bundle Frontend web with Rollup
```bash
yarn build
```

3. Build Go project
```bash
go build -o faucet
```

## Usage

**Use private key to fund users**

```bash
./faucet -httpport 8080 -wallet.provider http://localhost:8545 -wallet.privkey privkey
```

**Use keystore to fund users**

```bash
./faucet -httpport 8080 -wallet.provider http://localhost:8545 -wallet.keyjson keystore -wallet.keypass password.txt
```

### Configuration

You can configure the funder by using environment variables instead of command-line flags as follows:
```bash
export WEB3_PROVIDER=rpc endpoint
export PRIVATE_KEY=hex private key
```

or

```bash
export WEB3_PROVIDER=rpc endpoint
export KEYSTORE=keystore path
echo "your keystore password" > `pwd`/password.txt
```

Then run the faucet application without the wallet command-line flags:
```bash
./faucet -httpport 8080
```

**Optional Flags**

The following are the available command-line flags(excluding above wallet flags):

| Flag           | Description                                      | Default Value
| -------------- | ------------------------------------------------ | -------------
| -httpport      | Listener port to serve HTTP connection           | 8080
| -proxycount    | Count of reverse proxies in front of the server  | 0
| -queuecap      | Maximum transactions waiting to be sent          | 100
| -faucet.amount | Number of Core to transfer per user request      | 1
| -faucet.minutes| Number of minutes to wait between funding rounds | 1440
| -faucet.name   | Network name to display on the frontend          | devin

### Docker deployment

```bash
docker run -d -p 8080:8080 -e WEB3_PROVIDER=rpc endpoint -e PRIVATE_KEY=hex private key core-coin/faucet
```

or

```bash
docker run -d -p 8080:8080 -e WEB3_PROVIDER=rpc endpoint -e KEYSTORE=keystore path -v `pwd`/keystore:/app/keystore -v `pwd`/password.txt:/app/password.txt core-coin/faucet
```

### Heroku deployment

```bash
heroku create
heroku buildpacks:add heroku/nodejs
heroku buildpacks:add heroku/go
heroku config:set WEB3_PROVIDER=rpc endpoint
heroku config:set PRIVATE_KEY=hex private key
git push heroku main
heroku open
```

or

[![Deploy](https://www.herokucdn.com/deploy/button.png)](https://heroku.com/deploy)

> tip: Free web dyno goes to sleep and discards in-memory rate limiting records after 30 minutes of inactivity, so `faucet.minutes` configuration greater than 30 doesn't work properly in the free Heroku plan.

## License

Distributed under the CORE License. See LICENSE for more information.
