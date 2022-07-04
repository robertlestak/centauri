package keys

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/robertlestak/centauri/internal/persist"
	log "github.com/sirupsen/logrus"
)

var (
	PublicKeyChain map[string][]byte
)

type MessageHeader struct {
	Key   string `json:"k"`
	Nonce string `json:"n"`
}

func BytesToPubKey(publicKey []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(publicKey)
	if block == nil {
		return nil, errors.New("public key error")
	}
	pubInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	pub := pubInterface.(*rsa.PublicKey)
	return pub, nil
}

func BytesToPrivKey(privateKey []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(privateKey)
	if block == nil {
		return nil, errors.New("private key error")
	}
	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return priv, nil
}

func LoadPubKeyChainFromDirectory(d string) error {
	l := log.WithFields(log.Fields{
		"pkg": "keys",
		"fn":  "LoadPubKeyChainFromDirectory",
		"dir": d,
	})
	l.Info("Loading public key chain")
	// check if dir exists
	if _, err := os.Stat(d); os.IsNotExist(err) {
		l.Error("Directory does not exist")
		return err
	}
	// loop through files in dir, where file name is the key id
	// and the contents is the public key
	newPubKeyChain := make(map[string][]byte)
	files, err := ioutil.ReadDir(d)
	if err != nil {
		l.Error("Error reading directory")
		return err
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		keyID := file.Name()
		keyFile := filepath.Join(d, keyID)
		keyBytes, err := ioutil.ReadFile(keyFile)
		if err != nil {
			l.Error("Error reading key file")
			return err
		}
		newPubKeyChain[keyID] = keyBytes
	}
	if err := ensureDirs(PublicKeyChain, newPubKeyChain); err != nil {
		l.Error("Error ensuring directories")
		return err
	}
	PublicKeyChain = newPubKeyChain
	l.Infof("Public key chain loaded, %d keys loaded", len(PublicKeyChain))
	return nil
}

func ensureDirs(oldKeys map[string][]byte, newKeys map[string][]byte) error {
	l := log.WithFields(log.Fields{
		"pkg": "keys",
		"fn":  "ensureDirs",
	})
	l.Info("Ensuring directories")
	var removed []string
	for keyID := range oldKeys {
		if _, ok := newKeys[keyID]; !ok {
			removed = append(removed, keyID)
		}
	}
	for k, _ := range newKeys {
		if _, err := persist.EnsurePubKeyChainOutgoingDir(k); err != nil {
			l.Error("Error ensuring outgoing directory")
			return err
		}
	}
	for _, keyID := range removed {
		if _, err := persist.RemovePubKeyChainOutgoingDir(keyID); err != nil {
			l.Error("Error ensuring outgoing directory")
			return err
		}
	}

	return nil
}

func PubKeyLoader(d string) {
	l := log.WithFields(log.Fields{
		"pkg": "keys",
		"fn":  "PubKeyLoader",
		"dir": d,
	})
	l.Info("Loading public key chain")
	for {
		err := LoadPubKeyChainFromDirectory(d)
		if err != nil {
			l.Error("Error loading public key chain")
		}
		time.Sleep(time.Minute * 5)
	}
}

func RsaEncrypt(publicKey []byte, origData []byte) ([]byte, error) {
	l := log.WithFields(log.Fields{
		"pkg": "keys",
		"fn":  "RsaEncrypt",
	})
	l.Info("encrypting data")
	l.Debugf("public key: %s", publicKey)
	pub, err := BytesToPubKey(publicKey)
	if err != nil {
		l.Errorf("error converting public key: %v", err)
		return nil, err
	}
	return rsa.EncryptOAEP(sha1.New(), rand.Reader, pub, origData, nil)
}

func RsaDecrypt(privateKey []byte, ciphertext []byte) ([]byte, error) {
	l := log.WithFields(log.Fields{
		"pkg": "keys",
		"fn":  "RsaDecrypt",
	})
	l.Info("decrypting data")
	block, _ := pem.Decode(privateKey)
	if block == nil {
		l.Error("error decoding private key")
		return nil, errors.New("private key error")
	}
	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		l.Error("error parsing private key")
		return nil, err
	}
	return rsa.DecryptOAEP(sha1.New(), rand.Reader, priv, ciphertext, nil)
}

func GenerateNewAESKey() ([]byte, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}
	return key, nil
}

// AesGcmEncrypt takes an encryption key and a plaintext string and encrypts it with AES256 in GCM mode, which provides authenticated encryption. Returns the ciphertext and the used nonce.
func AesGcmEncrypt(key []byte, raw []byte) ([]byte, []byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}

	// Never use more than 2^32 random nonces with a given key because of the risk of a repeat.
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}

	ciphertext := aesgcm.Seal(nil, nonce, raw, nil)
	fmt.Printf("Ciphertext: %x\n", ciphertext)
	fmt.Printf("Nonce: %x\n", nonce)

	return ciphertext, nonce, nil
}

// AesGcmDecrypt takes an decryption key, a ciphertext and the corresponding nonce and decrypts it with AES256 in GCM mode. Returns the plaintext string.
func AesGcmDecrypt(key, ciphertext, nonce []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	plaintextBytes, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintextBytes, nil
}

func EncryptMessage(key, data []byte) (*string, error) {
	l := log.WithFields(log.Fields{
		"pkg": "keys",
		"fn":  "EncryptMessage",
	})
	l.Info("Encrypting message")
	// create a new key
	aesKey, err := GenerateNewAESKey()
	if err != nil {
		l.Error("Error generating new AES key")
		return nil, err
	}
	// encrypt the data
	ciphertext, nonce, err := AesGcmEncrypt(aesKey, data)
	if err != nil {
		l.Error("Error encrypting data")
		return nil, err
	}
	hdr := &MessageHeader{
		Key:   hex.EncodeToString(aesKey),
		Nonce: hex.EncodeToString(nonce),
	}
	hdrBytes, err := json.Marshal(hdr)
	if err != nil {
		l.Error("Error marshalling header")
		return nil, err
	}
	// encrypt the header with the rsa key
	hdrEncrypted, err := RsaEncrypt(key, hdrBytes)
	if err != nil {
		l.Error("Error encrypting header")
		return nil, err
	}
	hexHdr := hex.EncodeToString(hdrEncrypted)
	// join the header bytes and the ciphertext bytes together
	// with a string "."
	sep := "."
	mes := hexHdr + sep + hex.EncodeToString(ciphertext)
	return &mes, nil
}

func DecryptMessage(key []byte, data string) ([]byte, error) {
	l := log.WithFields(log.Fields{
		"pkg": "keys",
		"fn":  "DecryptMessage",
	})
	l.Info("Decrypting message")
	l.Debugf("data: %s", data)
	// split the data into the header and the ciphertext
	sep := "."
	parts := strings.Split(data, sep)
	if len(parts) != 2 {
		l.Error("Error splitting data")
		return nil, errors.New("data error")
	}
	// decrypt the header
	hdrEncrypted := parts[0]
	// decode the header
	hdrBytes, err := hex.DecodeString(hdrEncrypted)
	if err != nil {
		l.Error("Error decoding header")
		return nil, err
	}
	hdrb, err := RsaDecrypt(key, hdrBytes)
	if err != nil {
		l.Error("Error decrypting header")
		return nil, err
	}
	l.Debugf("hdrb: %s", hdrb)
	// unmarshal the header
	var hdr MessageHeader
	err = json.Unmarshal(hdrb, &hdr)
	if err != nil {
		l.Error("Error unmarshalling header")
		return nil, err
	}
	// decrypt the ciphertext
	ciphertext := parts[1]
	cd, err := hex.DecodeString(ciphertext)
	if err != nil {
		l.Error("Error decoding ciphertext")
		return nil, err
	}
	l.Debugf("Key: %s", hdr.Key)
	l.Debugf("Nonce: %s", hdr.Nonce)
	kd, err := hex.DecodeString(hdr.Key)
	if err != nil {
		l.Error("Error decoding key")
		return nil, err
	}
	nd, err := hex.DecodeString(hdr.Nonce)
	if err != nil {
		l.Error("Error decoding nonce")
		return nil, err
	}
	plaintext, err := AesGcmDecrypt(kd, cd, nd)
	if err != nil {
		l.Error("Error decrypting data")
		return nil, err
	}
	return plaintext, nil
}
