#!/usr/bin/sh

ylem \
    --output-dir . \
    --combined-json abi,bin \
    @coretoken/smart-contracts/=$(pwd | xargs dirname)/smart-contracts \
    ./smart-contracts/contracts/coretoken/CoreToken.sol \
    ./smart-contracts/contracts/Registry.sol \

abigen \
    --combined-json combined.json \
    --pkg contracts \
    --out generated.go

rm combined.json