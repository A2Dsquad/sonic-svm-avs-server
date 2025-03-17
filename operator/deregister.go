package operator

import (
	"fmt"

	solana "github.com/solana-labs/solana-go-sdk"
	"github.com/solana-labs/solana-go-sdk/bcs"
	"go.uber.org/zap"
)

func DeregisterFromQuorum(logger *zap.Logger, networkConfig solana.NetworkConfig, config OperatorConfig, operatorAccount *solana.Account, quorum uint8) error {
	client, err := solana.NewClient(networkConfig)
	if err != nil {
		panic("Failed to create client:" + err.Error())
	}

	contract := solana.AccountAddress{}
	err = contract.ParseStringRelaxed(config.AvsAddress)
	if err != nil {
		panic("Failed to parse address:" + err.Error())
	}


	quorumSerializer := &bcs.Serializer{}
	bcs.SerializeSequence([]U8Struct{
		{
			Value: quorum,
		},
	}, quorumSerializer)

	payload := solana.EntryFunction{
		Module: solana.ModuleId{
			Address: contract,
			Name:    "registry_coordinator",
		},
		Function: "deregister_operator",
		ArgTypes: []solana.TypeTag{},
		Args: [][]byte{
			quorumSerializer.ToBytes(),
		},
	}
	// Build transaction
	rawTxn, err := client.BuildTransaction(operatorAccount.AccountAddress(),
		solana.TransactionPayload{Payload: &payload})
	if err != nil {
		panic("Failed to build transaction:" + err.Error())
	}

	// Sign transaction
	signedTxn, err := rawTxn.SignedTransaction(operatorAccount)
	if err != nil {
		panic("Failed to sign transaction:" + err.Error())
	}
	fmt.Printf("Submit deregister operator for %s\n", operatorAccount.AccountAddress())

	// Submit and wait for it to complete
	submitResult, err := client.SubmitTransaction(signedTxn)
	if err != nil {
		panic("Failed to submit transaction:" + err.Error())
	}
	txnHash := submitResult.Hash

	// Wait for the transaction
	fmt.Printf("And we wait for the transaction %s to complete...\n", txnHash)
	userTxn, err := client.WaitForTransaction(txnHash)
	if err != nil {
		panic("Failed to wait for transaction:" + err.Error())
	}
	fmt.Printf("The transaction completed with hash: %s and version %d\n", userTxn.Hash, userTxn.Version)
	if !userTxn.Success {
		// TODO: log something more
		panic("Failed to deregister operator")
	}
	return nil
}
