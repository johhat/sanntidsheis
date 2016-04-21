package messages

import (
	"encoding/json"
	"errors"
	"log"
)

type wrappedMessage struct {
	Msg     Message
	MsgType string
}

func WrapMessage(message Message) wrappedMessage {
	w := wrappedMessage{Msg: message, MsgType: message.Type()}
	return w
}

func DecodeWrappedMessage(data []byte) (Message, error) {

	var err error

	log.Println(string(data))

	tempMap := make(map[string]*json.RawMessage)
	err = json.Unmarshal(data, &tempMap)

	if err != nil {
		return nil, err
	}

	msgTypeJSON, msgTypeOk := tempMap["MsgType"]
	msgJSON, msgOk := tempMap["Msg"]

	if !msgTypeOk {
		return nil, errors.New("Missing message type field")
	}

	if !msgOk {
		return nil, errors.New("Missing message contents field")
	}

	var msgType string
	err = json.Unmarshal(*msgTypeJSON, &msgType)

	if err != nil {
		return nil, err
	}

	var m Message

	switch msgType {
	case "MockMessage":
		temp := MockMessage{}
		err = json.Unmarshal(*msgJSON, &temp)
		m = temp
	case "Heartbeat":
		temp := heartbeat{}
		err = json.Unmarshal(*msgJSON, &temp)
		m = temp
	default:
		log.Println("Error in decode - type field not known")
		return nil, errors.New("Error in decode - type field not known")
	}

	if err != nil {
		log.Println(err)
	}

	return m, err
}

func (wrapped wrappedMessage) Encode() []byte {
	bytes, _ := json.Marshal(wrapped)
	return bytes
}

//The actual message interface - all message types must satisfy this interface
type Message interface {
	Type() string
}

//Mock format

type MockMessage struct {
	Number int
	Text   string
}

func (m MockMessage) Encode() []byte {
	bytes, _ := json.Marshal(m)
	return bytes
}

func (m MockMessage) Decode(data []byte) {
	json.Unmarshal(data, m)
}

func (m MockMessage) Type() string {
	return "MockMessage"
}

//Heartbeat format
func CreateHeartbeat() heartbeat {
	return heartbeat{Code: "MartinOgJohanSinHeis \n EnLinjeTil \n EndaEnLinje"}
}

type heartbeat struct {
	Code string
}

func (m heartbeat) Type() string {
	return "Heartbeat"
}
