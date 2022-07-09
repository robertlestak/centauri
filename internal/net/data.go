package net

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"

	"github.com/robertlestak/centauri/internal/persist"
	log "github.com/sirupsen/logrus"
)

type DataMessageType string

var (
	DataMessageRequest  = DataMessageType("request")
	DataMessageResponse = DataMessageType("response")
	ErrorMissingFields  = "missing fields"
)

type DataMessage struct {
	Type     DataMessageType `json:"type"`
	PeerAddr *string         `json:"peerAddr,omitempty"`
	PeerPort *int            `json:"peerPort,omitempty"`
	PubKeyID *string         `json:"pubKeyID,omitempty"`
	Channel  *string         `json:"channel,omitempty"`
	ID       *string         `json:"id,omitempty"`
	Data     *[]byte         `json:"data,omitempty"`
	Error    *string         `json:"error,omitempty"`
}

func writeMessage(conn net.Conn, m *DataMessage) error {
	l := log.WithFields(log.Fields{
		"module": "net",
		"method": "writeMessage",
	})
	l.Debug("Writing message")
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
	// parse message
	var dataMsg DataMessage
	err := json.Unmarshal(buffer.Bytes(), &dataMsg)
	if err != nil {
		l.Errorf("failed to unmarshal message: %v", err)
		return nil, err
	}
	l.Debugf("Parsed message: %v", dataMsg)
	return &dataMsg, nil
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
	l.Debugf("Received message: %v", dataMsg)
	md, err := persist.GetMessageByID(*dataMsg.PubKeyID, *dataMsg.Channel, *dataMsg.ID)
	if err != nil {
		e := err.Error()
		l.Errorf("failed to get message: %v", err)
		writeMessage(conn, &DataMessage{
			Type:  DataMessageResponse,
			Error: &e,
		})
		return
	}
	l.Debugf("Got message: %v", md)
	// write message
	writeMessage(conn, &DataMessage{
		Type:     DataMessageResponse,
		ID:       dataMsg.ID,
		PubKeyID: dataMsg.PubKeyID,
		Channel:  dataMsg.Channel,
		Data:     &md,
	})
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
