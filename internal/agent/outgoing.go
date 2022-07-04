package agent

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/robertlestak/mp/internal/persist"
	"github.com/robertlestak/mp/pkg/message"
	log "github.com/sirupsen/logrus"
)

func GetOutgoingMessages() ([]string, error) {
	l := log.WithFields(log.Fields{
		"pkg": "agent",
		"fn":  "GetOutgoingMessages",
	})
	l.Info("getting outgoing messages")
	// get all files in dataDir + outgoing/messages
	// return the file paths as a slice
	files, err := filepath.Glob(filepath.Join(persist.RootDataDir, "outgoing", "messages", "*/*"))
	if err != nil {
		l.Errorf("error getting outgoing messages: %v", err)
		return nil, err
	}
	return files, nil
}

func GetOutgoingFiles() ([]string, error) {
	l := log.WithFields(log.Fields{
		"pkg": "agent",
		"fn":  "GetOutgoingFies",
	})
	l.Info("getting outgoing files")
	// get all files in dataDir + outgoing/files
	// return the file paths as a slice
	files, err := filepath.Glob(filepath.Join(persist.RootDataDir, "outgoing", "files", "*/*"))
	if err != nil {
		l.Errorf("error getting outgoing files: %v", err)
		return nil, err
	}
	return files, nil
}

func StartWatcher() error {
	l := log.WithFields(log.Fields{
		"pkg": "agent",
		"fn":  "StartWatcher",
	})
	l.Info("starting watcher")
	outMsg, err := GetOutgoingMessages()
	if err != nil {
		l.Errorf("error getting outgoing messages: %v", err)
		return err
	}
	if len(outMsg) == 0 {
		l.Info("no outgoing messages")
	}
	outFile, err := GetOutgoingFiles()
	if err != nil {
		l.Errorf("error getting outgoing files: %v", err)
		return err
	}
	if len(outFile) == 0 {
		l.Info("no outgoing files")
	}
	if err := handleOutgoingMessages(outMsg); err != nil {
		l.Errorf("error handling outgoing messages: %v", err)
		return err
	}
	if err := handleOutgoingFiles(outFile); err != nil {
		l.Errorf("error handling outgoing files: %v", err)
		return err
	}
	return nil
}

func handleOutgoingFile(fp string, pubKeyID, id string) error {
	l := log.WithFields(log.Fields{
		"pkg": "agent",
		"fn":  "handleOutgoingFile",
		"fp":  fp,
		"key": pubKeyID,
		"id":  id,
	})
	l.Info("handling outgoing file")
	f, err := os.Open(fp)
	if err != nil {
		l.Errorf("error opening file: %v", err)
		return err
	}
	m, err := message.CreateMessage("file", id, pubKeyID, f)
	if err != nil {
		l.Errorf("error creating message: %v", err)
		return err
	}
	if err := SendMessageThroughPeer(m); err != nil {
		l.Errorf("error sending message: %v", err)
		return err
	}
	// delete file
	if err := os.Remove(fp); err != nil {
		l.Errorf("error removing file: %v", err)
		return err
	}
	return nil
}

func handleOutgoingFiles(files []string) error {
	l := log.WithFields(log.Fields{
		"pkg": "agent",
		"fn":  "handleOutgoingFiles",
	})
	l.Info("handling outgoing files")
	for _, file := range files {
		dir, fn := filepath.Split(file)
		key := filepath.Base(dir)
		l.Infof("handling file %s for key %s", fn, key)
		if err := handleOutgoingFile(file, key, fn); err != nil {
			l.Errorf("error handling outgoing file: %v", err)
			return err
		}
	}
	return nil
}

func handleOutgoingMessage(fp, pubKeyID, id string) error {
	l := log.WithFields(log.Fields{
		"pkg": "agent",
		"fn":  "handleOutgoingMessage",
		"fp":  fp,
		"key": pubKeyID,
		"id":  id,
	})
	l.Info("handling outgoing message")
	f, err := os.Open(fp)
	if err != nil {
		l.Errorf("error opening file: %v", err)
		return err
	}
	m, err := message.CreateMessage("message", "", pubKeyID, f)
	if err != nil {
		l.Errorf("error creating message: %v", err)
		return err
	}
	if err := SendMessageThroughPeer(m); err != nil {
		l.Errorf("error sending message: %v", err)
		return err
	}
	// delete file
	if err := os.Remove(fp); err != nil {
		l.Errorf("error removing file: %v", err)
		return err
	}
	return nil
}

func handleOutgoingMessages(msgs []string) error {
	l := log.WithFields(log.Fields{
		"pkg": "agent",
		"fn":  "handleOutgoingMessages",
	})
	l.Info("handling outgoing messages")
	for _, msg := range msgs {
		dir, fn := filepath.Split(msg)
		key := filepath.Base(dir)
		l.Infof("handling message %s for key %s", fn, key)
		if err := handleOutgoingMessage(msg, key, fn); err != nil {
			l.Errorf("error handling outgoing message: %v", err)
			return err
		}
	}
	return nil
}

func EnsureWatcher() error {
	l := log.WithFields(log.Fields{
		"pkg": "agent",
		"fn":  "EnsureWatcher",
	})
	l.Info("ensuring outgoing watcher")
	for {
		err := StartWatcher()
		if err != nil {
			l.Errorf("failed to start watcher: %v", err)
		}
		time.Sleep(time.Second * 10)
	}
}

func SendMessageThroughPeer(msg *message.Message) error {
	l := log.WithFields(log.Fields{
		"pkg":           "agent",
		"fn":            "SendMessageThroughPeer",
		"m.PublicKeyID": msg.PublicKeyID,
	})
	l.Info("sending message through peer")
	saddr := GetAgentServer()
	c := &http.Client{}
	jd, err := json.Marshal(msg)
	if err != nil {
		l.Errorf("error marshalling message: %v", err)
		return err
	}
	addr := saddr + "/message"
	req, err := http.NewRequest("POST", addr, bytes.NewReader(jd))
	if err != nil {
		l.Errorf("error creating request: %v", err)
		return err
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
	if err := json.NewDecoder(resp.Body).Decode(&msg); err != nil {
		l.Errorf("error decoding response: %v", err)
		return err
	}
	l.Infof("message sent: %v", msg.ID)
	return nil
}
