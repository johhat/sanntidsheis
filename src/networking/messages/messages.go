package messages

import (
	"encoding/json"
	"errors"
	"log"
)

//
// Message wrapper used to convert to and from JSON
//

type wrappedMessage struct {
	Msg     Message
	MsgType string
}

func WrapMessage(message Message) wrappedMessage {
	w := wrappedMessage{Msg: message, MsgType: message.Type()}
	return w
}

func DecodeWrappedMessage(data []byte, senderIp string) (Message, error) {

	var err error

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
		temp.Sender = senderIp
		err = json.Unmarshal(*msgJSON, &temp)
		m = temp
	case "Heartbeat":
		temp := Heartbeat{}
		temp.Sender = senderIp
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

//
// Broadcast message interfaces
//

type Message interface {
	GetSenderIp() string
	Type() string
}

//
// Directed message interface
//

type DirectedMessage interface {
	GetRecieverIp() string
	GetSenderIp() string
	Type() string
}

//
// Mock broadcast format
//

type MockMessage struct {
	Number int
	Text   string
	Sender string
}

func (m MockMessage) Type() string {
	return "MockMessage"
}

func (m MockMessage) GetSenderIp() string {
	return m.Sender
}

//
// Mock directed format
//

type MockDirectedMessage struct {
	Number   int
	Text     string
	Sender   string
	Reciever string
}

func (m MockDirectedMessage) Type() string {
	return "MockDirectedMessage"
}

func (m MockDirectedMessage) GetSenderIp() string {
	return m.Sender
}

func (m MockDirectedMessage) GetRecieverIp() string {
	return m.Reciever
}

//
// Heartbeat format - an actual broadcast format
//

func CreateHeartbeat(heartbeatNum int) Heartbeat {
	return Heartbeat{Code: "MartinOgJohanSinHeis", HeartbeatNum: heartbeatNum}
}

type Heartbeat struct {
	Code         string
	HeartbeatNum int
	Sender       string
}

func (m Heartbeat) Type() string {
	return "Heartbeat"
}

func (m Heartbeat) GetSenderIp() string {
	return m.Sender
}
