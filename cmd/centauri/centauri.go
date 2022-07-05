package main

import (
	"flag"
	"fmt"
	"io/ioutil"
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
	wg                           sync.WaitGroup
	flagAgentMode                *bool
	flagAgentChannel             *string
	flagAgentPrivateKeyPath      *string
	flagClientMode               *bool
	flagClientOutput             *string
	flagClientOutputFormat       *string
	flagClientPrivateKeyPath     *string
	flagClientRecipientPublicKey *string
	flagClientMessageType        *string
	flagClientMessageFileName    *string
	flagClientMessageInput       *string
	flagClientMessageID          *string
	flagPeerConnectionMode       *string
	flagPeerBindPort             *int
	flagPeerAdvertisePort        *int
	flagPeerAdvertiseAddr        *string
	flagPeerAllowedCidrs         *string
	flagServerPort               *string
	flagPeerAddrs                *string
	flagPeerName                 *string
	flagDataDir                  *string
	flagPeerMode                 *bool
	flagServerMode               *bool
	flagServerAuthToken          *string
	flagUpstreamServerAddrs      *string
	flagHelp                     *bool
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
	if flagUpstreamServerAddrs == nil {
		l.Error("no upstream server addrs specified")
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

func clnt() {
	l := log.WithFields(log.Fields{
		"pkg": "main",
		"fn":  "clnt",
	})
	l.Info("starting")
	if flagUpstreamServerAddrs == nil {
		l.Error("no upstream server addrs specified")
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
	if err := agent.LoadPrivateKeyFromFile(*flagClientPrivateKeyPath); err != nil {
		l.Errorf("failed to load private key: %v", err)
		os.Exit(1)
	}
	if flagClientRecipientPublicKey != nil && *flagClientRecipientPublicKey != "" {
		if *flagClientRecipientPublicKey == "-" {
			var err error
			agent.ClientRecipientPublicKey, err = ioutil.ReadAll(os.Stdin)
			if err != nil {
				l.Errorf("failed to read public key: %v", err)
				os.Exit(1)
			}
		} else {
			// read from file
			var err error
			agent.ClientRecipientPublicKey, err = ioutil.ReadFile(*flagClientRecipientPublicKey)
			if err != nil {
				l.Errorf("failed to read public key: %v", err)
				os.Exit(1)
			}
		}
		keys.AddKeyToPublicChain(agent.ClientRecipientPublicKey)
	}
	agent.DefaultChannel = *flagAgentChannel
	agent.Output = *flagClientOutput
	agent.OutputFormat = *flagClientOutputFormat
	agent.ClientMessageID = *flagClientMessageID
	agent.ClientMessageType = *flagClientMessageType
	agent.ClientMessageFileName = *flagClientMessageFileName
	agent.ClientMessageInput = *flagClientMessageInput
	if *flagServerAuthToken != "" {
		agent.ServerAuthToken = *flagServerAuthToken
	}
	if err := agent.Client(); err != nil {
		l.Errorf("failed to start client: %v", err)
		os.Exit(1)
	}
	wg.Done()
}

func printHelp() {
	fmt.Println("Usage:")
	fmt.Println("  agent [flags]")
	fmt.Println("  peer [flags]")
	fmt.Println("  client [flags]")
	fmt.Println("")
	fmt.Println("Flags:")
	flag.PrintDefaults()
}

func main() {
	l := log.WithFields(log.Fields{
		"pkg": "main",
		"fn":  "main",
	})
	l.Info("starting")
	var action string
	// first arg is the action
	if len(os.Args) > 1 {
		l.Infof("action: %s", os.Args[1])
		action = os.Args[1]
	} else {
		printHelp()
		os.Exit(1)
	}

	switch action {
	case "agent":
		flagAgent := flag.NewFlagSet("agent", flag.ExitOnError)
		flagAgentChannel = flagAgent.String("channel", "default", "channel to listen on")
		flagAgentPrivateKeyPath = flagAgent.String("key", "", "path to private key for agent")
		flagServerAuthToken = flagAgent.String("server-token", "", "auth token for server")
		flagUpstreamServerAddrs = flagAgent.String("server-addrs", "", "addresses to join as an agent")
		flagDataDir = flagAgent.String("data", "", "data directory")
		if err := flagAgent.Parse(os.Args[2:]); err != nil {
			l.Errorf("failed to parse flags: %v", err)
			os.Exit(1)
		}
		wg.Add(1)
		go agnt()
	case "client":
		flagClient := flag.NewFlagSet("client", flag.ExitOnError)
		flagAgentChannel = flagClient.String("channel", "default", "channel to listen on")
		flagClientPrivateKeyPath = flagClient.String("key", "", "path to private key for client")
		flagClientMessageID = flagClient.String("id", "", "message id to retrieve")
		flagClientMessageFileName = flagClient.String("file", "", "filename to set for outbound file message")
		flagClientRecipientPublicKey = flagClient.String("to-key", "", "public key of recipient")
		flagClientMessageType = flagClient.String("type", "message", "message type to set for outbound message (message, file)")
		flagClientMessageInput = flagClient.String("in", "-", "input to set for outbound message")
		flagClientOutput = flagClient.String("out", "-", "path to output file.")
		flagClientOutputFormat = flagClient.String("format", "json", "output format (json, text)")
		flagServerAuthToken = flagClient.String("server-token", "", "auth token for server")
		flagUpstreamServerAddrs = flagClient.String("server-addrs", "", "addresses to join as an agent")
		flagDataDir = flagClient.String("data", "", "data directory")
		if err := flagClient.Parse(os.Args[3:]); err != nil {
			l.Errorf("failed to parse flags: %v", err)
			os.Exit(1)
		}
		wg.Add(1)
		go clnt()
	case "peer":
		flagPeer := flag.NewFlagSet("peer", flag.ExitOnError)
		flagPeerBindPort = flagPeer.Int("bind-port", 0, "peer port to bind")
		flagPeerAdvertisePort = flagPeer.Int("advertise-port", 0, "peer port to advertise")
		flagPeerAdvertiseAddr = flagPeer.String("advertise-addr", "", "peer address to advertise")
		flagPeerAddrs = flagPeer.String("addrs", "", "addresses to join")
		flagPeerAllowedCidrs = flagPeer.String("cidrs", "", "cidrs to allow. comma separated. empty for all")
		flagPeerConnectionMode = flagPeer.String("mode", "lan", "peer connection mode (lan, wan, local)")
		flagServerAuthToken = flagPeer.String("server-token", "", "auth token for server")
		flagServerPort = flagPeer.String("server-port", "8080", "port to use for server")
		flagPeerName = flagPeer.String("name", "", "name of this node")
		flagDataDir = flagPeer.String("data", "", "data directory")
		if err := flagPeer.Parse(os.Args[2:]); err != nil {
			l.Errorf("failed to parse flags: %v", err)
			os.Exit(1)
		}
		wg.Add(1)
		go peer()
		go serv()
	default:
		l.Error("unknown action")
		os.Exit(1)
	}
	wg.Wait()
}
