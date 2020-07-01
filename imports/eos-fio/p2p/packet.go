package p2p

import (
	"github.com/fioprotocol/fio-go/imports/eos-fio"
)

type Envelope struct {
	Sender   *Peer
	Receiver *Peer
	Packet   *fos.Packet `json:"envelope"`
}

func NewEnvelope(sender *Peer, receiver *Peer, packet *fos.Packet) *Envelope {
	return &Envelope{
		Sender:   sender,
		Receiver: receiver,
		Packet:   packet,
	}
}
