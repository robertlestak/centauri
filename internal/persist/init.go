package persist

import (
	"os"

	log "github.com/sirupsen/logrus"
)

func EnsureNodeDataDir(name string) error {
	l := log.WithFields(log.Fields{
		"pkg": "persist",
		"fn":  "EnsureNodeDataDir",
	})
	l.Info("ensuring node data dir")
	NodeDataDir = RootDataDir + "/" + name
	return EnsureDir(NodeDataDir)
}

func EnsureAgentMessagesDir() error {
	l := log.WithFields(log.Fields{
		"pkg": "persist",
		"fn":  "EnsureAgentMessagesDir",
	})
	l.Info("ensuring agent messages dir")
	AgentMessagesDir = RootDataDir + "/" + "received/messages"
	return EnsureDir(AgentMessagesDir)
}

func EnsureAgentFilesDir() error {
	l := log.WithFields(log.Fields{
		"pkg": "persist",
		"fn":  "EnsureAgentFilesDir",
	})
	l.Info("ensuring agent files dir")
	AgentFilesDir = RootDataDir + "/" + "received/files"
	return EnsureDir(AgentFilesDir)
}

func EnsureAgentOutgoingDir() error {
	l := log.WithFields(log.Fields{
		"pkg": "persist",
		"fn":  "EnsureAgentOutgoingDir",
	})
	l.Info("ensuring agent messages dir")
	AgentOutgoingDir = RootDataDir + "/" + "outgoing"
	if err := EnsureDir(AgentOutgoingDir); err != nil {
		l.Errorf("failed to create agent outgoing dir: %v", err)
		return err
	}
	AgentOutgoingFilesDir = AgentOutgoingDir + "/files"
	if err := EnsureDir(AgentOutgoingFilesDir); err != nil {
		l.Errorf("failed to create agent outgoing files dir: %v", err)
		return err
	}
	AgentOutgoingMessagesDir = AgentOutgoingDir + "/messages"
	if err := EnsureDir(AgentOutgoingMessagesDir); err != nil {
		l.Errorf("failed to ensure agent outgoing messages dir: %v", err)
		return err
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
	return EnsureDir(MessagesDir)
}

func EnsurePubKeyDir(pubKeyID string) (string, error) {
	l := log.WithFields(log.Fields{
		"pkg": "persist",
		"fn":  "EnsurePubKeyDir",
	})
	l.Info("ensuring pub key dir")
	dir := PubKeyMessageDir(pubKeyID)
	return dir, EnsureDir(dir)
}

func EnsureDir(dir string) error {
	l := log.WithFields(log.Fields{
		"pkg": "persist",
		"fn":  "EnsureDir",
		"dir": dir,
	})
	l.Info("ensuring dir")
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0755); err != nil {
				l.Errorf("failed to create dir: %v", err)
				return err
			}
		}
	}
	return nil
}

func EnsureAgentPubKeyChainDir() error {
	l := log.WithFields(log.Fields{
		"pkg": "persist",
		"fn":  "EnsureAgentPubKeyChainDir",
	})
	l.Info("ensuring pub key dir")
	dir := RootDataDir + "/pubkeys"
	AgentPubKeyChainDir = dir
	return EnsureDir(AgentPubKeyChainDir)
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
	if err := EnsureDir(dir); err != nil {
		l.Errorf("failed to create pub key dir: %v", err)
		return dir, err
	}
	l.Infof("outgoing files dir: %s", dir)
	dir = AgentOutgoingMessagesDir + "/" + pubKeyID
	if err := EnsureDir(dir); err != nil {
		l.Errorf("failed to create pub key dir: %v", err)
		return dir, err
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
