package cfg

import (
	"io/ioutil"
	"os"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

var (
	// Config is the global configuration.
	Config Cfg
)

type ClientConfig struct {
	Channel         string   `yaml:"channel"`
	Output          string   `yaml:"output"`
	Format          string   `yaml:"format"`
	PrivateKeyPath  string   `yaml:"privateKeyPath"`
	ServerAuthToken string   `yaml:"serverAuthToken"`
	ServerAddrs     []string `yaml:"serverAddrs"`
}

type PeerConfig struct {
	Name                string   `yaml:"name"`
	ConnectionMode      string   `yaml:"connectionMode"`
	GossipBindPort      int      `yaml:"gossipBindPort"`
	GossipAdvertisePort int      `yaml:"gossipAdvertisePort"`
	DataBindPort        int      `yaml:"dataBindPort"`
	DataAdvertisePort   int      `yaml:"dataAdvertisePort"`
	AdvertiseAddr       string   `yaml:"advertiseAddr"`
	AllowedCidrs        []string `yaml:"allowedCidrs"`
	ServerPort          int      `yaml:"serverPort"`
	ServerTLSCertPath   string   `yaml:"serverTLSCertPath"`
	ServerTLSKeyPath    string   `yaml:"serverTLSKeyPath"`
	PeerAddrs           []string `yaml:"peerAddrs"`
	DataDir             string   `yaml:"dataDir"`
	ServerAuthToken     string   `yaml:"serverAuthToken"`
}

type AgentConfig struct {
	Channel         string   `yaml:"channel"`
	PrivateKeyPath  string   `yaml:"privateKeyPath"`
	DataDir         string   `yaml:"dataDir"`
	ServerAuthToken string   `yaml:"serverAuthToken"`
	ServerAddrs     []string `yaml:"serverAddrs"`
}

type Cfg struct {
	Client ClientConfig `yaml:"client"`
	Peer   PeerConfig   `yaml:"peer"`
	Agent  AgentConfig  `yaml:"agent"`
}

func (c *Cfg) Load(path string) error {
	l := log.WithFields(log.Fields{
		"pkg": "cfg",
		"fn":  "Load",
	})
	l.Debug("starting")
	data, err := ioutil.ReadFile(path)
	if err != nil {
		l.Errorf("failed to read config file: %v", err)
		return err
	}
	l.Debugf("read config file: %s, data: %s", path, string(data))
	if err := yaml.Unmarshal(data, c); err != nil {
		l.Errorf("failed to unmarshal config file: %v", err)
		return err
	}
	l.Debugf("unmarshaled config: %+v", c)
	l.Debug("finished")
	return nil
}

func LoadIfExists(path string) error {
	l := log.WithFields(log.Fields{
		"pkg": "cfg",
		"fn":  "LoadIfExists",
	})
	l.Debug("starting")
	if _, err := os.Stat(path); err != nil {
		l.Debug("config file does not exist")
		return nil
	}
	if err := Config.Load(path); err != nil {
		l.Errorf("failed to load config file: %v", err)
		return err
	}
	l.Debug("finished")
	return nil
}

func Init() {
	var cfgPath string
	if os.Getenv("CENTAURI_CONFIG") != "" {
		cfgPath = os.Getenv("CENTAURI_CONFIG")
	} else {
		// get user's home directory
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatal(err)
		}
		// append config file to home directory
		cfgPath = home + "/.centauri/config.yaml"
	}
	if err := LoadIfExists(cfgPath); err != nil {
		log.Fatal(err)
	}
}
