package net

import (
	"encoding/json"
	network "net"
	"sync"
	"time"

	"github.com/hashicorp/memberlist"
	log "github.com/sirupsen/logrus"
)

var (
	PeerName                  string
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
	PeerName string `json:"peerName"`
	ID       string `json:"id"`
	Data     []byte `json:"data"`
}

type broadcast struct {
	msg    []byte
	notify chan<- struct{}
}

type delegate struct{}

func (d *delegate) NodeMeta(limit int) []byte {
	return []byte{}
}

func (d *delegate) NotifyMsg(b []byte) {
	if len(b) == 0 {
		return
	}
	var msg BroadcastMessage
	if err := json.Unmarshal(b, &msg); err != nil {
		log.Errorf("error unmarshalling message: %s", err)
		return
	}
	if checkMessageHandled(msg.Type, msg.PubKeyID, msg.Channel, msg.ID) {
		return
	}
	log.Printf("Received message: %s\n", string(b))
	if NotifyMessageEventHandler != nil {
		if err := NotifyMessageEventHandler(b); err != nil {
			log.Errorf("error handling message: %s", err)
		}
	}
	storeNewMessage(msg.Type, msg.PubKeyID, msg.Channel, msg.ID)
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
	log.Println("A node has joined: " + node.String())
}

func (ed *eventDelegate) NotifyLeave(node *memberlist.Node) {
	log.Println("A node has left: " + node.String())
}

func (ed *eventDelegate) NotifyUpdate(node *memberlist.Node) {
	log.Println("A node was updated: " + node.String())
}

func (b *broadcast) Invalidates(other memberlist.Broadcast) bool {
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
	cfg.CIDRsAllowed = cidrsAllowed
	cfg.BindPort = bindPort
	cfg.AdvertisePort = advPort
	cfg.AdvertiseAddr = addr
	cfg.Name = nodeName
	cfg.Events = &eventDelegate{}
	cfg.Delegate = &delegate{}
	list, err := memberlist.Create(cfg)
	if err != nil {
		l.Errorf("failed to create memberlist: %v", err)
		return err
	}
	l.Info("created memberlist")
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
	l.Info("joined memberlist")
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

func Broadcast(msg []byte) {
	Queue.QueueBroadcast(&broadcast{
		msg:    msg,
		notify: nil,
	})
}

func BroadcastNewMessage(pubKeyID string, channel string, id string, data []byte) error {
	msg := &BroadcastMessage{
		Type:     "newMessage",
		Channel:  channel,
		PubKeyID: pubKeyID,
		PeerName: PeerName,
		ID:       id,
		Data:     data,
	}
	b, err := json.Marshal(msg)
	if err != nil {
		log.Errorf("failed to marshal message: %v", err)
		return err
	}
	Broadcast(b)
	return nil
}

func BroadcastDeleteMessage(pubKeyID string, channel string, id string) error {
	msg := &BroadcastMessage{
		Type:     "deleteMessage",
		PubKeyID: pubKeyID,
		PeerName: PeerName,
		Channel:  channel,
		ID:       id,
	}
	b, err := json.Marshal(msg)
	if err != nil {
		log.Errorf("failed to marshal message: %v", err)
		return err
	}
	Broadcast(b)
	return nil
}

func storeNewMessage(mtype string, pubKeyID, channel, id string) {
	mtx.Lock()
	recentMessages[pubKeyID] = append(recentMessages[pubKeyID], mtype+"_"+channel+"_"+id)
	mtx.Unlock()
}

func checkMessageHandled(mtype string, pubKeyID, channel, id string) bool {
	mtx.RLock()
	_, ok := recentMessages[pubKeyID]
	mtx.RUnlock()
	if !ok {
		return false
	}
	for _, v := range recentMessages[pubKeyID] {
		if v == mtype+"_"+channel+"_"+id {
			return true
		}
	}
	return false
}

func clearLocalCache() {
	mtx.Lock()
	recentMessages = map[string][]string{}
	mtx.Unlock()
}

func CacheCleaner() {
	for {
		time.Sleep(time.Minute * 5)
		clearLocalCache()
	}
}
