package message

import (
	"errors"
	"io"
	"io/ioutil"
	"regexp"

	"github.com/google/uuid"
	"github.com/robertlestak/centauri/internal/events"
	"github.com/robertlestak/centauri/internal/keys"
	"github.com/robertlestak/centauri/internal/net"
	"github.com/robertlestak/centauri/internal/persist"
	log "github.com/sirupsen/logrus"
)

type Metadata struct {
	Name string `json:"name"`
}

type Message struct {
	Type        string `json:"type"`
	Channel     string `json:"channel"`
	ID          string `json:"id"`
	PublicKeyID string `json:"pubKeyID,omitempty"`
	Data        []byte `json:"data"`
}

func validateType(t string) error {
	l := log.WithFields(log.Fields{
		"pkg": "message",
		"fn":  "validateType",
	})
	l.Debugf("validating type: %s", t)
	if t == "" {
		l.Error("type is empty")
		return errors.New("type is required")
	}
	switch t {
	case "file":
		l.Debug("type is valid")
	case "bytes":
		l.Debug("type is valid")
	default:
		l.Errorf("type is invalid: %s", t)
		return errors.New("type is invalid")
	}
	return nil
}

// CleanString will clean the string to [a-zA-Z0-9_]
func CleanString(s string) string {
	// given input s, replace all characters that are not a-zA-Z0-9_ with _
	var ns string
	for _, r := range s {
		rx := regexp.MustCompile(`[^a-zA-Z0-9\-]`)
		if !rx.MatchString(string(r)) {
			ns += string(r)
		} else {
			ns += "-"
		}
	}
	return ns
}

func (m *Message) Create() (*Message, error) {
	l := log.WithFields(log.Fields{
		"pkg": "message",
		"fn":  "Create",
	})
	l.Debug("creating message")
	if m.PublicKeyID == "" {
		l.Error("public key id is empty")
		return nil, errors.New("public key id is required")
	}
	if len(m.Data) == 0 {
		l.Error("data is empty")
		return nil, errors.New("data is empty")
	}
	if err := validateType(m.Type); err != nil {
		l.Errorf("invalid type: %v", err)
		return nil, err
	}
	m.ID = uuid.New().String()
	m.Channel = CleanString(m.Channel)
	if err := m.StoreLocal(); err != nil {
		l.Errorf("error storing message: %v", err)
		return nil, err
	}
	events.NewMessage(m.PublicKeyID, m.Channel, m.ID)
	return m, nil
}

func (m *Message) StoreLocal() error {
	l := log.WithFields(log.Fields{
		"pkg": "message",
		"fn":  "StoreLocal",
	})
	l.Debug("storing message locally")
	m.Channel = CleanString(m.Channel)
	if err := persist.StoreMessage(m.PublicKeyID, m.Channel, m.ID, m.Data); err != nil {
		l.Errorf("error storing message: %v", err)
		return err
	}
	return nil
}

func ListMessageMetaForPubKeyID(pubKeyID string, channel string) ([]persist.MessageMetaData, error) {
	l := log.WithFields(log.Fields{
		"pkg": "message",
		"fn":  "ListMessageMetaForPubKeyID",
		"id":  pubKeyID,
		"ch":  channel,
	})
	l.Debug("listing messages for public key")
	channel = CleanString(channel)
	return persist.ListMessageMetaForPubKeyID(pubKeyID, channel)
}

func GetMessageByID(pubKeyID string, channel string, id string) (*Message, error) {
	l := log.WithFields(log.Fields{
		"pkg": "message",
		"fn":  "GetMessageByID",
	})
	l.Debug("getting message by id")
	channel = CleanString(channel)
	data, err := persist.GetMessageByID(pubKeyID, channel, id)
	if err != nil {
		l.Errorf("error getting message: %v", err)
		return nil, err
	}
	m := &Message{
		ID:          id,
		PublicKeyID: pubKeyID,
		Channel:     channel,
		Data:        data,
	}
	return m, nil
}

func GetMessageFromPeer(pubKeyID string, channel string, id string, peerAddr string, peerPort int) error {
	l := log.WithFields(log.Fields{
		"pkg":      "message",
		"fn":       "GetMessageFromPeer",
		"pubKeyID": pubKeyID,
		"channel":  channel,
		"id":       id,
		"peerAddr": peerAddr,
		"peerPort": peerPort,
	})
	l.Debugf("getting message from peer %s:%d", peerAddr, peerPort)
	md, err := net.RequestDataFromPeerBestEffort(peerAddr, peerPort, pubKeyID, channel, id)
	if err != nil {
		l.Errorf("error getting message: %v", err)
		return err
	}
	channel = CleanString(channel)
	msg := &Message{
		Type:        "bytes",
		ID:          id,
		Channel:     channel,
		PublicKeyID: pubKeyID,
		Data:        md,
	}
	if err := msg.StoreLocal(); err != nil {
		l.Errorf("error storing message: %v", err)
		return err
	}
	return nil
}

func DeleteMessageByID(pubKeyID string, channel string, id string, eventTrigger bool) error {
	l := log.WithFields(log.Fields{
		"pkg": "message",
		"fn":  "DeleteMessageByID",
	})
	l.Debug("deleting message by id")
	channel = CleanString(channel)
	if err := persist.DeleteMessageByID(pubKeyID, channel, id); err != nil {
		l.Errorf("error deleting message: %v", err)
		return err
	}
	if !eventTrigger {
		events.DeleteMessage(pubKeyID, channel, id)
	}
	return nil
}

func CreateMessage(mType string, fileName string, channel string, pubKeyID string, rawDataReader io.ReadCloser) (*Message, error) {
	l := log.WithFields(log.Fields{
		"pkg":     "message",
		"fn":      "CreateMessage",
		"type":    mType,
		"file":    fileName,
		"channel": channel,
		"pubkey":  pubKeyID,
	})
	l.Debug("creating message")
	var pubKey []byte
	// get public key for pubKeyID
	channel = CleanString(channel)
	if k, ok := keys.PublicKeyChain[pubKeyID]; !ok {
		l.Errorf("public key not found: %s", pubKeyID)
		return nil, errors.New("public key not found")
	} else {
		pubKey = k
	}
	var rawData []byte
	// read rawData from rawDataReader
	if rawDataReader != nil {
		var err error
		rawData, err = ioutil.ReadAll(rawDataReader)
		if err != nil {
			l.Errorf("error reading raw data: %v", err)
			return nil, err
		}
	}
	if mType == "file" && fileName != "" {
		// add file:<filename>| prefix to rawData
		rawData = append([]byte("file:"+fileName+"|"), rawData...)
	}
	enc, err := keys.EncryptMessage(pubKey, rawData)
	if err != nil {
		l.Errorf("error encrypting data: %v", err)
		return nil, err
	}
	m := &Message{
		Type:        mType,
		Channel:     channel,
		PublicKeyID: pubKeyID,
		Data:        []byte(*enc),
	}
	return m, nil
}
