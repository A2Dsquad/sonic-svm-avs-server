# Stake-Pool Python Bindings

Preliminary Python bindings to interact with the restaking program, enabling
simple stake delegation bots.

## To do

* More reference bot implementations
* Add bindings for all stake pool instructions, see `TODO`s in `stake_pool/instructions.py`
* Finish bindings for vote and stake program
* Upstream vote and stake program bindings to https://github.com/michaelhly/solana-py

## Development

### Environment Setup

1. Ensure that Python 3 is installed with `venv`: https://www.python.org/downloads/
2. (Optional, but highly recommended) Setup and activate a virtual environment:

```
$ python3 -m venv venv
$ source venv/bin/activate
```

3. Install build and dev requirements

```
$ pip install -r requirements.txt
$ pip install -r optional-requirements.txt
```

4. Install the Solana tool suite: https://docs.solana.com/cli/install-solana-cli-tools

### Test

Testing through `pytest`:

```
$ python3 -m pytest
```

### Formatting

```
$ flake8 bot spl_token stake stake_pool system tests vote
```

### Type Checker

```
$ mypy bot stake stake_pool tests vote spl_token system
```

## Delegation Bots

The `./bot` directory contains sample stake pool delegation bot implementations:

* `rebalance`: simple bot to make the amount delegated to each validator
uniform, while also maintaining some SOL in the reserve if desired. Can be run
with the stake pool address, staker keypair, and SOL to leave in the reserve:

```
$ python3 bot/rebalance.py <restaking pool address> staker.json 10.5
```
