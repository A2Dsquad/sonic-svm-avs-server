package operator

import (
	"math/big"
	"net/rpc"

	solana "github.com/solana-labs/solana-go-sdk"
	"github.com/solana-labs/solana-go-sdk/bcs"
	"go.uber.org/zap"

	"github.com/Layr-Labs/eigensdk-go/crypto/bls"
)

type Config struct {
	CmcApi string `json:"cmc_api"`
}

type Operator struct {
	logger  *zap.Logger
	account *solana.Account
	// TODO: change this to solana-sdk fork
	operatorId   []byte
	avsAddress   solana.AccountAddress
	BlsPrivateKey        []byte
	AggRpcClient AggregatorRpcClient
	network      solana.NetworkConfig
	TaskQueue    chan Task
}

type Task struct {
	Id   uint64
	Task map[string]interface{}
}

type AggregatorRpcClient struct {
	rpcClient            *rpc.Client
	aggregatorIpPortAddr string
}

type OperatorConfig struct {
	BlsPrivateKey        []byte
	AvsAddress           string
	AggregatorIpPortAddr string
	// OperatorId           eigentypes.OperatorId
}

type AVSTask struct {
	// TODO
	task_created_timestamp uint64
	responded              bool
	respond_fee_token      uint64
	respond_fee_limit      uint64
}

type BlsConfig struct {
	KeyPair *bls.KeyPair
}

type MetadataStr struct {
	Inner string
}

type Metadata struct {
	Inner solana.AccountAddress
}

type U128Struct struct {
	Value *big.Int `json:"value"`
}

func (u *U128Struct) MarshalBCS(ser *bcs.Serializer) {
	ser.U128(*u.Value)
}

type U8Struct struct {
	Value uint8
}

func (u *U8Struct) MarshalBCS(ser *bcs.Serializer) {
	ser.U8(u.Value)
}

type U64Struct struct {
	Value uint64
}

func (u *U64Struct) MarshalBCS(ser *bcs.Serializer) {
	ser.U64(u.Value)
}
