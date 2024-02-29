package rollup

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/attestantio/go-eth2-client/spec/deneb"
	"github.com/sirupsen/logrus"
)

type BeaconBlockResponse struct {
	Version             string `json:"version"`
	ExecutionOptimistic bool   `json:"execution_optimistic"`
	Finalized           bool   `json:"finalized"`
	Data                struct {
		Message deneb.BeaconBlock `json:"message"`
	}
}

// json marshal and unmarshal
func (b *BeaconBlockResponse) Marshal() ([]byte, error) {
	return json.Marshal(b)
}

func (b *BeaconBlockResponse) Unmarshal(data []byte) error {
	return json.Unmarshal(data, b)
}

type EthBlockData struct {
	BlockHash     string `json:"block_hash"`
	StateRoot     string `json:"state_root"`
	ParentRoot    string `json:"parent_root"`
	Slot          uint64 `json:"slot"`
	ProposerIndex uint64 `json:"proposer_index"`
}

type ChainListeners struct {
	EthereumRpc    string `env:"ETHEREUM_RPC, default=http://localhost:8545"`
	EthDataSink    chan EthBlockData
	ShutdownSignal chan bool
}

func NewChainListeners(EthereumRpc string, ethDataSink chan EthBlockData, shutdownSignal chan bool) *ChainListeners {
	return &ChainListeners{
		EthereumRpc:    EthereumRpc,
		EthDataSink:    ethDataSink,
		ShutdownSignal: shutdownSignal,
	}
}

func (cl *ChainListeners) Run() {
	ticker := time.Tick(15 * time.Second)
	for {
		select {
		case <-ticker:
			logrus.Info("Making request to Ethereum RPC")
			// template to make a http GET request
			resp, err := http.Get(fmt.Sprintf("%s/eth/v2/beacon/blocks/head", cl.EthereumRpc))
			if err != nil {
				logrus.Error("Error making request to Ethereum RPC: ", err)
				continue
			}
			// print the response body
			beaconBlockRes := BeaconBlockResponse{}
			// unmarshall response body to beaconBlockRes
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				logrus.Error("Error reading response body: ", err)
				continue
			}
			err = beaconBlockRes.Unmarshal(body)
			if err != nil {
				logrus.Error("Error unmarshalling response body: ", err)
				continue
			}

			ethBlockData := EthBlockData{
				ParentRoot:    beaconBlockRes.Data.Message.ParentRoot.String(),
				BlockHash:     hex.EncodeToString(beaconBlockRes.Data.Message.Body.ETH1Data.BlockHash),
				StateRoot:     beaconBlockRes.Data.Message.StateRoot.String(),
				Slot:          uint64(beaconBlockRes.Data.Message.Slot),
				ProposerIndex: uint64(beaconBlockRes.Data.Message.ProposerIndex),
			}

			cl.EthDataSink <- ethBlockData

			fmt.Printf("ethBlockData is %+v\n", ethBlockData)

		case <-cl.ShutdownSignal:
			logrus.Debugf("Shutting ethereum chain listener down!")
			break
		}
	}
}
