package net

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/robertlestak/centauri/internal/keys"
	"github.com/robertlestak/centauri/internal/persist"
	log "github.com/sirupsen/logrus"
)

type DataMessageType string

var (
	DataMessageRequest  = DataMessageType("request")
	DataMessageResponse = DataMessageType("response")
	ErrorMissingFields  = "missing fields"
	SearchingPeerData   = make(map[*DataMessage][]*NodeMeta)
)

type DataMessage struct {
	Type     DataMessageType `json:"type"`
	PeerName *string         `json:"peer_name"`
	PeerAddr *string         `json:"peerAddr,omitempty"`
	PeerPort *int            `json:"peerPort,omitempty"`
	PubKeyID *string         `json:"pubKeyID,omitempty"`
	Channel  *string         `json:"channel,omitempty"`
	Sig      *string         `json:"sig,omitempty"`
	ID       *string         `json:"id,omitempty"`
	Data     *[]byte         `json:"data,omitempty"`
	Error    *string         `json:"error,omitempty"`
}

func (m *DataMessage) createSig() error {
	l := log.WithFields(log.Fields{
		"pkg": "net",
		"fn":  "DataMessage.createSig",
	})
	l.Debug("Creating signature")
	if len(PeerKey) == 0 {
		l.Error("Peer token is nil")
		return nil
	}
	var sm struct {
		Time int64 `json:"time"`
	}
	sm.Time = time.Now().Unix()
	data, err := json.Marshal(sm)
	if err != nil {
		l.Errorf("failed to marshal message: %v", err)
		return err
	}
	l.Debugf("Marshalled message: %s", string(data))
	// encrypt message with PeerKey
	enc, err := keys.AESEncrypt(data, PeerKey)
	if err != nil {
		l.Errorf("failed to encrypt message: %v", err)
		return err
	}
	l.Debugf("Encrypted message: %s", enc)
	m.Sig = &enc
	return nil
}

func (m *DataMessage) validateSig() error {
	l := log.WithFields(log.Fields{
		"pkg": "net",
		"fn":  "DataMessage.validateSig",
	})
	l.Debug("Validating signature")
	if len(PeerKey) == 0 {
		l.Error("Peer token is nil")
		return nil
	}
	if m.Sig == nil {
		l.Error("Signature is nil")
		return errors.New("signature is nil")
	}
	// decrypt message with PeerKey
	dec, err := keys.AESDecrypt(*m.Sig, PeerKey)
	if err != nil {
		l.Errorf("failed to decrypt message: %v", err)
		return err
	}
	l.Debugf("Decrypted message: %s", dec)
	// unmarshal message
	var sm struct {
		Time int64 `json:"time"`
	}
	err = json.Unmarshal(dec, &sm)
	if err != nil {
		l.Errorf("failed to unmarshal message: %v", err)
		return err
	}
	l.Debugf("Unmarshalled message: %v", sm)
	return nil
}

func writeMessage(conn net.Conn, m *DataMessage) error {
	l := log.WithFields(log.Fields{
		"module": "net",
		"method": "writeMessage",
	})
	l.Debug("Writing message")
	if err := m.createSig(); err != nil {
		l.Errorf("failed to create signature: %v", err)
		return err
	}
	// marshal message
	msg, err := json.Marshal(m)
	if err != nil {
		l.Errorf("error marshalling message: %v", err)
		return err
	}
	l.Debugf("Marshalled message: %s", string(msg))
	// write message
	// append newline to message
	msg = append(msg, '\n')
	_, err = conn.Write(msg)
	if err != nil {
		l.Errorf("error writing message: %v", err)
		return err
	}
	l.Debug("Message written")
	return nil
}

func readMessage(conn net.Conn) (*DataMessage, error) {
	l := log.WithFields(log.Fields{
		"module": "net",
		"method": "readMessage",
	})
	l.Debug("Reading message")
	reader := bufio.NewReader(conn)
	var buffer bytes.Buffer
	for {
		ba, isPrefix, err := reader.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		l.Debugf("Read %d bytes", len(ba))
		l.Debugf("data: %s", string(ba))
		buffer.Write(ba)
		if !isPrefix {
			break
		}
	}
	l.Debugf("Received message: %s", buffer.String())
	if buffer.String() == "" {
		return nil, nil
	}
	// parse message
	var dataMsg DataMessage
	err := json.Unmarshal(buffer.Bytes(), &dataMsg)
	if err != nil {
		l.Errorf("failed to unmarshal message: %v", err)
		return nil, err
	}
	l.Debugf("Parsed message: %v", dataMsg)
	if err := dataMsg.validateSig(); err != nil {
		l.Errorf("failed to validate signature: %v", err)
		return nil, err
	}
	return &dataMsg, nil
}

func RequestDataFromPeerBestEffort(peerAddr string, peerPort int, pubKeyID string, channel string, id string) ([]byte, error) {
	l := log.WithFields(log.Fields{
		"module": "net",
		"method": "RequestDataFromPeerBestEffort",
	})
	l.Debug("Requesting data from peer")
	d, err := RequestDataFromPeer(peerAddr, peerPort, pubKeyID, channel, id)
	if err != nil {
		l.Errorf("failed to request data from original peer: %v", err)
		// we were unable to get the data from the original peer, let's try from our other peers
		checkLimit := 10
		for i, p := range ListMembers() {
			if i >= checkLimit {
				return nil, errors.New("failed to get data from any peer")
			}
			nm := &NodeMeta{}
			if err := json.Unmarshal(p.Meta, nm); err != nil {
				l.Errorf("failed to unmarshal meta: %v", err)
				continue
			}
			if nm.PeerAddr == peerAddr && nm.PeerPort == peerPort {
				continue
			}
			if nm.PeerAddr == PeerAddr && nm.PeerPort == PeerDataPort {
				continue
			}
			d, err := RequestDataFromPeer(nm.PeerAddr, nm.PeerPort, pubKeyID, channel, id)
			if err != nil {
				l.Errorf("failed to request data from peer: %v", err)
				continue
			}
			return d, nil
		}
	}
	return d, nil
}

func RequestDataFromPeer(peerAddr string, peerPort int, pubKeyID string, channel string, id string) ([]byte, error) {
	l := log.WithFields(log.Fields{
		"module": "net",
		"method": "RequestDataFromPeer",
	})
	l.Debugf("Requesting data from peer %s:%d", peerAddr, peerPort)
	// create a tcp connection
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", peerAddr, peerPort))
	if err != nil {
		l.Errorf("failed to connect to peer: %v", err)
		return nil, err
	}
	l.Debug("connected to peer")
	// write message
	err = writeMessage(conn, &DataMessage{
		Type:     DataMessageRequest,
		PeerName: &PeerName,
		PubKeyID: &pubKeyID,
		Channel:  &channel,
		ID:       &id,
	})
	if err != nil {
		l.Errorf("failed to write message: %v", err)
		return nil, err
	}
	l.Debug("wrote message")
	dataMsg, err := readMessage(conn)
	if err != nil {
		l.Errorf("failed to read message: %v", err)
		return nil, err
	}
	if dataMsg == nil {
		l.Error("no data message received")
		return nil, errors.New("no data message received")
	}
	// handle message
	if dataMsg.Error != nil {
		l.Errorf("error in message: %v", *dataMsg.Error)
		return nil, fmt.Errorf("error in message: %v", *dataMsg.Error)
	}
	l.Debug("Message handled")
	// return data
	return *dataMsg.Data, nil
}

func handleDataConnection(conn net.Conn) {
	l := log.WithFields(log.Fields{
		"pkg": "net",
		"fn":  "handleDataConnection",
	})
	l.Debug("Handling data connection")
	defer conn.Close()
	var err error
	// read message
	dataMsg, err := readMessage(conn)
	if err != nil {
		l.Errorf("failed to read message: %v", err)
		return
	}
	if dataMsg == nil {
		l.Debug("No message received")
		return
	}
	if !PeerInList(*dataMsg.PeerName) {
		l.Debug("Peer not in list")
		err := "Peer not in list"
		writeMessage(conn, &DataMessage{
			Type:  DataMessageResponse,
			Error: &err,
		})
		return
	}
	switch dataMsg.Type {
	case DataMessageRequest:
		l.Debugf("Received message: %v", dataMsg)
		md, err := persist.GetMessageByID(*dataMsg.PubKeyID, *dataMsg.Channel, *dataMsg.ID)
		if err != nil {
			e := err.Error()
			l.Errorf("failed to get message: %v", err)
			writeMessage(conn, &DataMessage{
				Type:     DataMessageResponse,
				PeerName: &PeerName,
				Error:    &e,
			})
			return
		}
		l.Debugf("Got message: %v", md)
		// write message
		writeMessage(conn, &DataMessage{
			Type:     DataMessageResponse,
			PeerName: &PeerName,
			ID:       dataMsg.ID,
			PubKeyID: dataMsg.PubKeyID,
			Channel:  dataMsg.Channel,
			Data:     &md,
		})
	default:
		l.Errorf("Unknown message type: %v", dataMsg.Type)
	}
}

func DataServer(port int) {
	l := log.WithFields(log.Fields{
		"pkg": "net",
		"fn":  "DataServer",
	})
	l.Debugf("Starting data server on port %d", port)
	// create a tcp listener
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		l.Errorf("failed to start data server: %v", err)
		return
	}
	l.Debug("data server started")
	for {
		// accept a connection
		conn, err := listener.Accept()
		if err != nil {
			l.Errorf("failed to accept connection: %v", err)
			continue
		}
		l.Debug("accepted connection")
		// handle the connection
		go handleDataConnection(conn)
	}
}
