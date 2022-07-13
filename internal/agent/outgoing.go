package agent

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/robertlestak/centauri/internal/persist"
	"github.com/robertlestak/centauri/pkg/message"
	log "github.com/sirupsen/logrus"
)

var (
	// PendingOutgoingMessages is a map of pending outgoing messages with the time the file last changed on system
	// this is a poor-man's way of ensuring a larger file is not picked up mid-copy.
	PendingOutgoingMessages []string
	PendingOutgoingFiles    []string
)

func GetOutgoingMessages() ([]string, error) {
	l := log.WithFields(log.Fields{
		"pkg": "agent",
		"fn":  "GetOutgoingMessages",
	})
	l.Debug("getting outgoing messages")
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
	l.Debug("getting outgoing files")
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
	l.Debug("starting watcher")
	outMsg, err := GetOutgoingMessages()
	if err != nil {
		l.Errorf("error getting outgoing messages: %v", err)
		return err
	}
	if len(outMsg) == 0 {
		l.Debug("no outgoing messages")
	}
	outFile, err := GetOutgoingFiles()
	if err != nil {
		l.Errorf("error getting outgoing files: %v", err)
		return err
	}
	if len(outFile) == 0 {
		l.Debug("no outgoing files")
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

func handleOutgoingFile(fp string, channel, pubKeyID, id string) error {
	l := log.WithFields(log.Fields{
		"pkg": "agent",
		"fn":  "handleOutgoingFile",
		"fp":  fp,
		"key": pubKeyID,
		"id":  id,
	})
	l.Debug("handling outgoing file")
	f, err := os.Open(fp)
	if err != nil {
		l.Errorf("error opening file: %v", err)
		return err
	}
	m, err := message.CreateMessage("file", id, channel, pubKeyID, f)
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
	l.Debug("handling outgoing files")

	for _, file := range files {
		for _, fp := range PendingOutgoingFiles {
			if fp == file {
				return nil
			}
		}
		PendingOutgoingFiles = append(PendingOutgoingFiles, file)
	}
	return nil
}

func removeFromSlice(s []string, s2 string) []string {
	for i, v := range s {
		if v == s2 {
			return append(s[:i], s[i+1:]...)
		}
	}
	return s
}

func outgoingFileWorker() {
	l := log.WithFields(log.Fields{
		"pkg": "agent",
		"fn":  "outgoingFileWorker",
	})
	l.Debug("starting outgoing file worker")
	for {
		if len(PendingOutgoingFiles) == 0 {
			l.Debug("no pending outgoing files")
			time.Sleep(time.Second * 10)
			continue
		}
		l.Debugf("pending outgoing files: %v", PendingOutgoingFiles)
		for _, fp := range PendingOutgoingFiles {
			// check time file modified
			fi, err := os.Stat(fp)
			if err != nil {
				l.Errorf("error getting file info: %v", err)
				continue
			}
			if time.Since(fi.ModTime()) > time.Second*60 {
				l.Debugf("file %s has not changed, uploading", fp)
				dir, fn := filepath.Split(fp)
				key := filepath.Base(dir)
				l.Debugf("handling file %s for key %s", fn, key)
				if err := handleOutgoingFile(fp, DefaultChannel, key, fn); err != nil {
					l.Errorf("error handling outgoing file: %v", err)
					continue
				}
				// remove from pending array
				PendingOutgoingFiles = removeFromSlice(PendingOutgoingFiles, fp)
			}
		}
		time.Sleep(time.Second * 10)
	}
}

func outgoingMessageWorker() {
	l := log.WithFields(log.Fields{
		"pkg": "agent",
		"fn":  "outgoingMessageWorker",
	})
	l.Debug("starting outgoing message worker")
	for {
		if len(PendingOutgoingMessages) == 0 {
			l.Debug("no pending outgoing messages")
			time.Sleep(time.Second * 1)
			continue
		}
		l.Debugf("pending outgoing messages: %v", PendingOutgoingMessages)
		for _, fp := range PendingOutgoingMessages {
			// check time file modified
			fi, err := os.Stat(fp)
			if err != nil {
				l.Errorf("error getting file info: %v", err)
				continue
			}
			if time.Since(fi.ModTime()) > time.Second*10 {
				l.Debugf("file %s has not changed, uploading", fp)
				dir, fn := filepath.Split(fp)
				key := filepath.Base(dir)
				l.Debugf("handling file %s for key %s", fn, key)
				if err := handleOutgoingMessage(fp, key, fn); err != nil {
					l.Errorf("error handling outgoing file: %v", err)
					continue
				}
				// remove from pending array
				PendingOutgoingMessages = removeFromSlice(PendingOutgoingMessages, fp)
			}
		}
		time.Sleep(time.Second * 1)
	}
}

func handleOutgoingMessage(fp, pubKeyID, id string) error {
	l := log.WithFields(log.Fields{
		"pkg": "agent",
		"fn":  "handleOutgoingMessage",
		"fp":  fp,
		"key": pubKeyID,
		"id":  id,
	})
	l.Debug("handling outgoing message")
	f, err := os.Open(fp)
	if err != nil {
		l.Errorf("error opening file: %v", err)
		return err
	}
	m, err := message.CreateMessage("bytes", "", DefaultChannel, pubKeyID, f)
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
	l.Debug("handling outgoing messages")
	for _, m := range msgs {
		for _, mp := range PendingOutgoingMessages {
			if mp == m {
				return nil
			}
		}
		PendingOutgoingMessages = append(PendingOutgoingMessages, m)
	}
	return nil
}

func EnsureWatcher() error {
	l := log.WithFields(log.Fields{
		"pkg": "agent",
		"fn":  "EnsureWatcher",
	})
	l.Debug("ensuring outgoing watcher")
	go outgoingFileWorker()
	go outgoingMessageWorker()
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
	l.Debug("sending message through peer")
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
	if resp.StatusCode != http.StatusAccepted {
		l.Errorf("error confirming message receive: %v", resp.StatusCode)
		return err
	}
	return nil
}

func sendMessage(channel, pubKeyID, mType, fn string, data io.ReadCloser) error {
	l := log.WithFields(log.Fields{
		"pkg": "agent",
		"fn":  "sendMessage",
		"key": pubKeyID,
		"id":  fn,
	})
	l.Debug("sending message")
	m, err := message.CreateMessage(mType, fn, channel, pubKeyID, data)
	if err != nil {
		l.Errorf("error creating message: %v", err)
		return err
	}
	if err := SendMessageThroughPeer(m); err != nil {
		l.Errorf("error sending message: %v", err)
		return err
	}
	return nil
}
