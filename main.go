package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"avs/operator"
	"os"

	"github.com/solana-labs/solana-go-sdk"
	"github.com/solana-labs/solana-go-sdk/crypto"
	"golang.org/x/crypto/ed25519"
)

type Config struct {
	CmcApi string `json:"cmc_api"`
}

func loadConfig(filename string) (*Config, error) {
	// Open the config file
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("error opening config file: %v", err)
	}
	defer file.Close()

	// Read the file contents
	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %v", err)
	}

	// Unmarshal the JSON data into the Config struct
	var config Config
	err = json.Unmarshal(bytes, &config)
	if err != nil {
		return nil, fmt.Errorf("error parsing config file: %v", err)
	}

	return &config, nil
}

type AlternativeSigner struct {
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
}

func (signer *AlternativeSigner) PublicKey() *crypto.Ed25519PublicKey {
	pubKey := &crypto.Ed25519PublicKey{}
	err := pubKey.FromBytes(signer.publicKey)
	if err != nil {
		panic("Public key is not valid")
	}
	return pubKey
}

func (signer *AlternativeSigner) PubKey() crypto.PublicKey {
	return signer.PublicKey()
}

func (signer *AlternativeSigner) ToHex() string {
	return ""
}

func (signer *AlternativeSigner) Sign(msg []byte) (authenticator *crypto.AccountAuthenticator, err error) {
	sig, err := signer.SignMessage(msg)
	if err != nil {
		return nil, err
	}
	pubKey := signer.PublicKey()
	auth := &crypto.Ed25519Authenticator{
		PubKey: pubKey,
		Sig:    sig.(*crypto.Ed25519Signature),
	}
	// TODO: maybe make convenience functions for this
	return &crypto.AccountAuthenticator{
		Variant: crypto.AccountAuthenticatorEd25519,
		Auth:    auth,
	}, nil
}

func (signer *AlternativeSigner) SignMessage(msg []byte) (crypto.Signature, error) {
	sigBytes := ed25519.Sign(signer.privateKey, msg)
	sig := &crypto.Ed25519Signature{}
	copy(sig.Inner[:], sigBytes)
	return sig, nil
}

func (signer *AlternativeSigner) SimulationAuthenticator() *crypto.AccountAuthenticator {
	return &crypto.AccountAuthenticator{
		Variant: crypto.AccountAuthenticatorEd25519,
		Auth: &crypto.Ed25519Authenticator{
			PubKey: signer.PublicKey(),
			Sig:    &crypto.Ed25519Signature{},
		},
	}
}

func (signer *AlternativeSigner) AuthKey() *crypto.AuthenticationKey {
	authKey := &crypto.AuthenticationKey{}
	pubKey := signer.PublicKey()
	authKey.FromPublicKey(pubKey)
	return authKey
}

func example(network solana.NetworkConfig) {
	client, err := solana.NewClient(network)
	if err != nil {
		panic("Failed to create client:" + err.Error())
	}

	sender, err := operator.SignerFromConfig(".solana/config.yaml", "default")
	contract := solana.AccountAddress{}
	err = contract.ParseStringRelaxed("0xc453ce803fb6f9720d7dec971c3ded0beb633c28e09ef2f5809e4964c9d5d8e2")
	if err != nil {
		panic("Failed to parse address:" + err.Error())
	}

	echo := "lmao"
	payload := solana.EntryFunction{
		Module: solana.ModuleId{
			Address: contract,
			Name:    "lmao_echo",
		},
		Function: "echo",
		ArgTypes: []solana.TypeTag{},
		Args: [][]byte{
			[]byte(echo),
		},
	}

	// Build transaction
	rawTxn, err := client.BuildTransaction(sender.AccountAddress(),
		solana.TransactionPayload{Payload: &payload})
	if err != nil {
		panic("Failed to build transaction:" + err.Error())
	}

	// Sign transaction
	signedTxn, err := rawTxn.SignedTransaction(sender)
	if err != nil {
		panic("Failed to sign transaction:" + err.Error())
	}

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
}

// main This example shows you how to make an alternative signer for the SDK, if you prefer a different library
func main() {
	// example(solana.DevnetConfig)
	a, err := operator.SignerFromConfig(".solana/config.yaml", "default")
	fmt.Println("a: ", a.Address.String())
	fmt.Println("err: ", err)
}
