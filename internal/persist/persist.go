package persist

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

var (
	RootDataDir              string
	NodeDataDir              string
	MessagesDir              string
	AgentMessagesDir         string
	AgentFilesDir            string
	AgentPubKeyChainDir      string
	AgentOutgoingDir         string
	AgentOutgoingFilesDir    string
	AgentOutgoingMessagesDir string
)

type MessageMetaData struct {
	ID       string
	PubKeyID string
	Size     int64
}

func EnsureNodeDataDir(name string) error {
	l := log.WithFields(log.Fields{
		"pkg": "persist",
		"fn":  "EnsureNodeDataDir",
	})
	l.Info("ensuring node data dir")
	NodeDataDir = RootDataDir + "/" + name
	if _, err := os.Stat(NodeDataDir); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(NodeDataDir, 0755); err != nil {
				l.Errorf("failed to create node data dir: %v", err)
				return err
			}
		}
	}
	return nil
}

func EnsureAgentMessagesDir() error {
	l := log.WithFields(log.Fields{
		"pkg": "persist",
		"fn":  "EnsureAgentMessagesDir",
	})
	l.Info("ensuring agent messages dir")
	AgentMessagesDir = RootDataDir + "/" + "received/messages"
	//AgentMessagesDir = RootDataDir
	if _, err := os.Stat(AgentMessagesDir); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(AgentMessagesDir, 0755); err != nil {
				l.Errorf("failed to create agent messages dir: %v", err)
				return err
			}
		}
	}
	return nil
}

func EnsureAgentFilesDir() error {
	l := log.WithFields(log.Fields{
		"pkg": "persist",
		"fn":  "EnsureAgentFilesDir",
	})
	l.Info("ensuring agent files dir")
	AgentFilesDir = RootDataDir + "/" + "received/files"
	//AgentMessagesDir = RootDataDir
	if _, err := os.Stat(AgentFilesDir); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(AgentFilesDir, 0755); err != nil {
				l.Errorf("failed to create agent messages dir: %v", err)
				return err
			}
		}
	}
	return nil
}

func EnsureAgentOutgoingDir() error {
	l := log.WithFields(log.Fields{
		"pkg": "persist",
		"fn":  "EnsureAgentOutgoingDir",
	})
	l.Info("ensuring agent messages dir")
	AgentOutgoingDir = RootDataDir + "/" + "outgoing"
	//AgentMessagesDir = RootDataDir
	if _, err := os.Stat(AgentOutgoingDir); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(AgentOutgoingDir, 0755); err != nil {
				l.Errorf("failed to create agent messages dir: %v", err)
				return err
			}
		}
	}
	AgentOutgoingFilesDir = AgentOutgoingDir + "/files"
	if _, err := os.Stat(AgentOutgoingFilesDir); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(AgentOutgoingFilesDir, 0755); err != nil {
				l.Errorf("failed to create agent messages dir: %v", err)
				return err
			}
		}
	}
	AgentOutgoingMessagesDir = AgentOutgoingDir + "/messages"
	if _, err := os.Stat(AgentOutgoingMessagesDir); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(AgentOutgoingMessagesDir, 0755); err != nil {
				l.Errorf("failed to create agent messages dir: %v", err)
				return err
			}
		}
	}
	return nil
}

func EnsureMessagesDir() error {
	l := log.WithFields(log.Fields{
		"pkg": "persist",
		"fn":  "EnsureMessagesDir",
	})
	l.Info("ensuring messages dir")
	MessagesDir = NodeDataDir + "/messages"
	if _, err := os.Stat(MessagesDir); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(MessagesDir, 0755); err != nil {
				l.Errorf("failed to create messages dir: %v", err)
				return err
			}
		}
	}
	return nil
}

func EnsurePubKeyDir(pubKeyID string) (string, error) {
	l := log.WithFields(log.Fields{
		"pkg": "persist",
		"fn":  "EnsurePubKeyDir",
	})
	l.Info("ensuring pub key dir")
	dir := PubKeyMessageDir(pubKeyID)
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0755); err != nil {
				l.Errorf("failed to create pub key dir: %v", err)
				return dir, err
			}
		}
	}
	return dir, nil
}

func EnsureAgentPubKeyChainDir() error {
	l := log.WithFields(log.Fields{
		"pkg": "persist",
		"fn":  "EnsureAgentPubKeyChainDir",
	})
	l.Info("ensuring pub key dir")
	dir := RootDataDir + "/pubkeys"
	AgentPubKeyChainDir = dir
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0755); err != nil {
				l.Errorf("failed to create pub key dir: %v", err)
				return err
			}
		}
	}
	return nil
}

func EnsurePubKeyChainDir(pubKeyID string) (string, error) {
	l := log.WithFields(log.Fields{
		"pkg": "persist",
		"fn":  "EnsurePubKeyChainDir",
	})
	l.Info("ensuring pub key dir")
	dir := PubKeyChainDir(pubKeyID)
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0755); err != nil {
				l.Errorf("failed to create pub key dir: %v", err)
				return dir, err
			}
		}
	}
	return dir, nil
}

func EnsurePubKeyChainOutgoingDir(pubKeyID string) (string, error) {
	l := log.WithFields(log.Fields{
		"pkg": "persist",
		"fn":  "EnsurePubKeyChainOutgoingDir",
	})
	l.Info("ensuring pub key dir")
	dir := AgentOutgoingFilesDir + "/" + pubKeyID
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0755); err != nil {
				l.Errorf("failed to create pub key dir: %v", err)
				return dir, err
			}
		}
	}
	l.Infof("outgoing files dir: %s", dir)
	dir = AgentOutgoingMessagesDir + "/" + pubKeyID
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0755); err != nil {
				l.Errorf("failed to create pub key dir: %v", err)
				return dir, err
			}
		}
	}
	l.Infof("outgoing messages dir: %s", dir)
	return dir, nil
}

func RemovePubKeyChainOutgoingDir(pubKeyID string) (string, error) {
	l := log.WithFields(log.Fields{
		"pkg": "persist",
		"fn":  "RemovePubKeyChainOutgoingDir",
	})
	l.Info("removing pub key dir")
	dir := AgentOutgoingFilesDir + "/" + pubKeyID
	// delete the directory and all of its contents
	if err := os.RemoveAll(dir); err != nil {
		l.Errorf("failed to remove pub key dir: %v", err)
		return dir, err
	}
	dir = AgentOutgoingMessagesDir + "/" + pubKeyID
	// delete the directory and all of its contents
	if err := os.RemoveAll(dir); err != nil {
		l.Errorf("failed to remove pub key dir: %v", err)
		return dir, err
	}
	return dir, nil
}

func Init(rootDataDir, nodeName string) error {
	l := log.WithFields(log.Fields{
		"pkg": "persist",
		"fn":  "init",
	})
	l.Info("initializing")
	RootDataDir = rootDataDir
	if err := EnsureNodeDataDir(nodeName); err != nil {
		l.Errorf("failed to ensure node data dir: %v", err)
		return err
	}
	if err := EnsureMessagesDir(); err != nil {
		l.Errorf("failed to ensure messages dir: %v", err)
		return err
	}
	return nil
}

func InitAgent(rootDataDir string) error {
	l := log.WithFields(log.Fields{
		"pkg": "persist",
		"fn":  "InitAgent",
	})
	l.Info("initializing")
	RootDataDir = rootDataDir
	if err := EnsureAgentMessagesDir(); err != nil {
		l.Errorf("failed to ensure node data dir: %v", err)
		return err
	}
	if err := EnsureAgentFilesDir(); err != nil {
		l.Errorf("failed to ensure node data dir: %v", err)
		return err
	}
	if err := EnsureAgentOutgoingDir(); err != nil {
		l.Errorf("failed to ensure node data dir: %v", err)
		return err
	}
	if err := EnsureAgentPubKeyChainDir(); err != nil {
		l.Errorf("failed to ensure node data dir: %v", err)
		return err
	}
	return nil
}

func PubKeyMessageDir(pubKeyID string) string {
	return MessagesDir + "/" + pubKeyID
}

func PubKeyChainDir(pubKeyID string) string {
	return AgentPubKeyChainDir + "/" + pubKeyID
}

func StoreMessage(pubKeyID string, id string, data []byte) error {
	l := log.WithFields(log.Fields{
		"pkg": "persist",
		"fn":  "StoreMessage",
	})
	l.Info("storing message")
	dir, err := EnsurePubKeyDir(pubKeyID)
	if err != nil {
		l.Errorf("failed to ensure pub key dir: %v", err)
		return err
	}
	file := dir + "/" + id
	if err := ioutil.WriteFile(file, data, 0644); err != nil {
		l.Errorf("failed to write message: %v", err)
		return err
	}
	return nil
}

func ListMessageMetaForPubKeyID(pubKeyID string) ([]MessageMetaData, error) {
	l := log.WithFields(log.Fields{
		"pkg": "persist",
		"fn":  "ListMessageMetaForPubKeyID",
	})
	l.Info("listing messages for pub key id")
	var md []MessageMetaData
	dir := PubKeyMessageDir(pubKeyID)
	// check if dir exists
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return md, nil
		}
		l.Errorf("failed to stat dir: %v", err)
		return nil, err
	}
	// loop files in dir and get metadata
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		l.Errorf("failed to read dir: %v", err)
		return nil, err
	}
	for _, file := range files {
		md = append(md, MessageMetaData{
			ID:       file.Name(),
			PubKeyID: pubKeyID,
			Size:     file.Size(),
		})
	}
	return md, nil
}

func GetMessageByID(pubKeyID string, id string) ([]byte, error) {
	l := log.WithFields(log.Fields{
		"pkg": "persist",
		"fn":  "GetMessageByID",
	})
	l.Info("getting message by id")
	dir := PubKeyMessageDir(pubKeyID)
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New("message not found")
		}
		l.Errorf("failed to stat dir: %v", err)
		return nil, err
	}
	file := dir + "/" + id
	// check if file exists
	if _, err := os.Stat(file); err != nil {
		if os.IsNotExist(err) {
			l.Errorf("message does not exist: %v", err)
			return nil, errors.New("message does not exist")
		}
	}
	data, err := ioutil.ReadFile(file)
	if err != nil {
		l.Errorf("failed to read file: %v", err)
		return nil, err
	}
	return data, nil
}

func DeleteDirIfEmpty(dir string) error {
	l := log.WithFields(log.Fields{
		"pkg": "persist",
		"fn":  "DeleteDirIfEmpty",
	})
	l.Info("deleting dir if empty")
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		l.Errorf("failed to read dir: %v", err)
		return err
	}
	if len(files) == 0 {
		if err := os.Remove(dir); err != nil {
			l.Errorf("failed to remove dir: %v", err)
			return err
		}
	}
	return nil
}

func DeleteMessageByID(pubKeyID string, id string) error {
	l := log.WithFields(log.Fields{
		"pkg": "persist",
		"fn":  "DeleteMessageByID",
	})
	l.Info("deleting message by id")
	dir := PubKeyMessageDir(pubKeyID)
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return errors.New("message not found")
		}
		l.Errorf("failed to stat dir: %v", err)
		return err
	}
	file := dir + "/" + id
	if _, err := os.Stat(file); err != nil {
		if os.IsNotExist(err) {
			l.Errorf("message does not exist: %v", err)
			return errors.New("message does not exist")
		}
	}
	if err := os.Remove(file); err != nil {
		l.Errorf("failed to delete file: %v", err)
		return err
	}
	return DeleteDirIfEmpty(dir)
}

func cleanupOldFiles(dur time.Duration) error {
	l := log.WithFields(log.Fields{
		"pkg": "persist",
		"fn":  "cleanupOldFiles",
	})
	l.Info("cleaning up old files")
	// loop through MessagesDir recursivevely
	// the directory name is the pubKeyID and the file name is the messageID
	// if the file is older than dur, delete it
	deletions := make(map[string][]string)
	// walk file tree
	err := filepath.Walk(MessagesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			l.Errorf("failed to walk file tree: %v", err)
			return err
		}
		// if file is a directory, it is a pubKeyID
		if info.IsDir() {
			// get pubKeyID
			pubKeyID := filepath.Base(path)
			// get list of files in pubKeyID
			files, err := ioutil.ReadDir(path)
			if err != nil {
				l.Errorf("failed to read dir: %v", err)
				return err
			}
			// loop through files in pubKeyID
			for _, file := range files {
				// get messageID
				messageID := file.Name()
				// check if file is older than dur
				if time.Since(file.ModTime()) > dur {
					// delete file
					deletions[pubKeyID] = append(deletions[pubKeyID], messageID)
				}
			}
		}
		return nil
	})
	if err != nil {
		l.Errorf("failed to walk file tree: %v", err)
		return err
	}
	// delete files
	for pubKeyID, messageIDs := range deletions {
		for _, messageID := range messageIDs {
			err := DeleteMessageByID(pubKeyID, messageID)
			if err != nil {
				l.Errorf("failed to delete message: %v", err)
				return err
			}
		}
	}
	return nil
}

func TimeoutCleaner() {
	l := log.WithFields(log.Fields{
		"pkg": "persist",
		"fn":  "TimeoutCleaner",
	})
	l.Info("timeout cleaner started")
	for {
		time.Sleep(time.Hour * 24)
		l.Info("cleaning")
		if err := cleanupOldFiles(time.Hour * 24 * 90); err != nil {
			l.Errorf("failed to clean: %v", err)
		}
	}
}

func StoreAgentMessage(name string, mtype string, data []byte) error {
	l := log.WithFields(log.Fields{
		"pkg": "persist",
		"fn":  "StoreAgentMessage",
	})
	l.Info("storing agent file")
	var dir string
	switch mtype {
	case "message":
		dir = AgentMessagesDir
	case "file":
		dir = AgentFilesDir
	default:
		l.Errorf("invalid message type: %v", mtype)
		return errors.New("invalid message type")
	}
	file := dir + "/" + name
	// if file exists, append guid to new file name
	if _, err := os.Stat(file); err == nil {
		guid := uuid.New().String()
		file = file + "_" + guid
	}
	if err := ioutil.WriteFile(file, data, 0644); err != nil {
		l.Errorf("failed to write file: %v", err)
		return err
	}
	return nil
}
