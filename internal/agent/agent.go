package agent

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/robertlestak/centauri/internal/keys"
	"github.com/robertlestak/centauri/internal/persist"
	"github.com/robertlestak/centauri/internal/sign"
	"github.com/robertlestak/centauri/pkg/message"
	log "github.com/sirupsen/logrus"
)

var (
	ServerAddrs              []string
	ServerAuthToken          string
	DefaultChannel           string = "default"
	PrivateKey               *rsa.PrivateKey
	lastServer               int
	Output                   string
	OutputFormat             string = "json"
	ClientMessageID          string
	ClientRecipientPublicKey []byte
	ClientMessageInput       string
	ClientMessageType        string
	ClientMessageFileName    string
)

type MessageMeta struct {
	ID        string    `json:"id"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
}

type GetJob struct {
	Channel string
	ID      string
}

func getMessageData(channel, id string) (*message.Message, string, error) {
	l := log.WithFields(log.Fields{
		"pkg": "agent",
		"fn":  "getMessageData",
	})
	l.Debugf("getting message %s", id)
	m, err := GetMessage(channel, id)
	if err != nil {
		l.Errorf("error getting message %s: %v", id, err)
		return nil, "", err
	}
	if m == nil {
		l.Errorf("message %s not found", id)
		return nil, "", fmt.Errorf("message %s not found", id)
	}
	m, err = DecryptMessageData(m)
	if err != nil {
		l.Errorf("error getting message %s: %v", id, err)
		return nil, "", err
	}
	fn := id
	// check if data has optional file metadata prefix
	// format:
	// file:<filename>|<[]byte of file data>
	// get first 4 bytes of data to check if it is a file
	var firstFileByte int
	var mtype string
	mtype = "message"
	if len(m.Data) > 4 {
		ff := m.Data[:4]
		if string(ff) == "file" {
			var nfn string
			// get value between "file:" and "|"
			for i := 5; i < len(m.Data); i++ {
				if m.Data[i] == '|' {
					nfn = string(m.Data[5:i])
					firstFileByte = i + 1
					m.Data = m.Data[firstFileByte:]
					break
				}
			}
			if nfn != "" {
				fn = nfn
				mtype = "file"
			}
		}
	}
	m.Type = mtype
	return m, fn, nil
}

func getMessageWorker(jobs chan GetJob, res chan error) {
	l := log.WithFields(log.Fields{
		"pkg": "agent",
		"fn":  "getMessageWorker",
	})
	for job := range jobs {
		l.Debugf("getting message %s", job.ID)
		m, fn, err := getMessageData(job.Channel, job.ID)
		if err != nil {
			l.Errorf("error getting message %s: %v", job.ID, err)
			res <- err
			continue
		}
		if err := persist.StoreAgentMessage(job.Channel, fn, m.Type, m.Data); err != nil {
			l.Errorf("error storing message %s: %v", job.ID, err)
			res <- err
			continue
		}
		if err := ConfirmMessageReceive(job.Channel, m.ID); err != nil {
			l.Errorf("error confirming message %s: %v", job.ID, err)
			res <- err
			continue
		}
		res <- nil
	}
}

func Agent() error {
	l := log.WithFields(log.Fields{
		"pkg": "agent",
		"fn":  "agent",
	})
	l.Debug("agent")
	go EnsureWatcher()
	for {
		if len(ServerAddrs) == 0 {
			l.Error("no server addresses")
			time.Sleep(time.Second * 10)
			continue
		}
		msgs, err := CheckPendingMessages(DefaultChannel)
		if err != nil {
			l.Errorf("error checking pending messages: %v", err)
			time.Sleep(time.Second * 10)
			continue
		}
		if len(msgs) == 0 {
			l.Debug("no pending messages")
			time.Sleep(time.Second * 10)
			continue
		}
		l.Debugf("pending messages: %v", msgs)
		jobs := make(chan GetJob, len(msgs))
		res := make(chan error, len(msgs))
		for i := 0; i < 10; i++ {
			go getMessageWorker(jobs, res)
		}
		for _, m := range msgs {
			j := GetJob{
				Channel: DefaultChannel,
				ID:      m.ID,
			}
			jobs <- j
		}
		for i := 0; i < len(msgs); i++ {
			err := <-res
			if err != nil {
				l.Errorf("error getting message: %v", err)
				continue
			}
		}
		l.Debug("got all messages")
		time.Sleep(time.Second * 10)
	}
}

func Client() error {
	l := log.WithFields(log.Fields{
		"pkg": "agent",
		"fn":  "Client",
	})
	l.Debug("client")
	if len(ServerAddrs) == 0 {
		l.Error("no server addresses")
		return fmt.Errorf("no server addresses")
	}
	var action string
	// get action from command line
	// second arg is action
	if len(os.Args) > 2 {
		action = os.Args[2]
	}
	l.Debugf("action: %s", action)
	switch action {
	case "confirm":
		return ConfirmMessageReceive(DefaultChannel, ClientMessageID)
	case "get":
		return getMessage(DefaultChannel, ClientMessageID, Output)
	case "list":
		return listMessages(DefaultChannel, OutputFormat, Output)
	case "send":
		return sendMessageFromInput()
	default:
		return fmt.Errorf("unknown action: %s", action)
	}
}

func LoadPrivateKey(key []byte) error {
	l := log.WithFields(log.Fields{
		"pkg": "agent",
		"fn":  "LoadPrivateKey",
	})
	l.Debug("loading private key")
	k, err := keys.BytesToPrivKey(key)
	if err != nil {
		l.Errorf("error loading private key: %v", err)
		return err
	}
	PrivateKey = k
	return nil
}

func LoadPrivateKeyFromFile(file string) error {
	l := log.WithFields(log.Fields{
		"pkg": "agent",
		"fn":  "LoadPrivateKeyFromFile",
	})
	l.Debug("loading private key from file")
	fd, err := ioutil.ReadFile(file)
	if err != nil {
		l.Errorf("error loading private key from file: %v", err)
		return err
	}
	return LoadPrivateKey(fd)
}

func GetNextAgentServer() string {
	if lastServer+1 >= len(ServerAddrs) {
		return ServerAddrs[0]
	}
	lastServer = lastServer + 1
	return ServerAddrs[lastServer]
}

func GetAgentServer() string {
	return ServerAddrs[lastServer]
}

func CreateSignature() (string, string, error) {
	l := log.WithFields(log.Fields{
		"pkg": "agent",
		"fn":  "CreateSignature",
	})
	l.Debug("creating signature")
	ts := time.Now().Unix()
	var td struct {
		Timestamp int64 `json:"timestamp"`
	}
	var sigReq struct {
		PublicKey []byte `json:"public_key"`
		Data      []byte `json:"data"`
		Signature []byte `json:"signature"`
	}
	td.Timestamp = ts
	jd, err := json.Marshal(td)
	if err != nil {
		l.Errorf("error marshalling timestamp: %v", err)
		return "", "", err
	}
	l.Debugf("timestamp: %s", string(jd))
	sig, err := sign.Sign(jd, PrivateKey)
	if err != nil {
		l.Errorf("error creating signature: %v", err)
		return "", "", err
	}
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&PrivateKey.PublicKey)
	if err != nil {
		fmt.Printf("error when dumping publickey: %s \n", err)
		os.Exit(1)
	}
	publicKeyBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}
	publicKeyPem := pem.EncodeToMemory(publicKeyBlock)
	sigReq.PublicKey = publicKeyPem
	//log.Printf("public key: %s", publicKeyBlock)
	sigReq.Data = jd
	sigReq.Signature = sig
	j, err := json.Marshal(sigReq)
	if err != nil {
		l.Errorf("error marshalling signature request: %v", err)
		return "", "", err
	}
	keyID := keys.PubKeyID(publicKeyPem)
	l.Debugf("key ID: %s", keyID)
	return base64.StdEncoding.EncodeToString(j), keyID, nil
}

func CheckPendingMessages(channel string) ([]MessageMeta, error) {
	l := log.WithFields(log.Fields{
		"pkg": "agent",
		"fn":  "CheckPendingMessages",
		"ch":  channel,
	})
	l.Debug("checking pending messages")
	var msgs []MessageMeta
	saddr := GetAgentServer()
	c := &http.Client{}
	sig, keyID, err := CreateSignature()
	if err != nil {
		l.Errorf("error creating signature: %v", err)
		return msgs, err
	}
	addr := saddr + "/message/" + keyID + "/meta"
	if channel != "" {
		addr = addr + "?channel=" + channel
	}
	req, err := http.NewRequest("LIST", addr, nil)
	if err != nil {
		l.Errorf("error creating request: %v", err)
		return msgs, err
	}
	req.Header.Set("X-Signature", sig)
	if ServerAuthToken != "" {
		req.Header.Set("X-Token", ServerAuthToken)
	}
	resp, err := c.Do(req)
	if err != nil {
		l.Errorf("error sending request: %v", err)
		return msgs, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		l.Errorf("error checking pending messages: %v", resp.StatusCode)
		return msgs, err
	}
	bd, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		l.Errorf("error reading response: %v", err)
		return msgs, err
	}

	err = json.Unmarshal(bd, &msgs)
	if err != nil {
		l.Errorf("error unmarshalling response: %v", err)
		return msgs, err
	}
	return msgs, nil
}

func GetMessage(channel, id string) (*message.Message, error) {
	l := log.WithFields(log.Fields{
		"pkg": "agent",
		"fn":  "GetMessage",
		"id":  id,
		"ch":  channel,
	})
	l.Debug("getting message")
	saddr := GetAgentServer()
	c := &http.Client{}
	sig, keyID, err := CreateSignature()
	if err != nil {
		l.Errorf("error creating signature: %v", err)
		return nil, err
	}
	addr := saddr + "/message/" + keyID + "/" + channel + "/" + id
	req, err := http.NewRequest("GET", addr, nil)
	if err != nil {
		l.Errorf("error creating request: %v", err)
		return nil, err
	}
	req.Header.Set("X-Signature", sig)
	if ServerAuthToken != "" {
		req.Header.Set("X-Token", ServerAuthToken)
	}
	resp, err := c.Do(req)
	if err != nil {
		l.Errorf("error sending request: %v", err)
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		l.Errorf("error getting message: %v", resp.StatusCode)
		return nil, err
	}
	bd, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		l.Errorf("error reading response: %v", err)
		return nil, err
	}
	m := &message.Message{
		ID:          id,
		Channel:     channel,
		PublicKeyID: keyID,
		Data:        bd,
	}
	return m, nil
}

func ConfirmMessageReceive(channel, id string) error {
	l := log.WithFields(log.Fields{
		"pkg": "agent",
		"fn":  "ConfirmMessageReceive",
	})
	l.Debug("confirming message receive")
	saddr := GetAgentServer()
	c := &http.Client{}
	sig, keyID, err := CreateSignature()
	if err != nil {
		l.Errorf("error creating signature: %v", err)
		return err
	}
	if channel == "" || id == "" {
		l.Errorf("error: channel or id is empty")
		return errors.New("error: channel or id is empty")
	}
	addr := saddr + "/message/" + keyID + "/" + channel + "/" + id
	req, err := http.NewRequest("DELETE", addr, nil)
	if err != nil {
		l.Errorf("error creating request: %v", err)
		return err
	}
	req.Header.Set("X-Signature", sig)
	if ServerAuthToken != "" {
		req.Header.Set("X-Token", ServerAuthToken)
	}
	resp, err := c.Do(req)
	if err != nil {
		l.Errorf("error sending request: %v", err)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		l.Errorf("error confirming message receive: %v", resp.StatusCode)
		return err
	}
	return nil
}

func DecryptMessageData(m *message.Message) (*message.Message, error) {
	l := log.WithFields(log.Fields{
		"pkg": "agent",
		"fn":  "DecryptMessageData",
	})
	l.Debug("decrypting message data")
	priv := x509.MarshalPKCS1PrivateKey(PrivateKey)
	kb := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: priv,
	}
	kbp := pem.EncodeToMemory(kb)
	decrypted, err := keys.DecryptMessage(kbp, strings.TrimSpace(string(m.Data)))
	if err != nil {
		l.Errorf("error decrypting message data: %v", err)
		return m, err
	}
	m.Data = decrypted
	l.Debugf("decrypted message data: %s", m.Data)
	return m, nil
}
