# Sentra Layer AVS on Sonic SVM


## Requiments

- Git
- Golang : version > 1.22
- Already registered as an Operator in Sonic SVM Restaking core

## Minimum hardware requirements:

| Component     | Specification     |
|---------------|-------------------|
| **CPU**       | 16 cores          |
| **Memory**    | 32 GB RAM         |
| **Bandwidth** | 1 Gbps            |
| **Storage**   | 256 GB disk space |

## Steps to running an Operator

Step 1: Clone the repository

```bash
git clone https://github.com/A2Dsquad/sonic-svm-avs
cd avs
```

Step 2: Build the Operator binary

```bash
go build -mod=readonly -o ./build/avs ./cmd
```

Step 3: Configure the Operator

```bash
./build/avs operator config {avs address} {Aggregator rpc}
```

This will create a `operator-config.json` file in `config` folder. You can see the example in `config/example.json`.

Step 4: Start Operator
```bash
./build/avs operator start
```

Your operator is now up and running!

## AVS consensus

For each AVS, all of its operators have to interface through a middleware layer to relay/receive data on chain (receive tasks and return their result). In this layer, we have a configurable value `quorum` which is a percentage value indicate the threshold of stake amount needed consensus. That is, for every task emitted, each of the AVS's operators should return a signed message containing result to the middleware layer, if the amount of stake delegated to the operators having submitted reaches the `quorum` then we confirm a result for that task, in other word, the AVS has reached consensus for that task. 

![alt text](consensus.png)

## Structure

### Operator

#### Commands

