package operator

import (
	"avs/aggregator"
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	solana "github.com/solana-labs/solana-go-sdk"
	"github.com/solana-labs/solana-go-sdk/bcs"
	"github.com/solana-labs/solana-go-sdk/crypto"
	"go.uber.org/zap"
)

const (
	MaxRetries                        = 100
	RetryInterval                     = 2 * time.Second
	BlockInterval              uint64 = 1000
	PollLatestBatchInterval           = 5 * time.Second
	RemoveBatchFromSetInterval        = 5 * time.Minute
)

func (op *Operator) Start(ctx context.Context) error {
	op.logger.Info("Starting operator...")

	ctx, cancel := context.WithCancel(ctx)

	defer cancel()
	// Fetching tasks
	go func() {
		op.logger.Info("Fetching tasks process started...")
		err := op.FetchTasks(ctx)
		if err != nil {
			op.logger.Fatal("Error listening for tasks", zap.Any("err", err))
		}
	}()

	go func() {
		op.logger.Info("Respond tasks process started...")
		err := op.RespondTask(ctx)
		if err != nil {
			op.logger.Fatal("Error listening for tasks", zap.Any("err", err))
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for a signal to shutdown
	sig := <-sigChan
	op.logger.Info("Received signal, shutting down...", zap.Any("signal", sig))

	cancel()
	return nil
}

func (op *Operator) FetchTasks(ctx context.Context) error {
	client, err := solana.NewClient(op.network)
	if err != nil {
		return fmt.Errorf("failed to create solana client: %v", err)
	}

	var taskCount uint64
	// looping
	for {
		previousTaskCount := taskCount
		newTaskCount, err := LatestTaskCount(client, op.avsAddress)
		if err != nil {
			op.logger.Warn("Failed to subscribe to new tasks", zap.Any("err", err))
			time.Sleep(RetryInterval)
			continue
		}
		taskCount = newTaskCount

		if taskCount > previousTaskCount {
			err := op.QueueTask(ctx, client, previousTaskCount, taskCount)
			if err != nil {
				return fmt.Errorf("error queuing task: %v", err)
			}
		}
	}
}

func (op *Operator) RespondTask(ctx context.Context) error {

	client, err := solana.NewClient(op.network)
	if err != nil {
		return fmt.Errorf("failed to create solana client: %v", err)
	}
	// TODO
	for task := range op.TaskQueue {
		denom := task.Task["data_request"].(string)
		upperDenom := strings.ToUpper(denom)
		taskId := task.Id

		price := big.NewInt(int64(getCMCPrice(upperDenom) * 1000000))

		msghHash, err := GetMsgHash(client, op.avsAddress, taskId, *price)
		if err != nil {
			return fmt.Errorf("failed to GetMsgHash: %v", err)
		}

		trimmedMsgHash := strings.TrimPrefix(msghHash, "0x")
		bytesMsgHash, err := hex.DecodeString(trimmedMsgHash)
		if err != nil {
			return fmt.Errorf("failed to decode hex to string: %v", err)
		}

		var priv crypto.BlsPrivateKey
		err = priv.FromBytes(op.BlsPrivateKey)
		if err != nil {
			panic("Failed to create bls priv key" + err.Error())
		}
		signature, err := priv.Sign(bytesMsgHash)
		if err != nil {
			panic("Failed to create signature" + err.Error())
		}

		// pubKey, err := priv.GeneratePubkey()
		// if err != nil {
		// 	panic("Failed to generate pubkey from privkey" + err.Error())
		// }

		op.AggRpcClient.SendSignedTaskResponseToAggregator(aggregator.SignedTaskResponse{
			TaskId:    taskId,
			Pubkey:    signature.Auth.PublicKey().Bytes(),
			Signature: signature.Auth.Signature().Bytes(),
			Response:  price,
		})
	}
	return nil
}

func GetMsgHash(client *solana.Client, contract solana.AccountAddress, taskId uint64, response big.Int) (string, error) {
	taskIdBcs, err := bcs.SerializeU64(taskId)
	if err != nil {
		return "", fmt.Errorf("can not SerializeU64: %v", err)
	}

	responseBcs, err := bcs.SerializeU128(response)
	if err != nil {
		return "", fmt.Errorf("can not SerializeU128: %v", err)
	}
	payload := &solana.ViewPayload{
		Module: solana.ModuleId{
			Address: contract,
			Name:    "service_manager",
		},
		Function: "get_msg_hash",
		ArgTypes: []solana.TypeTag{},
		Args: [][]byte{
			taskIdBcs, responseBcs,
		},
	}
	vals, err := client.View(payload)
	if err != nil {
		return "", fmt.Errorf("can not get msg hash: %v", err)
	}
	task := vals[0].(string)
	return task, nil
}

func (op *Operator) QueueTask(ctx context.Context, client *solana.Client, start uint64, end uint64) error {
	for i := start + 1; i <= end; i++ {
		task, err := LoadTaskById(client, op.avsAddress, i)
		if err != nil {
			return fmt.Errorf("error loading task: %v", err)
		}
		responded := task["responded"].(bool)
		if responded {
			continue
		}
		op.logger.Info("Loaded new task with id:", zap.Any("task id", i))
		op.TaskQueue <- Task{
			Id:   i,
			Task: task,
		}
		op.logger.Info("Queued new task with id:", zap.Any("task id", i))
	}

	return nil
}

func LoadTaskById(client *solana.Client, contract solana.AccountAddress, taskId uint64) (map[string]interface{}, error) {
	taskIdBcs, err := bcs.SerializeU64(taskId)
	if err != nil {
		return nil, fmt.Errorf("can not SerializeU64: %v", err)
	}
	payload := &solana.ViewPayload{
		Module: solana.ModuleId{
			Address: contract,
			Name:    "service_manager",
		},
		Function: "task_by_id",
		ArgTypes: []solana.TypeTag{},
		Args: [][]byte{
			taskIdBcs,
		},
	}
	vals, err := client.View(payload)
	if err != nil {
		return nil, fmt.Errorf("can not get task count: %v", err)
	}
	task := vals[0].(map[string]interface{})
	return task, nil
}

func LatestTaskCount(client *solana.Client, contract solana.AccountAddress) (uint64, error) {
	payload := &solana.ViewPayload{
		Module: solana.ModuleId{
			Address: contract,
			Name:    "service_manager",
		},
		Function: "task_count",
		ArgTypes: []solana.TypeTag{},
		Args:     [][]byte{},
	}

	vals, err := client.View(payload)
	if err != nil {
		return 0, fmt.Errorf("can not get task count: %v", err)
	}
	countStr := vals[0].(string)

	count, err := strconv.ParseUint(countStr, 10, 64) // base 10 and 64-bit size
	if err != nil {
		return 0, fmt.Errorf("error parsing task count: %s", err)
	}
	return uint64(count), nil
}
