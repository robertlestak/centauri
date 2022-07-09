package events

import (
	"encoding/json"
	"errors"

	log "github.com/sirupsen/logrus"
)

var (
	PeerName                 string
	DeletionHandlers         = []func(pubKeyID, channel string, id string) error{}
	NewMessageHandlers       = []func(pubKeyID, channel string, id string) error{}
	ReceivedDeletionHandlers = []func(pubKeyID, channel string, id string, eventTrigger bool) error{}
	ReceivedMessageHandlers  = []func(pubKeyID string, channel string, id string, peerAddr string, peerPort int) error{}
)

func DeleteMessage(pubKeyID, channel, id string) {
	l := log.WithFields(log.Fields{
		"pkg": "events",
		"fn":  "DeleteMessage",
	})
	l.Debug("deleting message")
	for _, f := range DeletionHandlers {
		go f(pubKeyID, channel, id)
	}
}

func NewMessage(pubKeyID, channel string, id string) {
	l := log.WithFields(log.Fields{
		"pkg": "events",
		"fn":  "NewMessage",
	})
	l.Debug("new message")
	for _, f := range NewMessageHandlers {
		go f(pubKeyID, channel, id)
	}
}

func ReceiveMessage(data []byte) error {
	l := log.WithFields(log.Fields{
		"pkg": "events",
		"fn":  "ReceiveMessage",
	})
	l.Debug("receiving message")
	var md map[string]any
	if err := json.Unmarshal(data, &md); err != nil {
		l.Errorf("error unmarshalling message: %v", err)
		return err
	}
	switch md["type"] {
	case "newMessage":
		l.Debug("new message")
		pubKeyID := md["pubKeyID"].(string)
		id := md["id"].(string)
		channel := md["channel"].(string)
		peerAddr := md["peerAddr"].(string)
		peerDataPort := int(md["peerPort"].(float64))
		for _, f := range ReceivedMessageHandlers {
			if err := f(pubKeyID, channel, id, peerAddr, peerDataPort); err != nil {
				l.Errorf("error receiving message: %v", err)
				return err
			}
		}
	case "deleteMessage":
		l.Debug("delete message")
		pubKeyID := md["pubKeyID"].(string)
		channel := md["channel"].(string)
		id := md["id"].(string)
		for _, f := range ReceivedDeletionHandlers {
			if err := f(pubKeyID, channel, id, true); err != nil {
				l.Errorf("error deleting message: %v", err)
				return err
			}
		}
	default:
		l.Errorf("unknown message type: %v", md["type"])
		return errors.New("unknown message type")
	}
	return nil
}
