package events

import (
	"encoding/base64"
	"encoding/json"
	"errors"

	log "github.com/sirupsen/logrus"
)

var (
	PeerName                 string
	DeletionHandlers         = []func(pubKeyID, id string) error{}
	NewMessageHandlers       = []func(pubKeyID, id string, data []byte) error{}
	ReceivedDeletionHandlers = []func(pubKeyID, id string, eventTrigger bool) error{}
	ReceivedMessageHandlers  = []func(pubKeyID, id string, data []byte) error{}
)

func DeleteMessage(pubKeyID, id string) error {
	l := log.WithFields(log.Fields{
		"pkg": "events",
		"fn":  "DeleteMessage",
	})
	l.Info("deleting message")
	for _, f := range DeletionHandlers {
		if err := f(pubKeyID, id); err != nil {
			l.Errorf("error deleting message: %v", err)
			return err
		}
	}
	return nil
}

func NewMessage(pubKeyID, id string, data []byte) error {
	l := log.WithFields(log.Fields{
		"pkg": "events",
		"fn":  "NewMessage",
	})
	l.Info("new message")
	for _, f := range NewMessageHandlers {
		if err := f(pubKeyID, id, data); err != nil {
			l.Errorf("error new message: %v", err)
			return err
		}
	}
	return nil
}

func ReceiveMessage(data []byte) error {
	l := log.WithFields(log.Fields{
		"pkg": "events",
		"fn":  "ReceiveMessage",
	})
	l.Info("receiving message")
	var md map[string]any
	if err := json.Unmarshal(data, &md); err != nil {
		l.Errorf("error unmarshalling message: %v", err)
		return err
	}
	switch md["type"] {
	case "newMessage":
		l.Info("new message")
		pubKeyID := md["pubKeyID"].(string)
		id := md["id"].(string)
		data := md["data"].(string)
		bd, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
			l.Errorf("error decoding message: %v", err)
			return err
		}
		for _, f := range ReceivedMessageHandlers {
			if err := f(pubKeyID, id, bd); err != nil {
				l.Errorf("error receiving message: %v", err)
				return err
			}
		}
	case "deleteMessage":
		l.Info("delete message")
		pubKeyID := md["pubKeyID"].(string)
		id := md["id"].(string)
		for _, f := range ReceivedDeletionHandlers {
			if err := f(pubKeyID, id, true); err != nil {
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
