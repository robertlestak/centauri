package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/robertlestak/centauri/internal/keys"
	log "github.com/sirupsen/logrus"
)

func sendOutput(data []byte, out string) error {
	l := log.WithFields(log.Fields{
		"pkg": "agent",
		"fn":  "sendOutput",
	})
	l.Debug("sending output")
	// if out is "-" or empty, then write to stdout
	// otherwise, write to file
	if out == "-" || out == "" {
		_, err := os.Stdout.Write(data)
		if err != nil {
			l.Errorf("failed to write to stdout: %v", err)
			return err
		}
	} else {
		if err := ioutil.WriteFile(out, data, 0644); err != nil {
			l.Errorf("failed to write to file: %v", err)
			return err
		}
	}
	return nil
}

func messageListTable(msgs []MessageMeta) string {
	var tbl string
	tbl += "ID\tChannel\tSize\tCreatedAt\n"
	for _, msg := range msgs {
		strTime := msg.CreatedAt.Format(time.RFC3339)
		tbl += msg.ID + "\t" + msg.Channel + "\t" + strconv.Itoa(int(msg.Size)) + "\t" + strTime + "\n"
	}
	return tbl
}

func listMessages(channel string, format string, out string) error {
	l := log.WithFields(log.Fields{
		"pkg": "agent",
		"fn":  "listMessages",
	})
	l.Debug("listing messages")
	msgs, err := CheckPendingMessages(channel)
	if err != nil {
		l.Errorf("error checking pending messages: %v", err)
		return err
	}
	if len(msgs) == 0 {
		l.Debug("no pending messages")
		return nil
	}
	l.Debugf("pending messages: %v", msgs)
	var data []byte
	if format == "" {
		format = "json"
	}
	switch format {
	case "json":
		data, err = json.Marshal(msgs)
	case "text":
		data = []byte(messageListTable(msgs))
	default:
		l.Errorf("unknown format: %v", format)
		return errors.New("unknown format")
	}
	if err != nil {
		l.Errorf("error marshalling messages: %v", err)
		return err
	}
	return sendOutput(data, out)
}

func getMessage(channel string, id string, out string) error {
	l := log.WithFields(log.Fields{
		"pkg": "agent",
		"fn":  "getMessage",
	})
	l.Debug("getting message")
	msg, fn, err := getMessageData(channel, id)
	if err != nil {
		l.Errorf("error getting message: %v", err)
		return err
	}
	l.Debugf("message: %v", msg)
	// if out is "-" or empty, then write to stdout
	if out == "-" || out == "" {
		_, err := os.Stdout.Write(msg.Data)
		if err != nil {
			l.Errorf("failed to write to stdout: %v", err)
			return err
		}
		return nil
	}
	// if out is a directory, then write to filename in that directory if message
	// has a file name, otherwise, write to filename with message id
	if stat, err := os.Stat(out); err == nil && stat.IsDir() {
		if fn != "" {
			if err := ioutil.WriteFile(out+"/"+fn, msg.Data, 0644); err != nil {
				l.Errorf("failed to write to file: %v", err)
				return err
			}
		} else {
			if err := ioutil.WriteFile(out+"/"+msg.ID, msg.Data, 0644); err != nil {
				l.Errorf("failed to write to file: %v", err)
				return err
			}
		}
		return nil
	} else {
		// if out is not a directory, then write to out file
		if err := ioutil.WriteFile(out, msg.Data, 0644); err != nil {
			l.Errorf("failed to write to file: %v", err)
			return err
		}
	}
	return nil
}

func getNextMessage(channel string, out string) (string, error) {
	l := log.WithFields(log.Fields{
		"pkg": "agent",
		"fn":  "getNextMessage",
	})
	l.Debug("getting next message")
	// list messages, sort by created at, get first message
	msgs, err := CheckPendingMessages(channel)
	if err != nil {
		l.Errorf("error checking pending messages: %v", err)
		return "", err
	}
	if len(msgs) == 0 {
		l.Debug("no pending messages")
		return "", nil
	}
	l.Debugf("pending messages: %v", msgs)
	// sort by created at
	sort.Slice(msgs, func(i, j int) bool {
		return msgs[i].CreatedAt.Before(msgs[j].CreatedAt)
	})
	// get first message
	msg := msgs[0]
	// get message data
	// print message id to stderr
	fmt.Fprintf(os.Stderr, "id: %v\n", msg.ID)
	return msg.ID, getMessage(channel, msg.ID, out)
}

func consumeNextMessage(channel string, out string) error {
	l := log.WithFields(log.Fields{
		"pkg": "agent",
		"fn":  "consumeNextMessage",
	})
	l.Debug("consuming next message")
	// get next message
	id, err := getNextMessage(channel, out)
	if err != nil {
		l.Errorf("error getting next message: %v", err)
		return err
	}
	// delete message
	err = ConfirmMessageReceive(channel, id)
	if err != nil {
		l.Errorf("error deleting message: %v", err)
		return err
	}
	return nil
}

func sendMessageFromInput() error {
	l := log.WithFields(log.Fields{
		"pkg": "agent",
		"fn":  "sendMessageFromInput",
	})
	l.Debug("sending message from input")
	var recipID string
	for id := range keys.PublicKeyChain {
		recipID = id
		break
	}
	var in io.ReadCloser
	if ClientMessageInput == "-" || ClientMessageInput == "" {
		l.Debug("reading from stdin")
		in = os.Stdin
	} else {
		l.Debugf("reading from file: %v", ClientMessageInput)
		var err error
		in, err = os.Open(ClientMessageInput)
		if err != nil {
			l.Errorf("failed to open input file: %v", err)
			return err
		}
		defer in.Close()
	}
	if err := sendMessage(
		DefaultChannel,
		recipID,
		ClientMessageType,
		ClientMessageFileName,
		in,
	); err != nil {
		l.Errorf("error sending message: %v", err)
		return err
	}
	return nil
}
