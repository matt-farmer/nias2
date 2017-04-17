// encoding.go
package sms

import (
	"bytes"
	"encoding/gob"
	"github.com/nsip/nias2/lib"
	"log"
)

// helper routines collection for all things encoding

// binary encding for messages going to internal q/store.
func EncodeNiasMessage(msg *lib.NiasMessage) []byte {

	encBuf := new(bytes.Buffer)
	encoder := gob.NewEncoder(encBuf)
	err := encoder.Encode(msg)
	if err != nil {
		log.Printf("Encoder unable to binary encode message for: %#v\n", msg)
	}
	return encBuf.Bytes()

}

// binary decoding for messages coming from internal q/store.
func DecodeNiasMessage(bytemsg []uint8) *lib.NiasMessage {

	decBuf := bytes.NewBuffer(bytemsg)
	decoder := gob.NewDecoder(decBuf)
	var msgOut lib.NiasMessage
	err := decoder.Decode(&msgOut)
	if err != nil {
		log.Println("Error decoding message from q/store(internal):", err)
	}
	return &msgOut
}
