package p2p

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/pkg/errors"

	"go.uber.org/zap"

	"go.uber.org/zap/zapcore"

	"runtime"

	"bufio"

	fos "github.com/fioprotocol/fio-go/imports/eos-fio"
	"github.com/fioprotocol/fio-go/imports/eos-fio/fecc"
)

type Peer struct {
	Address                string
	Name                   string
	agent                  string
	NodeID                 []byte
	connection             net.Conn
	reader                 io.Reader
	listener               bool
	handshakeInfo          *HandshakeInfo
	connectionTimeout      time.Duration
	handshakeTimeout       time.Duration
	cancelHandshakeTimeout chan bool
}

// MarshalLogObject calls the underlying function from zap.
func (p Peer) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("name", p.Name)
	enc.AddString("address", p.Address)
	enc.AddString("agent", p.agent)
	return enc.AddObject("handshakeInfo", p.handshakeInfo)
}

type HandshakeInfo struct {
	ChainID                  fos.Checksum256
	HeadBlockNum             uint32
	HeadBlockID              fos.Checksum256
	HeadBlockTime            time.Time
	LastIrreversibleBlockNum uint32
	LastIrreversibleBlockID  fos.Checksum256
}

func (h *HandshakeInfo) String() string {
	return fmt.Sprintf("Handshake Info: HeadBlockNum [%d], LastIrreversibleBlockNum [%d]", h.HeadBlockNum, h.LastIrreversibleBlockNum)
}

// MarshalLogObject calls the underlying function from zap.
func (h HandshakeInfo) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("chainID", h.ChainID.String())
	enc.AddUint32("headBlockNum", h.HeadBlockNum)
	enc.AddString("headBlockID", h.HeadBlockID.String())
	enc.AddTime("headBlockTime", h.HeadBlockTime)
	enc.AddUint32("lastIrreversibleBlockNum", h.LastIrreversibleBlockNum)
	enc.AddString("lastIrreversibleBlockID", h.LastIrreversibleBlockID.String())
	return nil
}

func (p *Peer) SetHandshakeTimeout(timeout time.Duration) {
	p.handshakeTimeout = timeout
}

func (p *Peer) SetConnectionTimeout(timeout time.Duration) {
	p.connectionTimeout = timeout
}

func newPeer(address string, agent string, listener bool, handshakeInfo *HandshakeInfo) *Peer {

	return &Peer{
		Address:                address,
		agent:                  agent,
		listener:               listener,
		handshakeInfo:          handshakeInfo,
		cancelHandshakeTimeout: make(chan bool),
	}
}

func NewIncommingPeer(address string, agent string) *Peer {
	return newPeer(address, agent, true, nil)
}

func NewOutgoingPeer(address string, agent string, handshakeInfo *HandshakeInfo) *Peer {
	return newPeer(address, agent, false, handshakeInfo)
}

func (p *Peer) Read() (*fos.Packet, error) {
	packet, err := fos.ReadPacket(p.reader)
	if p.handshakeTimeout > 0 {
		p.cancelHandshakeTimeout <- true
	}
	if err != nil {
		p2pLog.Error("Connection Read Err", zap.String("address", p.Address), zap.Error(err))
		return nil, errors.Wrapf(err, "connection: read %s err", p.Address)
	}
	return packet, nil
}

func (p *Peer) SetConnection(conn net.Conn) {
	p.connection = conn
	p.reader = bufio.NewReader(p.connection)
}

func (p *Peer) Connect(errChan chan error) (ready chan bool) {

	nodeID := make([]byte, 32)
	_, err := rand.Read(nodeID)
	if err != nil {
		errChan <- errors.Wrap(err, "generating random node id")
	}

	p.NodeID = nodeID
	hexNodeID := hex.EncodeToString(p.NodeID)
	p.Name = fmt.Sprintf("Client Peer - %s", hexNodeID[0:8])

	ready = make(chan bool, 1)
	go func() {
		address2log := zap.String("address", p.Address)

		if p.listener {
			p2pLog.Debug("Listening on", address2log)

			ln, err := net.Listen("tcp", p.Address)
			if err != nil {
				errChan <- errors.Wrapf(err, "peer init: listening %s", p.Address)
			}

			p2pLog.Debug("Accepting connection on", address2log)
			conn, err := ln.Accept()
			if err != nil {
				errChan <- errors.Wrapf(err, "peer init: accepting connection on %s", p.Address)
			}
			p2pLog.Debug("Connected on", address2log)

			p.SetConnection(conn)
			ready <- true

		} else {
			if p.handshakeTimeout > 0 {
				go func(p *Peer) {
					select {
					case <-time.After(p.handshakeTimeout):
						p2pLog.Warn("handshake took too long", address2log)
						errChan <- errors.Wrapf(err, "handshake took too long: %s", p.Address)
					case <-p.cancelHandshakeTimeout:
						p2pLog.Warn("cancelHandshakeTimeout canceled", address2log)
					}
				}(p)
			}

			p2pLog.Info("Dialing", address2log, zap.Duration("timeout", p.connectionTimeout))
			conn, err := net.DialTimeout("tcp", p.Address, p.connectionTimeout)
			if err != nil {
				if p.handshakeTimeout > 0 {
					p.cancelHandshakeTimeout <- true
				}
				errChan <- errors.Wrapf(err, "peer init: dial %s", p.Address)
				return
			}
			p2pLog.Info("Connected to", address2log)
			p.connection = conn
			p.reader = bufio.NewReader(conn)
			ready <- true
		}
	}()

	return
}

func (p *Peer) Write(bytes []byte) (int, error) {

	return p.connection.Write(bytes)
}

func (p *Peer) WriteP2PMessage(message fos.P2PMessage) (err error) {

	packet := &fos.Packet{
		Type:       message.GetType(),
		P2PMessage: message,
	}

	buff := bytes.NewBuffer(make([]byte, 0, 512))

	encoder := fos.NewEncoder(buff)
	err = encoder.Encode(packet)
	if err != nil {
		return errors.Wrapf(err, "unable to encode message %s", message)
	}

	_, err = p.Write(buff.Bytes())
	if err != nil {
		return errors.Wrapf(err, "write msg to %s", p.Address)
	}

	return nil
}

func (p *Peer) SendSyncRequest(startBlockNum uint32, endBlockNumber uint32) (err error) {
	p2pLog.Debug("SendSyncRequest",
		zap.String("peer", p.Address),
		zap.Uint32("start", startBlockNum),
		zap.Uint32("end", endBlockNumber))

	syncRequest := &fos.SyncRequestMessage{
		StartBlock: startBlockNum,
		EndBlock:   endBlockNumber,
	}

	return errors.WithStack(p.WriteP2PMessage(syncRequest))
}
func (p *Peer) SendRequest(startBlockNum uint32, endBlockNumber uint32) (err error) {
	p2pLog.Debug("SendRequest",
		zap.String("peer", p.Address),
		zap.Uint32("start", startBlockNum),
		zap.Uint32("end", endBlockNumber))

	request := &fos.RequestMessage{
		ReqTrx: fos.OrderedBlockIDs{
			Mode:    [4]byte{0, 0, 0, 0},
			Pending: startBlockNum,
		},
		ReqBlocks: fos.OrderedBlockIDs{
			Mode:    [4]byte{0, 0, 0, 0},
			Pending: endBlockNumber,
		},
	}

	return errors.WithStack(p.WriteP2PMessage(request))
}

func (p *Peer) SendNotice(headBlockNum uint32, libNum uint32, mode byte) error {
	p2pLog.Debug("Send Notice",
		zap.String("peer", p.Address),
		zap.Uint32("head", headBlockNum),
		zap.Uint32("lib", libNum),
		zap.Uint8("type", mode))

	notice := &fos.NoticeMessage{
		KnownTrx: fos.OrderedBlockIDs{
			Mode:    [4]byte{mode, 0, 0, 0},
			Pending: headBlockNum,
		},
		KnownBlocks: fos.OrderedBlockIDs{
			Mode:    [4]byte{mode, 0, 0, 0},
			Pending: libNum,
		},
	}
	return errors.WithStack(p.WriteP2PMessage(notice))
}

func (p *Peer) SendTime() error {
	p2pLog.Debug("SendTime", zap.String("peer", p.Address))

	notice := &fos.TimeMessage{}
	return errors.WithStack(p.WriteP2PMessage(notice))
}

func (p *Peer) SendHandshake(info *HandshakeInfo) error {

	publicKey, err := fecc.NewPublicKey("EOS1111111111111111111111111111111114T1Anm")
	if err != nil {
		return errors.Wrapf(err, "sending handshake to %s: create public key", p.Address)
	}

	p2pLog.Debug("SendHandshake", zap.String("peer", p.Address), zap.Object("info", info))

	tstamp := fos.Tstamp{Time: info.HeadBlockTime}

	signature := fecc.Signature{
		Curve:   fecc.CurveK1,
		Content: make([]byte, 65, 65),
	}

	handshake := &fos.HandshakeMessage{
		NetworkVersion:           1206,
		ChainID:                  info.ChainID,
		NodeID:                   p.NodeID,
		Key:                      publicKey,
		Time:                     tstamp,
		Token:                    make([]byte, 32, 32),
		Signature:                signature,
		P2PAddress:               p.Name,
		LastIrreversibleBlockNum: info.LastIrreversibleBlockNum,
		LastIrreversibleBlockID:  info.LastIrreversibleBlockID,
		HeadNum:                  info.HeadBlockNum,
		HeadID:                   info.HeadBlockID,
		OS:                       runtime.GOOS,
		Agent:                    p.agent,
		Generation:               int16(1),
	}

	err = p.WriteP2PMessage(handshake)
	if err != nil {
		err = errors.Wrapf(err, "sending handshake to %s", p.Address)
	}

	return nil
}
