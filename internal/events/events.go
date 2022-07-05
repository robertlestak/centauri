package events

import (
	"encoding/base64"
	"encoding/json"
	"errors"

	log "github.com/sirupsen/logrus"
)

var (
	PeerName                 string
	DeletionHandlers         = []func(pubKeyID, channel string, id string) error{}
	NewMessageHandlers       = []func(pubKeyID, channel string, id string, data []byte) error{}
	ReceivedDeletionHandlers = []func(pubKeyID, channel string, id string, eventTrigger bool) error{}
	ReceivedMessageHandlers  = []func(pubKeyID, channel string, id string, data []byte) error{}
)

func DeleteMessage(pubKeyID, channel, id string) error {
	l := log.WithFields(log.Fields{
		"pkg": "events",
		"fn":  "DeleteMessage",
	})
	l.Debug("deleting message")
	for _, f := range DeletionHandlers {
		if err := f(pubKeyID, channel, id); err != nil {
			l.Errorf("error deleting message: %v", err)
			return err
		}
	}
	return nil
}

func NewMessage(pubKeyID, channel string, id string, data []byte) error {
	l := log.WithFields(log.Fields{
		"pkg": "events",
		"fn":  "NewMessage",
	})
	l.Debug("new message")
	for _, f := range NewMessageHandlers {
		if err := f(pubKeyID, channel, id, data); err != nil {
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
		data := md["data"].(string)
		bd, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
			l.Errorf("error decoding message: %v", err)
			return err
		}
		for _, f := range ReceivedMessageHandlers {
			if err := f(pubKeyID, channel, id, bd); err != nil {
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
