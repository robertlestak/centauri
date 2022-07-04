package main

import (
	"flag"
	network "net"
	"os"
	"strings"
	"sync"

	"github.com/robertlestak/centauri/internal/agent"
	"github.com/robertlestak/centauri/internal/events"
	"github.com/robertlestak/centauri/internal/keys"
	"github.com/robertlestak/centauri/internal/net"
	"github.com/robertlestak/centauri/internal/persist"
	"github.com/robertlestak/centauri/internal/server"
	"github.com/robertlestak/centauri/pkg/message"
	log "github.com/sirupsen/logrus"
)

var (
	wg                      sync.WaitGroup
	flagAgentMode           *bool
	flagAgentChannel        *string
	flagAgentPrivateKeyPath *string
	flagPeerConnectionMode  *string
	flagPeerBindPort        *int
	flagPeerAdvertisePort   *int
	flagPeerAdvertiseAddr   *string
	flagPeerAllowedCidrs    *string
	flagServerPort          *string
	flagPeerAddrs           *string
	flagPeerName            *string
	flagDataDir             *string
	flagPeerMode            *bool
	flagServerMode          *bool
	flagServerAuthToken     *string
	flagUpstreamServerAddrs *string
	flagHelp                *bool
)

func init() {
	ll, err := log.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		ll = log.InfoLevel
	}
	log.SetLevel(ll)
}

func peer() {
	l := log.WithFields(log.Fields{
		"pkg": "main",
		"fn":  "peer",
	})
	l.Info("starting")
	if err := persist.Init(*flagDataDir, *flagPeerName); err != nil {
		l.Errorf("failed to init persist: %v", err)
		os.Exit(1)
	}
	addrspl := strings.Split(*flagPeerAddrs, ",")
	var addrs []string
	for _, addr := range addrspl {
		if strings.TrimSpace(addr) == "" {
			continue
		}
		addrs = append(addrs, addr)
	}
	cidrSpl := strings.Split(*flagPeerAllowedCidrs, ",")
	var cidrs []network.IPNet
	for _, cidr := range cidrSpl {
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
		*flagPeerName,
		*flagPeerAdvertiseAddr,
		*flagPeerAdvertisePort,
		*flagPeerBindPort,
		*flagPeerConnectionMode,
		cidrs,
	)
	if err != nil {
		l.Errorf("failed to create peer: %v", err)
		os.Exit(1)
	}
	if len(addrs) > 0 {
		err = net.Join(addrs)
		if err != nil {
			l.Errorf("failed: %v", err)
			os.Exit(1)
		}
	}
	net.PeerName = *flagPeerName
	net.CreateQueue()
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
	l.Info("starting")
	if err := server.Server(*flagServerPort, *flagServerAuthToken); err != nil {
		l.Errorf("failed to start server: %v", err)
		os.Exit(1)
	}
}

func agnt() {
	l := log.WithFields(log.Fields{
		"pkg": "main",
		"fn":  "agnt",
	})
	l.Info("starting")
	if err := persist.InitAgent(*flagDataDir); err != nil {
		l.Errorf("failed to init persist: %v", err)
		os.Exit(1)
	}
	ss := strings.Split(*flagUpstreamServerAddrs, ",")
	var addrs []string
	for _, addr := range ss {
		if strings.TrimSpace(addr) == "" {
			continue
		}
		addrs = append(addrs, addr)
	}
	agent.ServerAddrs = addrs
	if err := agent.LoadPrivateKeyFromFile(*flagAgentPrivateKeyPath); err != nil {
		l.Errorf("failed to load private key: %v", err)
		os.Exit(1)
	}
	go keys.PubKeyLoader(*flagDataDir + "/pubkeys")
	agent.DefaultChannel = *flagAgentChannel
	if *flagServerAuthToken != "" {
		agent.ServerAuthToken = *flagServerAuthToken
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
	l.Info("starting")
	flagAgentMode = flag.Bool("agent", false, "run as agent")
	flagAgentChannel = flag.String("agent-channel", "default", "channel to listen on")
	flagAgentPrivateKeyPath = flag.String("agent-key", "", "path to private key for agent")
	flagPeerMode = flag.Bool("peer", false, "run as peer")
	flagPeerBindPort = flag.Int("peer-bind-port", 0, "peer port to bind")
	flagPeerAdvertisePort = flag.Int("peer-advertise-port", 0, "peer port to advertise")
	flagPeerAdvertiseAddr = flag.String("peer-advertise-addr", "", "peer address to advertise")
	flagPeerAddrs = flag.String("peer-addrs", "", "addresses to join")
	flagPeerAllowedCidrs = flag.String("peer-cidrs", "", "cidrs to allow. comma separated. empty for all")
	flagPeerConnectionMode = flag.String("peer-mode", "lan", "peer connection mode (lan, wan, local)")
	flagServerMode = flag.Bool("server", false, "run as server")
	flagUpstreamServerAddrs = flag.String("server-addrs", "", "addresses to join as an agent")
	flagServerPort = flag.String("server-port", "8080", "port to use for server")
	flagServerAuthToken = flag.String("server-token", "", "auth token for server")
	flagPeerName = flag.String("peer-name", "", "name of this node")
	flagDataDir = flag.String("data", "", "data directory")
	flagHelp = flag.Bool("help", false, "show help")
	flag.Parse()
	if *flagHelp {
		flag.PrintDefaults()
		os.Exit(0)
	}
	if *flagAgentMode {
		wg.Add(1)
		go agnt()
	}
	if *flagPeerMode {
		wg.Add(1)
		go peer()
	}
	if *flagServerMode {
		wg.Add(1)
		go serv()
	}
	wg.Wait()
}
