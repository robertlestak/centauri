package sign

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/robertlestak/centauri/internal/keys"
)

type SignedRequest struct {
	Action    string `json:"action"`
	PublicKey []byte `json:"public_key"`
	Signature []byte `json:"signature"`
	Data      []byte `json:"data"`
}

type SignedMessageData struct {
	Timestamp int64 `json:"timestamp"`
}

func PubKeyID(pubKeyBytes []byte) string {
	h := sha256.Sum256(pubKeyBytes)
	return fmt.Sprintf("%x", h[:])
}

func HashSumMessage(msg []byte) []byte {
	// sha256 hash of message
	h := sha256.New()
	h.Write(msg)
	return h.Sum(nil)
}

func Verify(msg, sig, pubKey []byte) error {
	pub, err := keys.BytesToPubKey(pubKey)
	if err != nil {
		return err
	}
	hs := HashSumMessage(msg)
	return rsa.VerifyPKCS1v15(pub, crypto.SHA256, hs, sig)
}

func Sign(msg []byte, priv *rsa.PrivateKey) ([]byte, error) {
	// priv, err := keys.BytesToPrivKey(privKey)
	// if err != nil {
	// 	return nil, err
	// }
	hs := HashSumMessage(msg)
	return rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, hs)
}

func (r *SignedRequest) Verify() error {
	if err := Verify(r.Data, r.Signature, r.PublicKey); err != nil {
		return err
	}
	sd := SignedMessageData{}
	if err := json.Unmarshal(r.Data, &sd); err != nil {
		return err
	}
	// ensure timestamp is within last 5 minutes
	if sd.Timestamp < (time.Now().Unix() - 300) {
		return errors.New("timestamp is too old")
	}
	return nil
}

func (r *SignedRequest) VerifyOwnsID(id string) error {
	if err := r.Verify(); err != nil {
		return err
	}
	if PubKeyID(r.PublicKey) != id {
		return errors.New("public key does not own ID")
	}
	return nil
}
