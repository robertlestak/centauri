package main

import (
	"flag"
	"fmt"
	network "net"
	"os"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/robertlestak/centauri/internal/cfg"
	"github.com/robertlestak/centauri/internal/events"
	"github.com/robertlestak/centauri/internal/net"
	"github.com/robertlestak/centauri/internal/persist"
	"github.com/robertlestak/centauri/internal/server"
	"github.com/robertlestak/centauri/pkg/message"
	log "github.com/sirupsen/logrus"
)

var (
	Version                     = "unknown"
	wg                          sync.WaitGroup
	flagPeerConnectionMode      *string
	flagPeerGossipBindPort      *int
	flagPeerGossipAdvertisePort *int
	flagPeerDataBindPort        *int
	flagPeerDataAdvertisePort   *int
	flagPeerAdvertiseAddr       *string
	flagPeerAllowedCidrs        *string
	flagServerPort              *int
	flagServerCors              *string
	flagServerTLSCertPath       *string
	flagServerTLSKeyPath        *string
	flagPeerAddrs               *string
	flagPeerName                *string
	flagDataDir                 *string
	flagServerAuthToken         *string
)

func init() {
	ll, err := log.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		ll = log.InfoLevel
	}
	log.SetLevel(ll)
}

func version() {
	fmt.Printf("version: %s\n", Version)
}

func loadcfg() {
	cfg.Init()
	if *flagPeerName != "" {
		cfg.Config.Peer.Name = *flagPeerName
	}
	if cfg.Config.Peer.Name == "" {
		hostname, err := os.Hostname()
		if err != nil {
			log.Fatal(err)
		}
		cfg.Config.Peer.Name = hostname + "-" + uuid.New().String()
	}
	if *flagDataDir != "" {
		cfg.Config.Peer.DataDir = *flagDataDir
	}
	if *flagPeerConnectionMode != "" {
		cfg.Config.Peer.ConnectionMode = *flagPeerConnectionMode
	}
	if *flagPeerGossipBindPort != 0 {
		cfg.Config.Peer.GossipBindPort = *flagPeerGossipBindPort
	}
	if *flagPeerGossipAdvertisePort != 0 {
		cfg.Config.Peer.GossipAdvertisePort = *flagPeerGossipAdvertisePort
	}
	if *flagPeerAdvertiseAddr != "" {
		cfg.Config.Peer.AdvertiseAddr = *flagPeerAdvertiseAddr
	}
	if *flagPeerDataBindPort != 0 {
		cfg.Config.Peer.DataBindPort = *flagPeerDataBindPort
	}
	if *flagPeerDataAdvertisePort != 0 {
		cfg.Config.Peer.DataAdvertisePort = *flagPeerDataAdvertisePort
	}
	if *flagPeerAllowedCidrs != "" {
		cidrSpl := strings.Split(*flagPeerAllowedCidrs, ",")
		for _, cidr := range cidrSpl {
			if strings.TrimSpace(cidr) == "" {
				continue
			}
			cfg.Config.Peer.AllowedCidrs = append(cfg.Config.Peer.AllowedCidrs, cidr)
		}
	}
	if *flagServerPort != 0 {
		cfg.Config.Peer.ServerPort = *flagServerPort
	}
	if *flagServerTLSCertPath != "" {
		cfg.Config.Peer.ServerTLSCertPath = *flagServerTLSCertPath
	}
	if *flagServerTLSKeyPath != "" {
		cfg.Config.Peer.ServerTLSKeyPath = *flagServerTLSKeyPath
	}
	if *flagServerAuthToken != "" {
		cfg.Config.Peer.ServerAuthToken = *flagServerAuthToken
	}
	if *flagServerCors != "" {
		for _, cors := range strings.Split(*flagServerCors, ",") {
			if strings.TrimSpace(cors) == "" {
				continue
			}
			cfg.Config.Peer.ServerCors = append(cfg.Config.Peer.ServerCors, cors)
		}
	}
	if *flagPeerAddrs != "" {
		addrSpl := strings.Split(*flagPeerAddrs, ",")
		for _, addr := range addrSpl {
			if strings.TrimSpace(addr) == "" {
				continue
			}
			cfg.Config.Peer.PeerAddrs = append(cfg.Config.Peer.PeerAddrs, addr)
		}
	}
}

func peer() {
	l := log.WithFields(log.Fields{
		"pkg": "main",
		"fn":  "peer",
	})
	l.Debug("starting")
	if err := persist.Init(cfg.Config.Peer.DataDir, cfg.Config.Peer.Name); err != nil {
		l.Errorf("failed to init persist: %v", err)
		os.Exit(1)
	}

	var cidrs []network.IPNet
	for _, cidr := range cfg.Config.Peer.AllowedCidrs {
		if strings.TrimSpace(cidr) == "" {
			continue
		}
		_, ipnet, err := network.ParseCIDR(cidr)
		if err != nil {
			l.Errorf("failed to parse cidr: %v", err)
			os.Exit(1)
		}
		cidrs = append(cidrs, *ipnet)
	}
	if len(cidrs) == 0 {
		cidrs = nil
	}
	var err error
	err = net.Create(
		cfg.Config.Peer.Name,
		cfg.Config.Peer.AdvertiseAddr,
		cfg.Config.Peer.GossipAdvertisePort,
		cfg.Config.Peer.GossipBindPort,
		cfg.Config.Peer.ConnectionMode,
		cidrs,
	)
	if err != nil {
		l.Errorf("failed to create peer: %v", err)
		os.Exit(1)
	}
	if len(cfg.Config.Peer.PeerAddrs) > 0 {
		err = net.Join(cfg.Config.Peer.PeerAddrs)
		if err != nil {
			l.Errorf("failed: %v", err)
			os.Exit(1)
		}
	}
	if cfg.Config.Peer.DataAdvertisePort == 0 {
		cfg.Config.Peer.DataAdvertisePort = cfg.Config.Peer.DataBindPort
	}
	net.PeerName = cfg.Config.Peer.Name
	net.PeerAddr = cfg.Config.Peer.AdvertiseAddr
	net.PeerDataPort = cfg.Config.Peer.DataAdvertisePort
	net.CreateQueue()
	go net.DataServer(cfg.Config.Peer.DataBindPort)
	go net.CacheCleaner()
	go persist.TimeoutCleaner()
	events.DeletionHandlers = append(events.DeletionHandlers, net.BroadcastDeleteMessage)
	events.NewMessageHandlers = append(events.NewMessageHandlers, net.BroadcastNewMessage)
	events.ReceivedDeletionHandlers = append(events.ReceivedDeletionHandlers, message.DeleteMessageByID)
	events.ReceivedMessageHandlers = append(events.ReceivedMessageHandlers, message.GetMessageFromPeer)
	net.NotifyMessageEventHandler = events.ReceiveMessage
}

func serv() {
	l := log.WithFields(log.Fields{
		"pkg": "main",
		"fn":  "serv",
	})
	l.Debug("starting")
	if err := server.Server(
		cfg.Config.Peer.ServerPort,
		cfg.Config.Peer.ServerAuthToken,
		cfg.Config.Peer.ServerCors,
		cfg.Config.Peer.ServerTLSCertPath,
		cfg.Config.Peer.ServerTLSKeyPath,
	); err != nil {
		l.Errorf("failed to start server: %v", err)
		os.Exit(1)
	}
}

func main() {
	l := log.WithFields(log.Fields{
		"pkg": "main",
		"fn":  "main",
	})
	l.Debug("starting")
	flagPeer := flag.NewFlagSet("centaurid", flag.ExitOnError)
	flagPeerDataBindPort = flagPeer.Int("data-bind-port", 5664, "peer port to bind")
	flagPeerDataAdvertisePort = flagPeer.Int("data-advertise-port", 5664, "peer port to advertise")
	flagPeerGossipBindPort = flagPeer.Int("gossip-bind-port", 5665, "peer port to bind")
	flagPeerGossipAdvertisePort = flagPeer.Int("gossip-advertise-port", 5665, "peer port to advertise")
	flagPeerAdvertiseAddr = flagPeer.String("advertise-addr", "", "peer address to advertise")
	flagPeerAddrs = flagPeer.String("addrs", "", "addresses to join")
	flagPeerAllowedCidrs = flagPeer.String("cidrs", "", "cidrs to allow. comma separated. empty for all")
	flagPeerConnectionMode = flagPeer.String("mode", "lan", "peer connection mode (lan, wan, local)")
	flagServerAuthToken = flagPeer.String("server-token", "", "auth token for server")
	flagServerPort = flagPeer.Int("server-port", 5666, "port to use for server")
	flagServerCors = flagPeer.String("server-cors", "*", "comma separated cors for server")
	flagServerTLSCertPath = flagPeer.String("server-cert", "", "path to server TLS cert")
	flagServerTLSKeyPath = flagPeer.String("server-key", "", "path to server TLS key")
	flagPeerName = flagPeer.String("name", "", "name of this node")
	flagDataDir = flagPeer.String("data", "", "data directory")
	if len(os.Args) > 1 {
		if err := flagPeer.Parse(os.Args[1:]); err != nil {
			l.Errorf("failed to parse flags: %v", err)
			os.Exit(1)
		}
		if os.Args[1] == "version" {
			version()
			os.Exit(0)
		}
	}
	wg.Add(1)
	loadcfg()
	go peer()
	go serv()
	wg.Wait()
}
