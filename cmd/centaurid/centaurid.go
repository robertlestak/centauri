package main

import (
	"flag"
	network "net"
	"os"
	"strings"
	"sync"

	"github.com/robertlestak/centauri/internal/events"
	"github.com/robertlestak/centauri/internal/net"
	"github.com/robertlestak/centauri/internal/persist"
	"github.com/robertlestak/centauri/internal/server"
	"github.com/robertlestak/centauri/pkg/message"
	log "github.com/sirupsen/logrus"
)

var (
	wg                     sync.WaitGroup
	flagPeerConnectionMode *string
	flagPeerBindPort       *int
	flagPeerAdvertisePort  *int
	flagPeerAdvertiseAddr  *string
	flagPeerAllowedCidrs   *string
	flagServerPort         *string
	flagServerTLSCertPath  *string
	flagServerTLSKeyPath   *string
	flagPeerAddrs          *string
	flagPeerName           *string
	flagDataDir            *string
	flagServerAuthToken    *string
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
	l.Debug("starting")
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
	l.Debug("starting")
	if err := server.Server(*flagServerPort, *flagServerAuthToken, *flagServerTLSCertPath, *flagServerTLSKeyPath); err != nil {
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
	flagPeerBindPort = flagPeer.Int("bind-port", 5665, "peer port to bind")
	flagPeerAdvertisePort = flagPeer.Int("advertise-port", 5665, "peer port to advertise")
	flagPeerAdvertiseAddr = flagPeer.String("advertise-addr", "", "peer address to advertise")
	flagPeerAddrs = flagPeer.String("addrs", "", "addresses to join")
	flagPeerAllowedCidrs = flagPeer.String("cidrs", "", "cidrs to allow. comma separated. empty for all")
	flagPeerConnectionMode = flagPeer.String("mode", "lan", "peer connection mode (lan, wan, local)")
	flagServerAuthToken = flagPeer.String("server-token", "", "auth token for server")
	flagServerPort = flagPeer.String("server-port", "5666", "port to use for server")
	flagServerTLSCertPath = flagPeer.String("server-cert", "", "path to server TLS cert")
	flagServerTLSKeyPath = flagPeer.String("server-key", "", "path to server TLS key")
	flagPeerName = flagPeer.String("name", "", "name of this node")
	flagDataDir = flagPeer.String("data", "", "data directory")
	if err := flagPeer.Parse(os.Args[1:]); err != nil {
		l.Errorf("failed to parse flags: %v", err)
		os.Exit(1)
	}
	wg.Add(1)
	go peer()
	go serv()
	wg.Wait()
}
