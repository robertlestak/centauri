package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/robertlestak/centauri/internal/agent"
	"github.com/robertlestak/centauri/internal/keys"
	log "github.com/sirupsen/logrus"
)

var (
	flagAgentChannel             *string
	flagClientMode               *bool
	flagClientOutput             *string
	flagClientOutputFormat       *string
	flagClientPrivateKeyPath     *string
	flagClientRecipientPublicKey *string
	flagClientMessageType        *string
	flagClientMessageFileName    *string
	flagClientMessageInput       *string
	flagClientMessageID          *string
	flagServerAuthToken          *string
	flagUpstreamServerAddrs      *string
	flagDataDir                  *string
)

func init() {
	ll, err := log.ParseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		ll = log.InfoLevel
	}
	log.SetLevel(ll)
}

func clnt() {
	l := log.WithFields(log.Fields{
		"pkg": "main",
		"fn":  "clnt",
	})
	l.Debug("starting")
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
	if flagClientPrivateKeyPath != nil && *flagClientPrivateKeyPath != "" {
		if err := agent.LoadPrivateKeyFromFile(*flagClientPrivateKeyPath); err != nil {
			l.Errorf("failed to load private key: %v", err)
			os.Exit(1)
		}
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
}

func main() {
	l := log.WithFields(log.Fields{
		"pkg": "main",
		"fn":  "main",
	})
	l.Debug("starting")

	flagClient := flag.NewFlagSet("cent", flag.ExitOnError)
	flagAgentChannel = flagClient.String("channel", "default", "channel to listen on")
	flagClientPrivateKeyPath = flagClient.String("key", "", "path to private key for client")
	flagClientMessageID = flagClient.String("id", "", "message id to retrieve")
	flagClientMessageFileName = flagClient.String("file", "", "filename to set for outbound file message")
	flagClientRecipientPublicKey = flagClient.String("to-key", "", "public key of recipient")
	flagClientMessageType = flagClient.String("type", "bytes", "message type to set for outbound message (bytes, file)")
	flagClientMessageInput = flagClient.String("in", "-", "input to set for outbound message")
	flagClientOutput = flagClient.String("out", "-", "path to output file.")
	flagClientOutputFormat = flagClient.String("format", "text", "output format (json, text)")
	flagServerAuthToken = flagClient.String("server-token", "", "auth token for server")
	flagUpstreamServerAddrs = flagClient.String("server-addrs", "", "addresses to join as an agent")
	flagDataDir = flagClient.String("data", "", "data directory")
	if len(os.Args) < 2 {
		fmt.Println(agent.ClientHelp())
		flagClient.PrintDefaults()
		os.Exit(1)
	}
	if err := flagClient.Parse(os.Args[2:]); err != nil {
		l.Errorf("failed to parse flags: %v", err)
		os.Exit(1)
	}
	clnt()
}
