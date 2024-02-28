## Blockchain Oracle Rollup

The Blockchain Oracle Rollup is an oracle n/w built as a rollup. It's job is to provide the latest block hash, state roots and other important fields of different blockchains.
Currently, we only support the Ethereum blockchain but it can be extended to chains like Solana, BSC, Polygon etc. 

We have listeners that listen for blocks on each chain and submit the latest chain data to the rollup. This rollup utilizes the Astria Shared Sequencer n/w rather than building its own sequencer
to make use of the shared sequencer's security, scalability and to ride on top of the shared sequencers DA guarantees rather than implementing its own DA related things.


## Observations

1. In `ExecuteBlock` step, the conductor returns the tx set back to the rollup. The Rollup team cannot likely predict the order of the txs. Given this, Would someone want to use a SS if they wanted predictable ordering of blocks?
2. 