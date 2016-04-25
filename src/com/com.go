package com

import (
	"../elevator"
	"../simdriver"
	"encoding/json"
	"errors"
	"log"
)

const HeartbeatCode = "SecretString"

type LiftEvents struct {
	FloorReached chan int
	StopButton   chan bool
	Obstruction  chan bool
}

///
/// State object definition. This is placed here since it is communicated to other nodes.
///

type FloorOrders map[int]bool
type Orderset map[simdriver.BtnType]FloorOrders

type State struct {
	LastPassedFloor int
	Direction       elevator.Direction_t
	Moving          bool
	Orders          Orderset
	Valid           bool
	SequenceNumber  int
	DoorOpen        bool
}

///
/// Event com stuff
///

type EventType int

const (
	//Click events
	NewExternalOrder EventType = iota
	NewInternalOrder
	SelfAssignedOrder

	//Sensor events
	PassingFloor
	DoorOpenedByInternalOrder
	StoppingToFinishOrder
	LeavingFloor
)

//
// Msg interfaces
//

type Message interface {
	MsgType() string
}

type DirectedMessage interface {
	GetRecieverIp() string
	Message
}

//
// Click Event Msg implementation
//

type ClickEventMsg struct {
	Type     EventType
	Button   simdriver.ClickEvent //Etasje og opp/ned
	NewState State
	Sender   string
}

func (m ClickEventMsg) MsgType() string {
	return "ClickEventMsg"
}

//
// Sensor Event Msg implementation
//

type SensorEventMsg struct {
	Type     EventType
	NewState State
	Sender   string
}

func (m SensorEventMsg) MsgType() string {
	return "SensorEventMsg"
}

//
// Initial State Msg implementation
//

type InitialStateMsg struct {
	NewState State
	Sender   string
}

func (m InitialStateMsg) MsgType() string {
	return "InitialStateMsg"
}

//
// Heartbeat Msg implementation
//

func CreateHeartbeat(heartbeatNum int) Heartbeat {
	return Heartbeat{Code: HeartbeatCode, HeartbeatNum: heartbeatNum}
}

type Heartbeat struct {
	Code         string
	HeartbeatNum int
	Sender       string
}

func (m Heartbeat) MsgType() string {
	return "Heartbeat"
}

//
// Order Assignment Directed msg implementation
//

type OrderAssignmentMsg struct {
	Button   simdriver.ClickEvent
	Assignee string
}

func (m OrderAssignmentMsg) MsgType() string {
	return "OrderAssignmentMsg"
}

func (m OrderAssignmentMsg) GetRecieverIp() string {
	return m.Assignee
}

//
// Message wrapper used to convert messages to and from JSON
//

type wrappedMessage struct {
	Msg     Message
	MsgType string
}

func (wrapped wrappedMessage) Encode() []byte {
	bytes, _ := json.Marshal(wrapped)
	return bytes
}

func WrapMessage(message Message) wrappedMessage {
	w := wrappedMessage{Msg: message, MsgType: message.MsgType()}
	return w
}

func unmarshallToMessage(msgJSON *json.RawMessage, msgType, senderIp string) (Message, error) {

	var err error
	var m Message

	switch msgType {
	case "ClickEventMsg":
		temp := ClickEventMsg{}
		err = json.Unmarshal(*msgJSON, &temp)
		temp.Sender = senderIp
		m = temp
	case "SensorEventMsg":
		temp := SensorEventMsg{}
		err = json.Unmarshal(*msgJSON, &temp)
		temp.Sender = senderIp
		m = temp
	case "InitialStateMsg":
		temp := InitialStateMsg{}
		err = json.Unmarshal(*msgJSON, &temp)
		temp.Sender = senderIp
		m = temp
	case "OrderAssignmentMsg":
		temp := OrderAssignmentMsg{}
		err = json.Unmarshal(*msgJSON, &temp)
		m = temp
	case "Heartbeat":
		temp := Heartbeat{}
		err = json.Unmarshal(*msgJSON, &temp)
		temp.Sender = senderIp
		m = temp
	default:
		return nil, errors.New("Error in decode - type field not known")
	}

	return m, err
}

func DecodeWrappedMessage(data []byte, senderIp string) (Message, error) {

	var err error

	tempMap := make(map[string]*json.RawMessage)
	err = json.Unmarshal(data, &tempMap)

	if err != nil {
		return nil, err
	}

	msgTypeJSON, msgTypeOk := tempMap["MsgType"]

	if !msgTypeOk {
		return nil, errors.New("Missing message type field")
	}

	msgJSON, msgOk := tempMap["Msg"]

	if !msgOk {
		return nil, errors.New("Missing message contents field")
	}

	var msgType string
	err = json.Unmarshal(*msgTypeJSON, &msgType)

	if err != nil {
		return nil, err
	}

	var m Message

	m, err = unmarshallToMessage(msgJSON, msgType, senderIp)

	if err != nil {
		log.Println(err)
	}

	return m, err
}
