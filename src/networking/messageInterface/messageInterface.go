package messageInterface

import (
	"encoding/json"
	"errors"
	"log"
)

//Wrapper to keep type information
type WrappedMessage struct {
	Msg     Message
	MsgType string
}

func (wrapped *WrappedMessage) Wrap(message Message) {
	wrapped.Msg = message
	wrapped.MsgType = message.Type()
}

func (wrapped WrappedMessage) Encode() []byte {
	bytes, _ := json.Marshal(wrapped)
	return bytes
}

func (wrapped *WrappedMessage) Decode(data []byte) error {

	tempMap := make(map[string]*json.RawMessage)

	json.Unmarshal(data, &tempMap)

	msgTypeJSON, msgTypeOk := tempMap["MsgType"]
	msgJSON, msgOk := tempMap["Msg"]

	if !msgTypeOk {
		return errors.New("Missing message type field")
	}

	if !msgOk {
		return errors.New("Missing message contents field")
	}

	var err error

	var msgType string
	err = json.Unmarshal(*msgTypeJSON, &msgType)

	if err != nil {
		return err
	}

	var m Message

	switch msgType {
	case "MockMessage":
		temp := MockMessage{}
		err = json.Unmarshal(*msgJSON, &temp)
		m = temp
	case "Hearbeat":
		temp := Heartbeat{}
		err = json.Unmarshal(*msgJSON, &temp)
		m = temp
	default:
		log.Println("Error in decode - type field not known")
		return errors.New("Error in decode - type field not known")
	}

	if err != nil {
		log.Println(err)
	}

	wrapped.Msg = m
	wrapped.MsgType = m.Type()

	return err
}

//The actual message interface - all message types must satisfy this interface
type Message interface {
	Encode() []byte
	Type() string
}

//Mock format

type MockMessage struct {
	Number    int
	Text      string
	MockState State
}

func (m MockMessage) Encode() []byte {
	bytes, _ := json.Marshal(m)
	return bytes
}

func (m *MockMessage) Decode(data []byte) {
	json.Unmarshal(data, m)
}

func (m MockMessage) Type() string {
	return "MockMessage"
}

//Heartbeat format

type Heartbeat struct {
	Msg string
}

func (m Heartbeat) Encode() []byte {
	bytes, _ := json.Marshal(m)
	return bytes
}

func (m Heartbeat) Type() string {
	return "Heartbeat"
}

//Mock struct to see if this will be unmarshalled correctly when it is contained in MockMessage
type State struct {
	LastFloor        int
	CurrentDirection int
	SomeRandomText   string
}
