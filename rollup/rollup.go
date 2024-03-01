package rollup

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"time"

	astriaPb "buf.build/gen/go/astria/execution-apis/protocolbuffers/go/astria/execution/v1alpha2"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// we now need to define a transaction and a block

type Transaction struct {
	FinalizedEthBlockData EthBlockData `json:"ethBlockData"`
}

func HashTxs(txs []Transaction) ([32]byte, error) {
	txBytes := [][]byte{}
	for _, tx := range txs {
		if bs, err := json.Marshal(tx); err != nil {
			return [32]byte{}, err
		} else {
			txBytes = append(txBytes, bs)
		}
	}

	hash := sha256.Sum256(bytes.Join(txBytes, []byte{}))

	return hash, nil
}

type Block struct {
	ParentHash [32]byte
	Hash       [32]byte
	Height     uint32
	Timestamp  time.Time
	// ideally each tx will have an individual chains finalized data. like tx1 = eth finalized data, tx2 = solana finalized data etc
	// if a block has multiple txs with the same chain finalized data, we should consider the last tx as the chains finalized data for the sake of this example
	Txs []Transaction
}

func NewBlock(parentHash []byte, height uint32, txs []Transaction, timestamp time.Time) Block {
	txHash, err := HashTxs(txs)
	if err != nil {
		panic(err)
	}

	return Block{
		ParentHash: [32]byte(parentHash),
		Hash:       txHash,
		Height:     height,
		Txs:        txs,
		Timestamp:  timestamp,
	}
}

func (b *Block) ToPb() (*astriaPb.Block, error) {
	return &astriaPb.Block{
		Number:          b.Height,
		Hash:            b.Hash[:],
		ParentBlockHash: b.ParentHash[:],
		Timestamp:       timestamppb.New(b.Timestamp),
	}, nil
}

// GenesisBlock creates the genesis block.
func GenesisBlock() Block {
	genesisTx := Transaction{
		EthBlockData{
			BlockHash:     "",
			StateRoot:     "",
			ParentRoot:    "",
			Slot:          0,
			ProposerIndex: 0,
		},
	}

	genesisHash, err := HashTxs([]Transaction{genesisTx})
	if err != nil {
		logrus.Errorf("error hashing genesis tx: %s\n", err)
		panic(err)
	}

	return Block{
		ParentHash: [32]byte{0x00000000},
		Hash:       genesisHash,
		Height:     0,
		Timestamp:  time.Now(),
		Txs: []Transaction{
			genesisTx,
		},
	}
}

type Rollup struct {
	Blocks []Block
	soft   uint32
	firm   uint32
}

func NewRollup() Rollup {
	return Rollup{
		Blocks: []Block{GenesisBlock()},
		soft:   0,
		firm:   0,
	}
}

func (r *Rollup) GetSingleBlock(height uint32) (*Block, error) {
	if height >= uint32(len(r.Blocks)) {
		return nil, errors.New("block not found")
	}
	return &r.Blocks[height], nil
}

func (r *Rollup) GetSoftBlock() *Block {
	return &r.Blocks[r.soft]
}

func (r *Rollup) GetFirmBlock() *Block {
	return &r.Blocks[r.firm]
}

func (r *Rollup) GetLatestBlock() *Block {
	return &r.Blocks[len(r.Blocks)-1]
}

func (r *Rollup) Height() uint32 {
	return uint32(len(r.Blocks))
}
