package operator

import (
	"encoding/hex"
	"log"
	"os"
	"strings"

	solana "github.com/solana-labs/solana-go-sdk"
	"github.com/solana-labs/solana-go-sdk/crypto"
	"gopkg.in/yaml.v3"
)

type SolanaConfig struct {
	Profiles map[string]Profile `yaml:"profiles"`
}

type Profile struct {
	Network    string `yaml:"devnet"`
	PrivateKey string `yaml:"private_key"`
	PublicKey  string `yaml:"public_key"`
	Account    string `yaml:"account"`
	RestURL    string `yaml:"rest_url"`
	FaucetURL  string `yaml:"faucet_url"`
}

func SignerFromConfig(path string, profile string) (*solana.Account, error) {
	yamlFile, err := os.ReadFile(path)
	if err != nil {
		log.Printf("yamlFile.Get err   #%v ", err)
	}

	var solanaConfig SolanaConfig
	err = yaml.Unmarshal(yamlFile, &solanaConfig)
	if err != nil {
		log.Fatalf("Unmarshal: %v", err)
	}

	trimmedPriv := strings.TrimPrefix(solanaConfig.Profiles[profile].PrivateKey, "0x")
	trimmedPub := strings.TrimPrefix(solanaConfig.Profiles[profile].PublicKey, "0x")

	privateKeyBytes, err := hex.DecodeString(trimmedPriv)
	if err != nil {
		panic("Could not get operator priv key:" + err.Error())
	}
	publicKeyBytes, err := hex.DecodeString(trimmedPub)
	if err != nil {
		panic("Could not get operator pub key:" + err.Error())
	}

	privateKey := make([]byte, 64)
	copy(privateKey[:32], privateKeyBytes)
	copy(privateKey[32:], publicKeyBytes)
	priv := crypto.Ed25519PrivateKey{
		Inner: privateKey,
	}
	// Create the sender from the key locally
	sender, err := solana.NewAccountFromSigner(&priv)
	if err != nil {
		panic("Could not get account from signer:" + err.Error())
	}
	return sender, nil
}
