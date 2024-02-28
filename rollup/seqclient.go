package rollup

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"fmt"

	astriaPb "buf.build/gen/go/astria/astria/protocolbuffers/go/astria/sequencer/v1alpha1"
	client "github.com/astriaorg/go-sequencer-client/client"
	tendermintPb "github.com/cometbft/cometbft/rpc/core/types"

	log "github.com/sirupsen/logrus"
)

// SequencerClient is a client for interacting with the sequencer.
type SequencerClient struct {
	c        *client.Client
	signer   *client.Signer
	nonce    uint32
	rollupId []byte
}

// NewSequencerClient creates a new SequencerClient.
func NewSequencerClient(sequencerAddr string, rollupId []byte, private ed25519.PrivateKey) *SequencerClient {
	log.Debug("creating new sequencer client")
	signer := client.NewSigner(private)

	// default tendermint RPC endpoint
	c, err := client.NewClient(sequencerAddr)
	if err != nil {
		panic(err)
	}

	return &SequencerClient{
		c:        c,
		signer:   signer,
		rollupId: rollupId,
	}
}

// broadcastTxSync broadcasts a transaction synchronously.
func (sc *SequencerClient) broadcastTxSync(tx *astriaPb.SignedTransaction) (*tendermintPb.ResultBroadcastTx, error) {
	log.Debug("broadcasting tx")
	return sc.c.BroadcastTxSync(context.Background(), tx)
}

// SequenceTx sends a tx to the astria shared sequencer..
func (sc *SequencerClient) SequenceTx(tx Transaction) (*tendermintPb.ResultBroadcastTx, error) {
	log.Debug("sending message")
	data, err := json.Marshal(tx)
	if err != nil {
		return nil, err
	}

	log.WithFields(log.Fields{
		"ethBlockData": tx.FinalizedEthBlockData,
	}).Debug("submitting tx to sequencer.")

	currentNonce := sc.nonce
	resp := &tendermintPb.ResultBroadcastTx{}
	// retry till we get it right if there is an issue with the nonce
	for {
		unsigned := &astriaPb.UnsignedTransaction{
			Nonce: currentNonce,
			Actions: []*astriaPb.Action{
				{
					Value: &astriaPb.Action_SequenceAction{
						SequenceAction: &astriaPb.SequenceAction{
							RollupId: sc.rollupId,
							Data:     data,
						},
					},
				},
			},
		}

		signed, err := sc.signer.SignTransaction(unsigned)
		if err != nil {
			panic(err)
		}

		resp, err = sc.broadcastTxSync(signed)
		if err != nil {
			return nil, err
		}
		if resp.Code == 4 {
			// fetch new nonce
			newNonce, err := sc.c.GetNonce(context.Background(), sc.signer.Address())
			if err != nil {
				return nil, err
			}
			currentNonce = newNonce
			sc.nonce = currentNonce
			continue
		} else if resp.Code != 0 {
			return nil, fmt.Errorf("unexpected error code: %d", resp.Code)
		}
		// we are good since the request successfully executed as of now
		break
	}

	return resp, nil
}
