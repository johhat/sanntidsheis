package com

import (
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

type FloorOrders map[int]bool
type Orderset map[simdriver.BtnType]FloorOrders
type State struct{
	Last_passed_floor int
	Direction elevator.Direction_t
	Moving bool
	Orders Orderset
	Valid bool
	SequenceNumber int
	DoorOpen bool
}

///
/// ---External com---
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

type ClickEventMsg struct {
	Type     EventType
	Button   simdriver.ClickEvent //Etasje og opp/ned
	NewState State
}

type SensorEventMsg struct {
	Type     EventType
	NewState State
}

type InitialStateMsg struct {
	NewState State
}

type OrderAssignmentMsg struct {
	Button   simdriver.ClickEvent
	Assignee string
}

//
// Message wrapper used to convert to and from JSON
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
	w := wrappedMessage{Msg: message, MsgType: message.Type()}
	return w
}

func unmarshallToMessage(msgJSON *json.RawMessage, msgType, senderIp string) (Message, error) {

	var err error
	var m Message

	switch msgType {
	case "StateChange":
		temp := StateChange{}
		err = json.Unmarshal(*msgJSON, &temp)
		temp.Sender = senderIp
		m = temp
	case "OrderAssignmentMsg":
		temp := OrderAssignment{}
		err = json.Unmarshal(*msgJSON, &temp)
		temp.Sender = senderIp
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

//
// Broadcast message interfaces
//

type Message interface {
	Type() string
}

//
// Directed message interface
//

type DirectedMessage interface {
	GetRecieverIp() string
	Message
}

//
// Heartbeat format
//

func CreateHeartbeat(heartbeatNum int) Heartbeat {
	return Heartbeat{Code: HeartbeatCode, HeartbeatNum: heartbeatNum}
}

type Heartbeat struct {
	Code         string
	HeartbeatNum int
	Sender       string
}

func (m Heartbeat) Type() string {
	return "Heartbeat"
}

//
// Order assignment format
//

func (oa OrderAssignment) Type() string {
	return "OrderAssignmentMsg"
}

func (oa OrderAssignment) GetRecieverIp() string {
	return m.Assignee
}

//
// State change format
//

type StateChange struct {
	NewState State
	Event    EventType
	Sender   string
}

func (sc StateChange) Type() string {
	return "StateChange"
}

func (sc StateChange) GetSenderIp() string {
	return m.Sender
}
