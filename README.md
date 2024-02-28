## Blockchain Oracle Rollup

The Blockchain Oracle Rollup is an oracle n/w built as a rollup. It's job is to provide the latest block hash, state roots and other important fields of different blockchains.
Currently, we only support the Ethereum blockchain but it can be extended to chains like Solana, BSC, Polygon etc. 

We have listeners that listen for blocks on each chain and submit the latest chain data to the rollup. This rollup utilizes the Astria Shared Sequencer n/w rather than building its own sequencer
to make use of the shared sequencer's security, scalability and to ride on top of the shared sequencers DA guarantees rather than implementing its own DA related things.


## Observations

1. In `ExecuteBlock` step, the conductor returns the tx set back to the rollup. The Rollup team cannot likely predict the order of the txs. Given this, Would someone want to use a SS if they wanted predictable ordering of blocks?
2. The Rollup team is responsible for maintaining their own blockchain. It is not SS job to know the latest soft confirmed block and the latest finalized block.
3. The Execution API is an interesting place at which we can implement rollup commitments and verifications of sorts i guess. We could have an extra method in the Execution API which allows the Rollup to specify
the kind of block they want. When we receive the block via the `ExecuteBlock` step, we could verify according to the commitment. 
4. What happens if the conductor is connected to an out of sync SS validator? They wouldn't get the latest actual block. Either the SS team has to make sure that this 
never happens by not sending blocks to the conductor if its not in sync or the Rollup team has to make sure that they are connected to a SS validator that is in sync. This could be an additional check
the rollup team would have to make. The conductor could have a fallback mechanism too where it tries to connect to another SS validator if the current one is out of sync.