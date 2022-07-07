package persist

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
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
	ID        string    `json:"id"`
	Channel   string    `json:"channel"`
	PubKeyID  string    `json:"pubkey_id"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
}

func PubKeyMessageDir(pubKeyID string) string {
	return MessagesDir + "/" + pubKeyID
}

func PubKeyChainDir(pubKeyID string) string {
	return AgentPubKeyChainDir + "/" + pubKeyID
}

func StoreMessage(pubKeyID string, channel string, id string, data []byte) error {
	l := log.WithFields(log.Fields{
		"pkg": "persist",
		"fn":  "StoreMessage",
	})
	l.Debug("storing message")
	dir := PubKeyMessageDir(pubKeyID)
	if channel == "" {
		channel = "default"
	}
	dir = dir + "/" + channel
	if err := EnsureDir(dir); err != nil {
		l.Errorf("failed to create dir: %v", err)
		return err
	}
	file := dir + "/" + id
	if err := ioutil.WriteFile(file, data, 0644); err != nil {
		l.Errorf("failed to write message: %v", err)
		return err
	}
	return nil
}

func ListMessageMetaForPubKeyID(pubKeyID string, channel string) ([]MessageMetaData, error) {
	l := log.WithFields(log.Fields{
		"pkg":      "persist",
		"fn":       "ListMessageMetaForPubKeyID",
		"pubKeyID": pubKeyID,
		"channel":  channel,
	})
	l.Debug("listing messages for pub key id")
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
	var chanselect string
	if channel == "" {
		chanselect = "*"
	} else {
		chanselect = channel
	}
	files, err := filepath.Glob(dir + "/" + chanselect + "/*")
	if err != nil {
		l.Errorf("failed to glob dir: %v", err)
		return nil, err
	}
	// for each file, the file name is the message id
	// and the parent dir is the channel
	for _, file := range files {
		id := filepath.Base(file)
		channel := filepath.Base(filepath.Dir(file))
		// get file size
		if stat, err := os.Stat(file); err != nil {
			l.Errorf("failed to stat file: %v", err)
			return nil, err
		} else {
			size := stat.Size()
			md = append(md, MessageMetaData{
				ID:        id,
				PubKeyID:  pubKeyID,
				Size:      size,
				Channel:   channel,
				CreatedAt: stat.ModTime(),
			})
		}
	}
	return md, nil
}

func GetMessageByID(pubKeyID string, channel string, id string) ([]byte, error) {
	l := log.WithFields(log.Fields{
		"pkg": "persist",
		"fn":  "GetMessageByID",
	})
	l.Debug("getting message by id")
	dir := PubKeyMessageDir(pubKeyID)
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New("message not found")
		}
		l.Errorf("failed to stat dir: %v", err)
		return nil, err
	}
	if channel == "" {
		channel = "default"
	}
	file := dir + "/" + channel + "/" + id
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

func StoreAgentMessage(channel string, name string, mtype string, data []byte) error {
	l := log.WithFields(log.Fields{
		"pkg": "persist",
		"fn":  "StoreAgentMessage",
	})
	l.Debug("storing agent file")
	var dir string
	switch mtype {
	case "bytes":
		dir = AgentMessagesDir
	case "file":
		dir = AgentFilesDir
	default:
		l.Errorf("invalid message type: %v", mtype)
		return errors.New("invalid message type")
	}
	cdir := dir + "/" + channel
	if err := EnsureDir(cdir); err != nil {
		l.Errorf("failed to ensure dir: %v", err)
		return err
	}
	file := cdir + "/" + name
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

func DeleteMessageByID(pubKeyID string, channel string, id string) error {
	l := log.WithFields(log.Fields{
		"pkg": "persist",
		"fn":  "DeleteMessageByID",
	})
	l.Debug("deleting message by id")
	mdir := PubKeyMessageDir(pubKeyID)
	if _, err := os.Stat(mdir); err != nil {
		if os.IsNotExist(err) {
			return errors.New("message not found")
		}
		l.Errorf("failed to stat dir: %v", err)
		return err
	}
	if channel == "" {
		channel = "default"
	}
	file := mdir + "/" + channel + "/" + id
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
	return DeleteDirIfEmpty(mdir + "/" + channel)
}

//  DeleteDirIfEmpty deletes the specified directory if it is empty.
// If the directory is deleted, check the parent directory and delete it if empty.
func DeleteDirIfEmpty(dir string) error {
	l := log.WithFields(log.Fields{
		"pkg": "persist",
		"fn":  "DeleteDirIfEmpty",
		"dir": dir,
	})
	l.Debug("deleting dir if empty")
	// check if dir exists
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		l.Errorf("failed to stat dir: %v", err)
		return err
	}
	// check if dir is empty
	files, err := filepath.Glob(dir + "/*")
	if err != nil {
		l.Errorf("failed to glob dir: %v", err)
		return err
	}
	if len(files) > 0 {
		return nil
	}
	// delete dir
	if err := os.RemoveAll(dir); err != nil {
		l.Errorf("failed to delete dir: %v", err)
		return err
	}
	// check if parent dir is empty
	parent := filepath.Dir(dir)
	if parent == MessagesDir || parent == RootDataDir {
		return nil
	}
	if err := DeleteDirIfEmpty(parent); err != nil {
		l.Errorf("failed to delete parent dir: %v", err)
		return err
	}
	return nil
}

func getFilesOlderThan(dir string, dur time.Duration) ([]string, error) {
	l := log.WithFields(log.Fields{
		"pkg": "persist",
		"fn":  "getFilesOlderThan",
	})
	l.Debug("getting files older than")
	// recurse through dir and get all files older than dur
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			l.Errorf("failed to walk dir: %v", err)
			return err
		}
		if info.IsDir() {
			return nil
		}
		if time.Since(info.ModTime()) > dur {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		l.Errorf("failed to walk dir: %v", err)
		return nil, err
	}
	return files, nil
}

func cleanupOldFiles(dur time.Duration) error {
	l := log.WithFields(log.Fields{
		"pkg": "persist",
		"fn":  "cleanupOldFiles",
	})
	l.Debug("cleaning up old files")
	// loop through MessagesDir recursivevely
	// the directory name is the pubKeyID and the file name is the messageID
	// if the file is older than dur, delete it
	type Deletion struct {
		PubKeyID string
		Channel  string
		ID       string
	}
	deletions := []Deletion{}
	// walk file tree
	deleteFiles, err := getFilesOlderThan(MessagesDir, dur)
	if err != nil {
		l.Errorf("failed to get files older than: %v", err)
		return err
	}
	for _, file := range deleteFiles {
		// file path in format:
		// MessagesDir/pubKeyID/channel/messageID
		// split on / to get pubKeyID, channel, and messageID
		// first, remove MessagesDir from path
		file = strings.Replace(file, MessagesDir, "", 1)
		// split on / to get pubKeyID, channel, and messageID
		parts := strings.Split(file, "/")
		if len(parts) != 3 {
			l.Errorf("invalid file path: %v", file)
			continue
		}
		deletions = append(deletions, Deletion{
			PubKeyID: parts[0],
			Channel:  parts[1],
			ID:       parts[2],
		})
	}
	// delete files
	for _, deletion := range deletions {
		err := DeleteMessageByID(deletion.PubKeyID, deletion.Channel, deletion.ID)
		if err != nil {
			l.Errorf("failed to delete message: %v", err)
			return err
		}
	}
	return nil
}

func TimeoutCleaner() {
	l := log.WithFields(log.Fields{
		"pkg": "persist",
		"fn":  "TimeoutCleaner",
	})
	l.Debug("timeout cleaner started")
	for {
		time.Sleep(time.Hour * 24)
		l.Debug("cleaning")
		if err := cleanupOldFiles(time.Hour * 24 * 90); err != nil {
			l.Errorf("failed to clean: %v", err)
		}
	}
}
