package server

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/robertlestak/centauri/internal/keys"
	"github.com/robertlestak/centauri/pkg/message"
	"github.com/robertlestak/centauri/pkg/sign"
	"github.com/rs/cors"
	log "github.com/sirupsen/logrus"
)

func HandleCreateMessage(w http.ResponseWriter, r *http.Request) {
	l := log.WithFields(log.Fields{
		"pkg": "server",
		"fn":  "HandleCreateMessage",
	})
	l.Debug("creating message")
	mr := message.Message{}
	if err := json.NewDecoder(r.Body).Decode(&mr); err != nil {
		l.Errorf("error decoding message: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	m, err := mr.Create()
	if err != nil {
		l.Errorf("error creating message: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := json.NewEncoder(w).Encode(m); err != nil {
		l.Errorf("error encoding message: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func HandleListMesageMetaForPublicKey(w http.ResponseWriter, r *http.Request) {
	l := log.WithFields(log.Fields{
		"pkg": "server",
		"fn":  "HandleListMessageMetaForPublicKey",
	})
	l.Debug("listing message meta for public key")
	channel := message.CleanString(r.URL.Query().Get("channel"))
	pubKeyID, err := ValidateSignedRequest(r)
	if err != nil {
		l.Errorf("error validating signed request: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	l.Debugf("listing message meta for public key: %v", pubKeyID)
	messages, err := message.ListMessageMetaForPubKeyID(pubKeyID, channel)
	if err != nil {
		l.Errorf("error listing message meta for public key: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := json.NewEncoder(w).Encode(messages); err != nil {
		l.Errorf("error encoding message meta for public key: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func HandleGetMessageByID(w http.ResponseWriter, r *http.Request) {
	l := log.WithFields(log.Fields{
		"pkg": "server",
		"fn":  "HandleGetMessageByID",
	})
	l.Debug("getting message by id")
	vars := mux.Vars(r)
	id := vars["id"]
	channel := message.CleanString(vars["channel"])
	keyID := vars["keyID"]
	pubKeyID, err := ValidateSignedRequest(r)
	if err != nil {
		l.Errorf("error validating signed request: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if keyID != pubKeyID {
		l.Errorf("key id mismatch: %v != %v", keyID, pubKeyID)
		http.Error(w, "key id mismatch", http.StatusBadRequest)
		return
	}
	m, err := message.GetMessageByID(keyID, channel, id)
	if err != nil {
		l.Errorf("error getting message by id: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Write(m.Data)
}

func HandleDeleteMessageByID(w http.ResponseWriter, r *http.Request) {
	l := log.WithFields(log.Fields{
		"pkg": "server",
		"fn":  "HandleDeleteMessageByID",
	})
	l.Debug("deleting message by id")
	vars := mux.Vars(r)
	id := vars["id"]
	keyID := vars["keyID"]
	channel := message.CleanString(vars["channel"])
	pubKeyID, err := ValidateSignedRequest(r)
	if err != nil {
		l.Errorf("error validating signed request: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if keyID != pubKeyID {
		l.Errorf("key id mismatch: %v != %v", keyID, pubKeyID)
		http.Error(w, "key id mismatch", http.StatusBadRequest)
		return
	}
	if err := message.DeleteMessageByID(keyID, channel, id, false); err != nil {
		l.Errorf("error deleting message by id: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}

func ValidateSignedRequest(r *http.Request) (string, error) {
	l := log.WithFields(log.Fields{
		"pkg": "server",
		"fn":  "ValidateSignedRequest",
	})
	var pubKeyID string
	l.Debug("validating signed request")
	// signed request payload will be in header X-Signature
	// header X-Signature is base64 encoded
	sd := r.Header.Get("X-Signature")
	if sd == "" {
		l.Error("no signature header")
		return pubKeyID, errors.New("no signature header")
	}
	// decode base64
	sig, err := base64.StdEncoding.DecodeString(sd)
	if err != nil {
		l.Errorf("error decoding signature: %v", err)
		return pubKeyID, err
	}
	sr := &sign.SignedRequest{}
	if err := json.Unmarshal(sig, sr); err != nil {
		l.Errorf("error unmarshaling signature: %v", err)
		return pubKeyID, err
	}
	if err := sr.Verify(); err != nil {
		l.Errorf("error verifying signature: %v", err)
		return pubKeyID, err
	}
	pubKeyID = keys.PubKeyID(sr.PublicKey)
	return pubKeyID, nil
}

func handleHealthcheck(w http.ResponseWriter, r *http.Request) {
	l := log.WithFields(log.Fields{
		"pkg": "server",
		"fn":  "handleHealthcheck",
	})
	l.Debug("healthcheck")
	w.Write([]byte("OK"))
}

func Server(port int, authToken string, corsList []string, tlsCrtPath string, tlsKeyPath string) error {
	l := log.WithFields(log.Fields{
		"pkg": "server",
		"fn":  "Server",
	})
	l.Debug("starting server")
	r := mux.NewRouter()
	if authToken != "" {
		r.Use(func(h http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("X-Token") != authToken {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
				h.ServeHTTP(w, r)
			})
		})
	}

	r.HandleFunc("/message", HandleCreateMessage).Methods("POST")
	r.HandleFunc("/messages", HandleListMesageMetaForPublicKey).Methods("LIST")
	r.HandleFunc("/message/{keyID}/{channel}/{id}", HandleGetMessageByID).Methods("GET")
	r.HandleFunc("/message/{keyID}/{channel}/{id}", HandleDeleteMessageByID).Methods("DELETE")
	r.HandleFunc("/statusz", handleHealthcheck).Methods("GET")
	sPort := fmt.Sprintf(":%d", port)
	if len(corsList) == 0 {
		corsList = []string{"*"}
	}
	c := cors.New(cors.Options{
		AllowedOrigins:   corsList,
		AllowedMethods:   []string{"GET", "POST", "DELETE", "PUT", "LIST"},
		AllowedHeaders:   []string{"X-Token", "X-Signature", "Content-Type"},
		AllowCredentials: true,
		Debug:            false,
	})
	h := c.Handler(r)
	if tlsCrtPath != "" && tlsKeyPath != "" {
		l.Debug("starting server with TLS")
		return http.ListenAndServeTLS(sPort, tlsCrtPath, tlsKeyPath, h)
	} else {
		l.Debug("starting server without TLS")
		return http.ListenAndServe(sPort, h)
	}
}
