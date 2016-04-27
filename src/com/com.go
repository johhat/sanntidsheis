package com

import (
	driver "../simdriver"
	s "../statetype"
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
/// Event com stuff
///

type EventType int

const (
	//Order events
	NewExternalOrder EventType = iota
	NewInternalOrder

	//Sensor events
	PassingFloor
	DoorOpenedByInternalOrder
	StoppingToFinishOrder
	LeavingFloor
	DoorClosed
	DirectionChanged
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

type OrderEventMsg struct {
	Button   driver.ClickEvent //Etasje og opp/ned
	NewState s.State
	Sender   string
}

func (m OrderEventMsg) MsgType() string {
	return "OrderEventMsg"
}

func CreateOrderEventMsg(btn driver.ClickEvent, newState s.State) OrderEventMsg {
	return OrderEventMsg{Button: btn, NewState: newState}
}

//
// Sensor Event Msg implementation
//

type SensorEventMsg struct {
	Type     EventType
	NewState s.State
	Sender   string
}

func (m SensorEventMsg) MsgType() string {
	return "SensorEventMsg"
}

func CreateSensorEventMsg(eventType EventType, newState s.State) SensorEventMsg {
	return SensorEventMsg{Type: eventType, NewState: newState}
}

//
// Initial s.State Msg implementation
//

type InitialStateMsg struct {
	NewState s.State
	Sender   string
}

func (m InitialStateMsg) MsgType() string {
	return "InitialStateMsg"
}

func CreateInitialStateMsg(newState s.State) InitialStateMsg {
	return InitialStateMsg{NewState: newState}
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
	Button   driver.ClickEvent
	Assignee string
}

func (m OrderAssignmentMsg) MsgType() string {
	return "OrderAssignmentMsg"
}

func (m OrderAssignmentMsg) GetRecieverIp() string {
	return m.Assignee
}

func CreateOrderAssignmentMsg(btn driver.ClickEvent, assignee string) OrderAssignmentMsg {
	return OrderAssignmentMsg{Button: btn, Assignee: assignee}
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
	case "OrderEventMsg":
		temp := OrderEventMsg{}
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
