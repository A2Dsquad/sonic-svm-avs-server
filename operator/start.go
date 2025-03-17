package operator

import (
	"fmt"
	"log"
	"math/big"

	solana "github.com/gagliardetto/solana-go"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"go.uber.org/zap"
)

type SolanaAccountConfig struct {
	configPath string
	profile    string
}

func SolanaClient(networkConfig solana.NetworkConfig) *solana.Client {
	// Create a client for Solana
	client, err := solana.NewClient(networkConfig)
	if err != nil {
		panic("Failed to create client:" + err.Error())
	}
	return client
}

func NewOperator(logger *zap.Logger, networkConfig solana.NetworkConfig, config OperatorConfig, accountConfig SolanaAccountConfig, blsPriv []byte) (*Operator, error) {
	operatorAccount, err := SignerFromConfig(accountConfig.configPath, accountConfig.profile)
	if err != nil {
		panic("Failed to create operator account:" + err.Error())
	}
	client, err := solana.NewClient(networkConfig)
	if err != nil {
		panic("Failed to create client:" + err.Error())
	}

	// Get operator Status
	avsAddress := solana.AccountAddress{}
	if err := avsAddress.ParseStringRelaxed(config.AvsAddress); err != nil {
		panic("Failed to parse avsAddress:" + err.Error())
	}
	registered := IsOperatorRegistered(client, avsAddress, operatorAccount.Address.String())

	if !registered {
		log.Println("Operator is not registered with A2D avs AVS, registering...")

		quorumCount := QuorumCount(client, avsAddress)
		if quorumCount == 0 {
			panic("No quorum found, please initialize quorum first ")
		}

		quorumNumbers := quorumCount
		fmt.Println("quorumNumbers:", quorumNumbers)
		// Register Operator
		// ignore error here because panic all the time
		var priv crypto.BlsPrivateKey
		msg := []byte("PubkeyRegistration")
		bcsOperatorAccount, err := bcs.Serialize(&operatorAccount.Address)
		if err != nil {
			panic("Failed to bsc serialize account" + err.Error())
		}

		msg = append(msg, bcsOperatorAccount...)
		keccakMsg := ethcrypto.Keccak256(msg)
		err = priv.FromBytes(config.BlsPrivateKey)
		if err != nil {
			panic("Failed to create bls priv key" + err.Error())
		}

		signature, err := priv.Sign(keccakMsg)
		if err != nil {
			panic("Failed to create signature" + err.Error())
		}
		pop, err := priv.GenerateBlsPop()
		if err != nil {
			panic("Failed to generate bls proof of possession" + err.Error())
		}
		_ = RegisterOperator(
			client,
			operatorAccount,
			avsAddress.String(),
			quorumNumbers,
			signature.Auth.Signature().Bytes(),
			signature.PubKey().Bytes(),
			pop.Bytes(),
		)
	}

	// Get OperatorId
	var privKey crypto.BlsPrivateKey
	privKey.FromBytes(config.BlsPrivateKey)
	operatorId := privKey.Inner.PublicKey().Marshal()

	aggClient, err := NewAggregatorRpcClient(config.AggregatorIpPortAddr)
	if err != nil {
		return nil, fmt.Errorf("can not create new aggregator Rpc Client: %v", err)
	}

	// return Operator
	operator := Operator{
		logger:        logger,
		account:       operatorAccount,
		operatorId:    operatorId,
		avsAddress:    avsAddress,
		BlsPrivateKey: blsPriv,
		AggRpcClient:  *aggClient,
		network:       networkConfig,
		TaskQueue:     make(chan Task, 100),
	}
	return &operator, nil
}

func InitQuorum(
	networkConfig solana.NetworkConfig,
	config OperatorConfig,
	accountConfig SolanaAccountConfig,
	maxOperatorCount uint32,
	minimumStake big.Int,
) error {
	client, err := solana.NewClient(networkConfig)
	if err != nil {
		panic("Failed to create client:" + err.Error())
	}

	accAddress := solana.AccountAddress{}
	err = accAddress.ParseStringRelaxed("0x603053371d0eec6befaf41489f506b7b3e8e31dbca3d9b9c5cb92bb308dc2eec")
	if err != nil {
		panic("Failed to parse account address " + err.Error())
	}

	operatorAccount, err := SignerFromConfig(accountConfig.configPath, accountConfig.profile)
	if err != nil {
		panic("Failed to create operator account:" + err.Error())
	}

	maxOperatorCountBz, err := bcs.SerializeU32(maxOperatorCount)
	if err != nil {
		return fmt.Errorf("failed to serialize maxOperatorCount: %s", err)
	}

	minimumStakeBz, err := bcs.SerializeU128(minimumStake)
	if err != nil {
		return fmt.Errorf("failed to serialize minimumStake: %s", err)
	}

	metadataAddr := GetMetadata(client).Inner

	strategiesSerializer := &bcs.Serializer{}
	bcs.SerializeSequence([]solana.AccountAddress{metadataAddr}, strategiesSerializer)

	multiplier := new(big.Int)
	multiplier.SetString("10000000", 10)

	multipliersSerializer := &bcs.Serializer{}
	bcs.SerializeSequence([]U128Struct{{
		Value: multiplier,
	}}, multipliersSerializer)

	// Get operator Status
	avsAddress := solana.AccountAddress{}
	if err := avsAddress.ParseStringRelaxed(config.AvsAddress); err != nil {
		panic("Failed to parse avsAddress:" + err.Error())
	}

	payload := solana.EntryFunction{
		Module: solana.ModuleId{
			Address: avsAddress,
			Name:    "registry_coordinator",
		},
		Function: "create_quorum",
		ArgTypes: []solana.TypeTag{},
		Args: [][]byte{
			maxOperatorCountBz, minimumStakeBz, strategiesSerializer.ToBytes(), multipliersSerializer.ToBytes(),
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
	fmt.Printf("Submit register operator for %s\n", operatorAccount.AccountAddress())

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
		panic("Failed to create quorum")
	}
	return nil
}
func QuorumCount(client *solana.Client, contract solana.AccountAddress) uint8 {
	payload := &solana.ViewPayload{
		Module: solana.ModuleId{
			Address: contract,
			Name:    "registry_coordinator",
		},
		Function: "quorum_count",
		ArgTypes: []solana.TypeTag{},
		Args:     [][]byte{},
	}

	vals, err := client.View(payload)
	if err != nil {
		panic("Could not get quorum count:" + err.Error())
	}
	count := vals[0].(float64)
	return uint8(count)
}

func IsOperatorRegistered(client *solana.Client, contract solana.AccountAddress, operator_addr string) bool {
	account := solana.AccountAddress{}
	err := account.ParseStringRelaxed(operator_addr)
	if err != nil {
		panic("Could not ParseStringRelaxed:" + err.Error())
	}
	operator, err := bcs.Serialize(&account)
	if err != nil {
		panic("Could not serialize operator address:" + err.Error())
	}
	payload := &solana.ViewPayload{
		Module: solana.ModuleId{
			Address: contract,
			Name:    "registry_coordinator",
		},
		Function: "get_operator_status",
		ArgTypes: []solana.TypeTag{},
		Args: [][]byte{
			operator,
		},
	}

	vals, err := client.View(payload)
	if err != nil {
		panic("Could not get operator status:" + err.Error())
	}
	status := vals[0].(float64)
	return status != 0
}

func RegisterOperator(
	client *solana.Client,
	operatorAccount *solana.Account,
	contractAddr string,
	quorumNumbers uint8,
	signature []byte,
	pubkey []byte,
	proofPossession []byte,
) error {
	contract := solana.AccountAddress{}
	err := contract.ParseStringRelaxed(contractAddr)
	if err != nil {
		panic("Failed to parse address:" + err.Error())
	}
	quorumSerializer := &bcs.Serializer{}
	bcs.SerializeSequence([]U8Struct{
		{
			Value: quorumNumbers,
		},
	}, quorumSerializer)

	sig, err := bcs.SerializeBytes(signature)
	if err != nil {
		panic("Failed to bcs serialize signature:" + err.Error())
	}
	pk, err := bcs.SerializeBytes(pubkey)
	if err != nil {
		panic("Failed to bcs serialize pubkey:" + err.Error())
	}
	pop, err := bcs.SerializeBytes(proofPossession)
	if err != nil {
		panic("Failed to bcs serialize proof of possession:" + err.Error())
	}
	payload := solana.EntryFunction{
		Module: solana.ModuleId{
			Address: contract,
			Name:    "registry_coordinator",
		},
		Function: "registor_operator",
		ArgTypes: []solana.TypeTag{},
		Args: [][]byte{
			quorumSerializer.ToBytes(), sig, pk, pop,
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
	fmt.Printf("Submit register operator for %s\n", operatorAccount.AccountAddress())

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
		panic("Failed to register operator")
	}
	return nil
}
