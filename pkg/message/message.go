package message

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"

	"github.com/google/uuid"
	"github.com/robertlestak/mp/internal/events"
	"github.com/robertlestak/mp/internal/keys"
	"github.com/robertlestak/mp/internal/persist"
	log "github.com/sirupsen/logrus"
)

type Metadata struct {
	Name string `json:"name"`
}

type Message struct {
	Type        string `json:"type"`
	ID          string `json:"id"`
	PublicKeyID string `json:"public_key_id,omitempty"`
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
	case "message":
		l.Debug("type is valid")
	default:
		l.Errorf("type is invalid: %s", t)
		return errors.New("type is invalid")
	}
	return nil
}

func RsaEncrypt(publicKey []byte, origData []byte) ([]byte, error) {
	l := log.WithFields(log.Fields{
		"pkg": "message",
		"fn":  "RsaEncrypt",
	})
	l.Info("encrypting data")
	l.Debugf("public key: %s", publicKey)
	pub, err := keys.BytesToPubKey(publicKey)
	if err != nil {
		l.Errorf("error converting public key: %v", err)
		return nil, err
	}
	return rsa.EncryptPKCS1v15(rand.Reader, pub, origData)
}

func RsaDecrypt(privateKey []byte, ciphertext []byte) ([]byte, error) {
	block, _ := pem.Decode(privateKey)
	if block == nil {
		return nil, errors.New("private key error")
	}
	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return rsa.DecryptPKCS1v15(rand.Reader, priv, ciphertext)
}

func (m *Message) Create() (*Message, error) {
	l := log.WithFields(log.Fields{
		"pkg": "message",
		"fn":  "Create",
	})
	l.Info("creating message")
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
	//m.PublicKeyID = sign.PubKeyID(m.PublicKeyBytes)
	// enc, err := RsaEncrypt(m.PublicKeyBytes, m.Data)
	// if err != nil {
	// 	l.Errorf("error encrypting data: %v", err)
	// 	return nil, err
	// }
	// m.Data = enc
	if err := m.StoreLocal(); err != nil {
		l.Errorf("error storing message: %v", err)
		return nil, err
	}
	if err := events.NewMessage(m.PublicKeyID, m.ID, m.Data); err != nil {
		l.Errorf("error creating message: %v", err)
		return nil, err
	}
	return m, nil
}

func (m *Message) StoreLocal() error {
	l := log.WithFields(log.Fields{
		"pkg": "message",
		"fn":  "StoreLocal",
	})
	l.Info("storing message locally")
	if err := persist.StoreMessage(m.PublicKeyID, m.ID, m.Data); err != nil {
		l.Errorf("error storing message: %v", err)
		return err
	}
	return nil
}

func ListMessageMetaForPubKeyID(pubKeyID string) ([]persist.MessageMetaData, error) {
	l := log.WithFields(log.Fields{
		"pkg": "message",
		"fn":  "ListMessageMetaForPubKeyID",
	})
	l.Info("listing messages for public key")
	return persist.ListMessageMetaForPubKeyID(pubKeyID)
}

func GetMessageByID(pubKeyID string, id string) (*Message, error) {
	l := log.WithFields(log.Fields{
		"pkg": "message",
		"fn":  "GetMessageByID",
	})
	l.Info("getting message by id")
	data, err := persist.GetMessageByID(pubKeyID, id)
	if err != nil {
		l.Errorf("error getting message: %v", err)
		return nil, err
	}
	m := &Message{
		ID:          id,
		PublicKeyID: pubKeyID,
		Data:        data,
	}
	return m, nil
}

func GetMessageFromPeer(pubKeyID string, id string, data []byte) error {
	l := log.WithFields(log.Fields{
		"pkg": "message",
		"fn":  "GetMessageFromPeer",
	})
	l.Info("getting message from peer")
	msg := &Message{
		Type:        "message",
		ID:          id,
		PublicKeyID: pubKeyID,
		Data:        data,
	}
	if err := msg.StoreLocal(); err != nil {
		l.Errorf("error storing message: %v", err)
		return err
	}
	return nil
}

func DeleteMessageByID(pubKeyID string, id string, eventTrigger bool) error {
	l := log.WithFields(log.Fields{
		"pkg": "message",
		"fn":  "DeleteMessageByID",
	})
	l.Info("deleting message by id")
	if err := persist.DeleteMessageByID(pubKeyID, id); err != nil {
		l.Errorf("error deleting message: %v", err)
		return err
	}
	if !eventTrigger {
		if err := events.DeleteMessage(pubKeyID, id); err != nil {
			l.Errorf("error deleting message: %v", err)
			return err
		}
	}
	return nil
}
