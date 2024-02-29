package rollup

type Config struct {
	EthereumRpc  string `env:"ETHEREUM_RPC, default=http://localhost:8545"`
	SequencerRpc string `env:"SEQUENCER_RPC, default=http://localhost:26657"`
	ConductorRpc string `env:"CONDUCTOR_RPC, default=http://localhost:50051"`
	RollupName   string `env:"ROLLUP_NAME, default=blockchain-oracle-rollup"`
	RollupId     string `env:"ROLLUP_ID, default=blockchain-oracle-rollup"`
	SeqPrivate   string `env:"SEQUENCER_PRIVATE, default="`
	RESTApiPort  string `env:"RESTAPI_PORT, default=:8080"`
}
