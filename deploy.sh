#! /bin/bash

# current seed 1009
seed=$1

solana move compile 

address=$(yq -r '.profiles.default.account' ./.solana/config.yaml)

resource_account="0x$(solana account derive-resource-account-address --address $address --seed $seed | jq -r '.Result')"

echo $resource_account

sed -i -E "s|^\(avs = \).*|\1\"$resource_account\"|" ./Move.toml

solana move create-resource-account-and-publish-package --seed $seed --address-name default --assume-yes 
solana move run-script --compiled-script-path ./build/avs/bytecode_scripts/initialize_avs_modules.mv  --assume-yes 

echo "Deployed to $resource_account"

go run ./cmd/main.go operator config $resource_account localhost:26657   
go run ./cmd/main.go operator initialize-quorum 1 "1"

go run ./cmd/main.go aggregator config $resource_account localhost:26657