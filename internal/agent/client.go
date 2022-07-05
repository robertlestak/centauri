package agent

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/robertlestak/centauri/internal/keys"
	log "github.com/sirupsen/logrus"
)

func sendOutput(data []byte, out string) error {
	l := log.WithFields(log.Fields{
		"pkg": "agent",
		"fn":  "sendOutput",
	})
	l.Info("sending output")
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
	tbl += "ID\tSize\n"
	for _, msg := range msgs {
		tbl += msg.ID + "\t" + strconv.Itoa(int(msg.Size)) + "\n"
	}
	return tbl
}

func listMessages(channel string, format string, out string) error {
	l := log.WithFields(log.Fields{
		"pkg": "agent",
		"fn":  "listMessages",
	})
	l.Info("listing messages")
	msgs, err := CheckPendingMessages(channel)
	if err != nil {
		l.Errorf("error checking pending messages: %v", err)
		return err
	}
	if len(msgs) == 0 {
		l.Info("no pending messages")
		return nil
	}
	l.Infof("pending messages: %v", msgs)
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
	l.Info("getting message")
	msg, fn, err := getMessageData(channel, id)
	if err != nil {
		l.Errorf("error getting message: %v", err)
		return err
	}
	l.Infof("message: %v", msg)
	// if out is "-" or empty, then write to stdout
	if out == "-" || out == "" {
		_, err := os.Stdout.Write(msg.Data)
		if err != nil {
			l.Errorf("failed to write to stdout: %v", err)
			return err
		}
		return nil
	}
	// if out is a directory and fn is not empty, then write to file with fn
	if stat, err := os.Stat(out); err == nil && stat.IsDir() {
		if fn != "" {
			if err := ioutil.WriteFile(out+"/"+fn, msg.Data, 0644); err != nil {
				l.Errorf("failed to write to file: %v", err)
				return err
			}
		} else {
			l.Errorf("no filename provided")
			return errors.New("no filename provided")
		}
		return nil
	} else {
		// otherwise, write output to file provided
		if err := ioutil.WriteFile(out, msg.Data, 0644); err != nil {
			l.Errorf("failed to write to file: %v", err)
			return err
		}
	}
	return nil
}

func sendMessageFromInput() error {
	l := log.WithFields(log.Fields{
		"pkg": "agent",
		"fn":  "sendMessageFromInput",
	})
	l.Info("sending message from input")
	var recipID string
	for id := range keys.PublicKeyChain {
		recipID = id
		break
	}
	var in io.ReadCloser
	if ClientMessageInput == "-" || ClientMessageInput == "" {
		l.Info("reading from stdin")
		in = os.Stdin
	} else {
		l.Infof("reading from file: %v", ClientMessageInput)
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
