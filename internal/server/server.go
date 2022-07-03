package server

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/robertlestak/mp/internal/keys"
	"github.com/robertlestak/mp/internal/sign"
	"github.com/robertlestak/mp/pkg/message"
	log "github.com/sirupsen/logrus"
)

func HandleCreateMessage(w http.ResponseWriter, r *http.Request) {
	l := log.WithFields(log.Fields{
		"pkg": "server",
		"fn":  "HandleCreateMessage",
	})
	l.Info("creating message")
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
	l.Info("listing message meta for public key")
	vars := mux.Vars(r)
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
	messages, err := message.ListMessageMetaForPubKeyID(keyID)
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
	l.Info("getting message by id")
	vars := mux.Vars(r)
	id := vars["id"]
	keyID := vars["keyID"]
	// pubKeyID, err := ValidateSignedRequest(r)
	// if err != nil {
	// 	l.Errorf("error validating signed request: %v", err)
	// 	http.Error(w, err.Error(), http.StatusBadRequest)
	// 	return
	// }
	// if keyID != pubKeyID {
	// 	l.Errorf("key id mismatch: %v != %v", keyID, pubKeyID)
	// 	http.Error(w, "key id mismatch", http.StatusBadRequest)
	// 	return
	// }
	m, err := message.GetMessageByID(keyID, id)
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
	l.Info("deleting message by id")
	vars := mux.Vars(r)
	id := vars["id"]
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
	if err := message.DeleteMessageByID(keyID, id, false); err != nil {
		l.Errorf("error deleting message by id: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}

func HandleSignDataRequest(w http.ResponseWriter, r *http.Request) {
	l := log.WithFields(log.Fields{
		"pkg": "server",
		"fn":  "HandleSignDataRequest",
	})
	l.Info("signing data")
	type SignDataRequest struct {
		Data       []byte `json:"data"`
		PrivateKey []byte `json:"private_key"`
	}
	mr := SignDataRequest{}
	if err := json.NewDecoder(r.Body).Decode(&mr); err != nil {
		l.Errorf("error decoding message: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	pk, err := keys.BytesToPrivKey(mr.PrivateKey)
	if err != nil {
		l.Errorf("error converting private key: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	sd, err := sign.Sign(mr.Data, pk)
	if err != nil {
		l.Errorf("error signing data: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := json.NewEncoder(w).Encode(sd); err != nil {
		l.Errorf("error encoding message: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func HandleValidateDataSignature(w http.ResponseWriter, r *http.Request) {
	l := log.WithFields(log.Fields{
		"pkg": "server",
		"fn":  "HandleValidateDataSignature",
	})
	l.Info("validating data signature")
	type ValidateDataSignatureRequest struct {
		Data      []byte `json:"data"`
		Signature []byte `json:"signature"`
		PublicKey []byte `json:"public_key"`
	}
	mr := ValidateDataSignatureRequest{}
	if err := json.NewDecoder(r.Body).Decode(&mr); err != nil {
		l.Errorf("error decoding message: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := sign.Verify(mr.Data, mr.Signature, mr.PublicKey); err != nil {
		l.Errorf("error validating data signature: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := json.NewEncoder(w).Encode(true); err != nil {
		l.Errorf("error encoding message: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func ValidateSignedRequest(r *http.Request) (string, error) {
	l := log.WithFields(log.Fields{
		"pkg": "server",
		"fn":  "ValidateSignedRequest",
	})
	var pubKeyID string
	l.Info("validating signed request")
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
	pubKeyID = sign.PubKeyID(sr.PublicKey)
	return pubKeyID, nil
}

func Server(port string) error {
	l := log.WithFields(log.Fields{
		"pkg": "server",
		"fn":  "Server",
	})
	l.Info("starting server")
	r := mux.NewRouter()
	r.HandleFunc("/message", HandleCreateMessage).Methods("POST")
	r.HandleFunc("/message/{keyID}/meta", HandleListMesageMetaForPublicKey).Methods("LIST")
	r.HandleFunc("/message/{keyID}/{id}", HandleGetMessageByID).Methods("GET")
	r.HandleFunc("/message/{keyID}/{id}", HandleDeleteMessageByID).Methods("DELETE")

	// just for testing, this should be removed
	r.HandleFunc("/sign", HandleSignDataRequest).Methods("POST")
	r.HandleFunc("/sign/validate", HandleValidateDataSignature).Methods("POST")

	// start server
	return http.ListenAndServe(":"+port, r)
}
