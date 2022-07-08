package main

import (
	"flag"
	"os"
	"strings"
	"sync"

	"github.com/robertlestak/centauri/internal/agent"
	"github.com/robertlestak/centauri/internal/keys"
	"github.com/robertlestak/centauri/internal/persist"
	"github.com/robertlestak/centauri/pkg/cfg"
	log "github.com/sirupsen/logrus"
)

var (
	wg                      sync.WaitGroup
	flagAgentChannel        *string
	flagAgentPrivateKeyPath *string
	flagDataDir             *string
	flagServerAuthToken     *string
	flagUpstreamServerAddrs *string
)

func init() {
	ll, err := log.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		ll = log.InfoLevel
	}
	log.SetLevel(ll)
}

func loadcfg() {
	cfg.Init()
	if cfg.Config.Agent.Channel == "" {
		cfg.Config.Agent.Channel = *flagAgentChannel
	}
	if cfg.Config.Agent.PrivateKeyPath == "" {
		cfg.Config.Agent.PrivateKeyPath = *flagAgentPrivateKeyPath
	}
	if cfg.Config.Agent.ServerAuthToken == "" {
		cfg.Config.Agent.ServerAuthToken = *flagServerAuthToken
	}
	if len(cfg.Config.Agent.ServerAddrs) == 0 {
		ss := strings.Split(*flagUpstreamServerAddrs, ",")
		var addrs []string
		for _, addr := range ss {
			if strings.TrimSpace(addr) == "" {
				continue
			}
			addrs = append(addrs, addr)
		}
		cfg.Config.Agent.ServerAddrs = addrs
	}
	if cfg.Config.Agent.DataDir == "" {
		cfg.Config.Agent.DataDir = *flagDataDir
	}
}

func agnt() {
	l := log.WithFields(log.Fields{
		"pkg": "main",
		"fn":  "agnt",
	})
	l.Debug("starting")
	loadcfg()
	if err := persist.InitAgent(cfg.Config.Agent.DataDir); err != nil {
		l.Errorf("failed to init persist: %v", err)
		os.Exit(1)
	}
	agent.ServerAddrs = cfg.Config.Agent.ServerAddrs
	if err := agent.LoadPrivateKeyFromFile(cfg.Config.Agent.PrivateKeyPath); err != nil {
		l.Errorf("failed to load private key: %v", err)
		os.Exit(1)
	}
	go keys.PubKeyLoader(cfg.Config.Agent.DataDir + "/pubkeys")
	agent.DefaultChannel = cfg.Config.Agent.Channel
	if cfg.Config.Agent.ServerAuthToken != "" {
		agent.ServerAuthToken = cfg.Config.Agent.ServerAuthToken
	}
	if err := agent.Agent(); err != nil {
		l.Errorf("failed to start agent: %v", err)
		os.Exit(1)
	}
}

func main() {
	l := log.WithFields(log.Fields{
		"pkg": "main",
		"fn":  "main",
	})
	l.Debug("starting")
	flagAgent := flag.NewFlagSet("centauri-agent", flag.ExitOnError)
	flagAgentChannel = flagAgent.String("channel", "default", "channel to listen on")
	flagAgentPrivateKeyPath = flagAgent.String("key", "", "path to private key for agent")
	flagServerAuthToken = flagAgent.String("server-token", "", "auth token for server")
	flagUpstreamServerAddrs = flagAgent.String("server-addrs", "", "addresses to join as an agent")
	flagDataDir = flagAgent.String("data", "", "data directory")
	if err := flagAgent.Parse(os.Args[1:]); err != nil {
		l.Errorf("failed to parse flags: %v", err)
		os.Exit(1)
	}
	wg.Add(1)
	go agnt()
	wg.Wait()
}
