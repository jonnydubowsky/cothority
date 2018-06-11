Navigation: [DEDIS](https://github.com/dedis/doc/tree/master/README.md) ::
[Cothority](https://github.com/dedis/cothority/tree/master/README.md) ::
[Building Blocks](https://github.com/dedis/cothority/tree/master/BuildingBlocks.md) ::
[OmniLedger](README.md) ::
Contracts

# Contracts and Instances

A contract in omniledger is similar to a smart contract in Ethereum, except that
it is pre-compiled in the code and all nodes need to have the same version of
the contract available.

Every instance in omniledger is stored with the following information in the
global state:
- `InstanceID` is a globally unique identifier of that instance, composed of:
  - `DarcID`, defining the access rights to that instance
  - `Nonce`, which is randomly chosen. The special `Nonce` of `0` indicates
the Darc responsible for all the instances starting with the same `DarcID`
- `ContractID` points to the contract that will be called if that instance
receives an instruction from the client
- `Data` is interpreted by the contract and can change over time

The special `InstanceID` with 64 x 0 bytes is the genesis configuration
pointer that has as the data the `DarcID` of the genesis Darc.

## Interaction between Instructions and Instances

Every instruction sent by a client indicates the `InstanceID` it is sent to.
Omniledger will use this `InstanceID` to look up the responsible contract for
this instance and then send the instruction to that contract, giving it the data
part of the instance as an argument. A client can call an instance with the
following three basic instructions:

- `Spawn` - will ask the instance to create a new instance. The client indicates the
requested new contract-type and the arguments. Currently only `Darc` instances can
spawn new instances.
- `Invoke` - sends a method and its arguments to the instance
- `Delete` - requests to delete that instance

Every instruction sent by the client to an instance will first be verified by
omniledger whether it can be authenticated using the `Darc` defined by the first
part of the `InstanceID`. Only if this authentication succeeds will the
corresponding contract be called.

# Existing Contracts

In omniledger, there are two contracts that can only be instantiated once in the
whole system:
- `GenesisDarc`, which has the `InstanceID` of 64 x 0x00
- `Configuration`, which defines 

## Darc Contract

The most basic contract in omniledger is the `Darc` contract that defines the
access rules for all clients. When creating a new omniledger blockchain, a
genesis Darc instance is created, which indicates what instructions need which
signatures to be accepted.

### Spawn

When the client sends a spawn instruction to a Darc contract, he asks the instance
to create a new instance with the given ContractID, which can be different from
the Darc itself. The client must be able to authenticate against a
`Spawn_contractid` rule defined in the Darc instance.

### Invoke

The only method that a client can invoke on a Darc instance is `Evolve`, which
asks omniledger to store a new version of the Darc in the global state.

### Delete

When a Darc instance receives a `Delete` instruction, it will be removed from the
global state.

*TODO* - discuss what happens with the instances that depend on that darc.

Examples of contracts and some of their methods are:

- Darc:
  - create a new Darc
	- update a darc
- OmniLedger Configuration
  - create new configuration
  - Add or remove nodes
  - Change the block interval time
- Onchain-secrets write request:
  - create a write request
- Onchain-secrets read request:
  - create a read request
- Onchain-secrets reencryption request:
  - create a reencryption request
- Evoting:
  - Creating a new election
  - Casting a vote
  - Requesting mix
  - Requesting decryption
- PoP:
  - Create a new party
  - Adding attendees
	- Finalizing the party
- PoPCoin:
  - Creating a popcoin source
- PoPCoinAccount:
  - Creating an account
	- Transfer coins from one account to another
