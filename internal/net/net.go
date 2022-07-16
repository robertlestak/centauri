package net

import (
	"encoding/json"
	"errors"
	"math/rand"
	"net"
	network "net"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/memberlist"
	log "github.com/sirupsen/logrus"
)

var (
	PeerName                  string
	PeerAddr                  string
	PeerKey                   []byte
	PeerDataPort              int
	Meta                      []byte
	NotifyMessageEventHandler func(data []byte) error
	mtx                       sync.RWMutex
	List                      *memberlist.Memberlist
	Queue                     *memberlist.TransmitLimitedQueue
	recentMessages            = map[string][]string{}
)

type BroadcastMessage struct {
	Type     string `json:"type"`
	Channel  string `json:"channel"`
	PubKeyID string `json:"pubKeyID"`
	ID       string `json:"id"`
	PeerAddr string `json:"peerAddr"`
	PeerPort int    `json:"peerPort"`
}

type broadcast struct {
	msgType  string
	pubKeyID string
	channel  string
	msgID    string
	msg      []byte
	notify   chan<- struct{}
}

type delegate struct{}

type NodeMeta struct {
	PeerAddr string `json:"peerAddr"`
	PeerPort int    `json:"peerPort"`
}

func GetLocalIP() string {
	addrs, err := network.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

func createMeta() []byte {
	meta := &NodeMeta{}
	meta.PeerAddr = PeerAddr
	if meta.PeerAddr == "" {
		// get local address
		addr := GetLocalIP()
		if addr != "" {
			meta.PeerAddr = addr
		}
	}
	meta.PeerPort = PeerDataPort
	var err error
	Meta, err = json.Marshal(&meta)
	if err != nil {
		log.Errorf("failed to marshal meta: %v", err)
	}
	return Meta
}

func (d *delegate) NodeMeta(limit int) []byte {
	return Meta
}

func GetRandomPeer() (*memberlist.Node, error) {
	llen := len(List.Members())
	if llen == 0 {
		return nil, errors.New("no peers")
	}
	idx := rand.Intn(llen)
	return List.Members()[idx], nil
}

func handleReceiveMessageData(b []byte) {
	l := log.WithFields(log.Fields{
		"module": "net",
		"method": "handleReceiveMessageData",
	})
	l.Debugf("received message: %s", string(b))
	var msg BroadcastMessage
	if err := json.Unmarshal(b, &msg); err != nil {
		l.Errorf("error unmarshalling message: %s", err)
		return
	}
	if checkMessageHandled(msg.Type, msg.PubKeyID, msg.Channel, msg.ID) {
		return
	}
	l.Debugf("message not handled: %s", string(b))
	if NotifyMessageEventHandler != nil {
		if err := NotifyMessageEventHandler(b); err != nil {
			l.Errorf("error handling message: %s", err)
		}
	}
	storeNewMessage(msg.Type, msg.PubKeyID, msg.Channel, msg.ID)
	l.Debugf("message handled: %s", string(b))
}

func (d *delegate) NotifyMsg(b []byte) {
	l := log.WithFields(log.Fields{
		"module": "net",
		"method": "NotifyMsg",
	})
	cp := make([]byte, len(b))
	copy(cp, b)
	l.Debugf("received message: %s", string(cp))
	if len(b) == 0 {
		return
	}
	go handleReceiveMessageData(cp)
}

func (d *delegate) GetBroadcasts(overhead, limit int) [][]byte {
	return Queue.GetBroadcasts(overhead, limit)
}

func (d *delegate) LocalState(join bool) []byte {
	mtx.RLock()
	m := recentMessages
	mtx.RUnlock()
	b, _ := json.Marshal(m)
	return b
}

func (d *delegate) MergeRemoteState(buf []byte, join bool) {
	if len(buf) == 0 {
		return
	}
	if !join {
		return
	}
	var m map[string][]string
	if err := json.Unmarshal(buf, &m); err != nil {
		return
	}
	mtx.Lock()
	for k, v := range m {
		recentMessages[k] = v
	}
	mtx.Unlock()
}

type eventDelegate struct{}

func (ed *eventDelegate) NotifyJoin(node *memberlist.Node) {
	l := log.WithFields(log.Fields{
		"pkg": "net",
		"fn":  "NotifyJoin",
	})
	l.Debugf("A node has joined: " + node.String())
}

func (ed *eventDelegate) NotifyLeave(node *memberlist.Node) {
	l := log.WithFields(log.Fields{
		"pkg": "net",
		"fn":  "NotifyLeave",
	})
	l.Debugf("A node has left: " + node.String())
}

func (ed *eventDelegate) NotifyUpdate(node *memberlist.Node) {
	l := log.WithFields(log.Fields{
		"pkg": "net",
		"fn":  "NotifyUpdate",
	})
	l.Debugf("A node has updated: " + node.String())
}

func (b *broadcast) Invalidates(other memberlist.Broadcast) bool {
	l := log.WithFields(log.Fields{
		"pkg": "net",
		"fn":  "Invalidates",
	})
	l.Debugf("checking if broadcast invalidates: %s", other)
	ob := other.(*broadcast)
	if b.pubKeyID != ob.pubKeyID {
		return false
	}
	if b.channel != ob.channel {
		return false
	}
	if b.msgID != ob.msgID {
		return false
	}
	if b.msgType == ob.msgType {
		return false
	}
	if b.msgType == "deleteMessage" && ob.msgType == "newMessage" {
		return true
	}
	l.Debugf("broadcast invalidates: %s", other)
	return false
}

func (b *broadcast) Message() []byte {
	return b.msg
}

func (b *broadcast) Finished() {
	if b.notify != nil {
		close(b.notify)
	}
}

func ListMembers() []*memberlist.Node {
	// for _, member := range List.Members() {
	// 	log.Printf("Member: %s %s\n", member.Name, member.Addr)
	// }
	return List.Members()
}

func PeerInList(peerName string) bool {
	for _, member := range List.Members() {
		if member.Name == peerName {
			return true
		}
	}
	return false
}

func NodeAddr() string {
	return List.LocalNode().Addr.String()
}

func AdvertiseAddr() string {
	if PeerAddr != "" {
		return PeerAddr
	}
	return NodeAddr()
}

// resolveAddr will check if the given address is an IP
// if so, it will return as is. If input is not an IP,
// it will do a DNS lookup and return the first IP.
func resolveAddr(addr string) (string, error) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return "", nil
	}
	// check if it's an IP
	ip := network.ParseIP(addr)
	if ip == nil {
		// not an IP, do a DNS lookup
		ips, err := network.LookupIP(addr)
		if err != nil {
			return "", err
		}
		if len(ips) == 0 {
			return "", nil
		}
		addr = ips[0].String()
	}
	return addr, nil
}

func Create(nodeName string, addr string, advPort int, bindPort int, connMode string, cidrsAllowed []network.IPNet) error {
	l := log.WithFields(log.Fields{
		"pkg": "net",
		"fn":  "Create",
	})
	var cfg *memberlist.Config
	if connMode == "wan" {
		cfg = memberlist.DefaultWANConfig()
	} else if connMode == "local" {
		cfg = memberlist.DefaultLANConfig()
	} else {
		cfg = memberlist.DefaultLocalConfig()
	}
	if len(PeerKey) > 0 {
		cfg.SecretKey = PeerKey
	}
	cfg.CIDRsAllowed = cidrsAllowed
	cfg.BindPort = bindPort
	cfg.AdvertisePort = advPort
	raddr, err := resolveAddr(addr)
	if err != nil {
		l.Errorf("error resolving address: %s", err)
		return err
	}
	cfg.AdvertiseAddr = raddr
	cfg.Name = nodeName
	cfg.Events = &eventDelegate{}
	cfg.Delegate = &delegate{}
	createMeta()
	list, err := memberlist.Create(cfg)
	if err != nil {
		l.Errorf("failed to create memberlist: %v", err)
		return err
	}
	l.Debug("created memberlist")
	List = list
	return nil
}

func Join(addrs []string) error {
	l := log.WithFields(log.Fields{
		"pkg": "net",
		"fn":  "Join",
	})
	var err error
	_, err = List.Join(addrs)
	if err != nil {
		l.Errorf("failed to join memberlist: %v", err)
		return err
	}
	l.Debug("joined memberlist")
	return nil
}

func CreateQueue() {
	Queue = &memberlist.TransmitLimitedQueue{
		NumNodes: func() int {
			return List.NumMembers()
		},
		RetransmitMult: 3,
	}
}

func Broadcast(b *broadcast) {
	Queue.QueueBroadcast(b)
}

func BroadcastNewMessage(pubKeyID string, channel string, id string) error {
	l := log.WithFields(log.Fields{
		"pkg": "net",
		"fn":  "BroadcastNewMessage",
	})
	l.Debugf("Broadcasting new message for pubKeyID: %s, channel: %s, id: %s", pubKeyID, channel, id)
	msg := &BroadcastMessage{
		Type:     "newMessage",
		Channel:  channel,
		PubKeyID: pubKeyID,
		PeerAddr: AdvertiseAddr(),
		PeerPort: PeerDataPort,
		ID:       id,
	}
	b, err := json.Marshal(msg)
	if err != nil {
		l.Errorf("failed to marshal message: %v", err)
		return err
	}
	bm := &broadcast{
		msgType:  "newMessage",
		pubKeyID: pubKeyID,
		channel:  channel,
		msgID:    id,
		msg:      b,
		notify:   nil,
	}
	go Broadcast(bm)
	l.Debug("broadcasted message")
	return nil
}

func BroadcastDeleteMessage(pubKeyID string, channel string, id string) error {
	l := log.WithFields(log.Fields{
		"pkg": "net",
		"fn":  "BroadcastDeleteMessage",
	})
	l.Debugf("Broadcasting delete message for pubKeyID: %s, channel: %s, id: %s", pubKeyID, channel, id)
	msg := &BroadcastMessage{
		Type:     "deleteMessage",
		PubKeyID: pubKeyID,
		Channel:  channel,
		ID:       id,
	}
	b, err := json.Marshal(msg)
	if err != nil {
		l.Errorf("failed to marshal message: %v", err)
		return err
	}
	bm := &broadcast{
		msgType:  "deleteMessage",
		pubKeyID: pubKeyID,
		channel:  channel,
		msgID:    id,
		msg:      b,
		notify:   nil,
	}
	go Broadcast(bm)
	l.Debug("broadcasted message")
	return nil
}

func storeNewMessage(mtype string, pubKeyID, channel, id string) {
	l := log.WithFields(log.Fields{
		"pkg": "net",
		"fn":  "storeNewMessage",
	})
	l.Debugf("Storing new message for pubKeyID: %s, channel: %s, id: %s", pubKeyID, channel, id)
	mtx.Lock()
	recentMessages[pubKeyID] = append(recentMessages[pubKeyID], mtype+"_"+channel+"_"+id)
	mtx.Unlock()
}

func checkMessageHandled(mtype string, pubKeyID, channel, id string) bool {
	l := log.WithFields(log.Fields{
		"pkg": "net",
		"fn":  "checkMessageHandled",
	})
	l.Debugf("Checking if message handled for pubKeyID: %s, channel: %s, id: %s", pubKeyID, channel, id)
	mtx.RLock()
	_, ok := recentMessages[pubKeyID]
	mtx.RUnlock()
	if !ok {
		l.Debug("pubkey not handled")
		return false
	}
	for _, v := range recentMessages[pubKeyID] {
		if v == mtype+"_"+channel+"_"+id {
			l.Debug("message handled")
			return true
		}
	}
	l.Debug("message not handled")
	return false
}

func clearLocalCache() {
	l := log.WithFields(log.Fields{
		"pkg": "net",
		"fn":  "clearLocalCache",
	})
	l.Debug("Clearing local cache")
	mtx.Lock()
	recentMessages = map[string][]string{}
	mtx.Unlock()
}

func CacheCleaner() {
	l := log.WithFields(log.Fields{
		"pkg": "net",
		"fn":  "CacheCleaner",
	})
	l.Debug("Cache cleaner started")
	for {
		time.Sleep(time.Minute * 5)
		l.Debug("Cleaning local cache")
		clearLocalCache()
		l.Debug("Local cache cleaned")
	}
}
