package v2

import (
	"fmt"

	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/state"
	"github.com/tendermint/tendermint/types"
)

type iIO interface {
	sendBlockRequest(peerID p2p.ID, height int64) error
	sendBlockToPeer(block *types.Block, peerID p2p.ID) error
	sendBlockNotFound(height int64, peerID p2p.ID) error
	sendStatusResponse(height int64, peerID p2p.ID) error

	broadcastStatusRequest(height int64)

	switchToConsensus(state state.State, blocksSynced int)
}

type switchIO struct {
	sw *p2p.Switch
}

func newSwitchIo(sw *p2p.Switch) *switchIO {
	return &switchIO{
		sw: sw,
	}
}

const (
	// BlockchainChannel is a channel for blocks and status updates (`BlockStore` height)
	BlockchainChannel = byte(0x40)
)

type consensusReactor interface {
	// for when we switch from blockchain reactor and fast sync to
	// the consensus machine
	SwitchToConsensus(state.State, int)
}

func (sio *switchIO) sendBlockRequest(peerID p2p.ID, height int64) error {
	peer := sio.sw.Peers().Get(peerID)
	if peer == nil {
		return fmt.Errorf("peer not found")
	}

	msgBytes := cdc.MustMarshalBinaryBare(&bcBlockRequestMessage{Height: height})
	queued := peer.TrySend(BlockchainChannel, msgBytes)
	if !queued {
		return fmt.Errorf("send queue full")
	}
	return nil
}

func (sio *switchIO) sendStatusResponse(height int64, peerID p2p.ID) error {
	peer := sio.sw.Peers().Get(peerID)
	if peer == nil {
		return fmt.Errorf("peer not found")
	}
	msgBytes := cdc.MustMarshalBinaryBare(&bcStatusResponseMessage{Height: height})

	if queued := peer.TrySend(BlockchainChannel, msgBytes); !queued {
		return fmt.Errorf("peer queue full")
	}

	return nil
}

func (sio *switchIO) sendBlockToPeer(block *types.Block, peerID p2p.ID) error {
	peer := sio.sw.Peers().Get(peerID)
	if peer == nil {
		return fmt.Errorf("peer not found")
	}
	if block == nil {
		return fmt.Errorf("nil block")
	}
	msgBytes := cdc.MustMarshalBinaryBare(&bcBlockResponseMessage{Block: block})
	if queued := peer.TrySend(BlockchainChannel, msgBytes); !queued {
		return fmt.Errorf("peer queue full")
	}

	return nil
}

func (sio *switchIO) sendBlockNotFound(height int64, peerID p2p.ID) error {
	peer := sio.sw.Peers().Get(peerID)
	if peer == nil {
		return fmt.Errorf("peer not found")
	}
	msgBytes := cdc.MustMarshalBinaryBare(&bcNoBlockResponseMessage{Height: height})
	if queued := peer.TrySend(BlockchainChannel, msgBytes); !queued {
		return fmt.Errorf("peer queue full")
	}

	return nil
}

func (sio *switchIO) switchToConsensus(state state.State, blocksSynced int) {
	conR, ok := sio.sw.Reactor("CONSENSUS").(consensusReactor)
	if ok {
		conR.SwitchToConsensus(state, blocksSynced)
	}
}

func (sio *switchIO) broadcastStatusRequest(height int64) {
	msgBytes := cdc.MustMarshalBinaryBare(&bcStatusRequestMessage{height})
	// XXX: maybe we should use an io specific peer list here
	sio.sw.Broadcast(BlockchainChannel, msgBytes)
}