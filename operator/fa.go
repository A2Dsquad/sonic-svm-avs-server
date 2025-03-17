package operator

import (
	"encoding/json"
	"fmt"

	solana "github.com/solana-labs/solana-go-sdk"
)

// TODO
const FAContract = "0x24f4512994b08ea6f6914a776670d0e06541b2e72ffcbf32a279be4c8eb4e845"

func GetMetadata(
	client *solana.Client,
) Metadata {
	contractAcc := solana.AccountAddress{}
	err := contractAcc.ParseStringRelaxed(FAContract)
	if err != nil {
		panic("Could not ParseStringRelaxed:" + err.Error())
	}

	var noTypeTags []solana.TypeTag
	viewResponse, err := client.View(&solana.ViewPayload{
		Module: solana.ModuleId{
			Address: contractAcc,
			Name:    "fungible_asset",
		},
		Function: "get_metadata",
		ArgTypes: noTypeTags,
		Args:     [][]byte{},
	})
	if err != nil {
		panic("Failed to view fa address:" + err.Error())
	}
	metadataMap := viewResponse[0].(map[string]interface{})
	metadataBz, err := json.Marshal(metadataMap)
	if err != nil {
		panic("Failed to marshal metadata to json:" + err.Error())
	}

	var metadataStr MetadataStr
	err = json.Unmarshal(metadataBz, &metadataStr)
	if err != nil {
		panic("Failed to unmarshal metadata from json:" + err.Error())
	}
	metadataAcc := solana.AccountAddress{}
	err = metadataAcc.ParseStringRelaxed(metadataStr.Inner)
	if err != nil {
		panic("Could not ParseStringRelaxed:" + err.Error())
	}

	fmt.Println("metadata: ", metadataAcc.String())
	return Metadata{
		Inner: metadataAcc,
	}
}

func FAMetdataClient(
	client *solana.Client,
) *solana.FungibleAssetClient {
	contractAcc := solana.AccountAddress{}
	err := contractAcc.ParseStringRelaxed(FAContract)
	if err != nil {
		panic("Could not ParseStringRelaxed:" + err.Error())
	}

	var noTypeTags []solana.TypeTag
	viewResponse, err := client.View(&solana.ViewPayload{
		Module: solana.ModuleId{
			Address: contractAcc,
			Name:    "fungible_asset",
		},
		Function: "get_metadata",
		ArgTypes: noTypeTags,
		Args:     [][]byte{},
	})
	if err != nil {
		panic("Failed to view fa address:" + err.Error())
	}

	metadataMap := viewResponse[0].(map[string]interface{})
	metadataBz, err := json.Marshal(metadataMap)
	if err != nil {
		panic("Failed to marshal metadata to json:" + err.Error())
	}

	var metadataStr MetadataStr
	err = json.Unmarshal(metadataBz, &metadataStr)
	if err != nil {
		panic("Failed to unmarshal metadata from json:" + err.Error())
	}
	metadataAcc := solana.AccountAddress{}
	err = metadataAcc.ParseStringRelaxed(metadataStr.Inner)
	if err != nil {
		panic("Could not ParseStringRelaxed:" + err.Error())
	}

	faClient, err := solana.NewFungibleAssetClient(client, &metadataAcc)
	if err != nil {
		panic("Failed to create fa client:" + err.Error())
	}
	return faClient
}
