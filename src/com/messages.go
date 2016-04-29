package com

import (
	driver "../driver"
	s "../statetype"
)

//
// Msg interfaces
//

type Message interface {
	MsgType() string
	GetSenderIp() string
}

type DirectedMessage interface {
	GetRecieverIp() string
	Message
}

//
// Click Event Msg implementation
//

type OrderEventMsg struct {
	Button   driver.ClickEvent
	NewState s.State
	Sender   string
}

func (m OrderEventMsg) MsgType() string {
	return "OrderEventMsg"
}

func (m OrderEventMsg) GetSenderIp() string {
	return m.Sender
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

func (m SensorEventMsg) GetSenderIp() string {
	return m.Sender
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

func (m InitialStateMsg) GetSenderIp() string {
	return m.Sender
}

//
// Heartbeat Msg implementation
//

const HeartbeatCode = "SecretString"

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

func (m Heartbeat) GetSenderIp() string {
	return m.Sender
}

//
// Order Assignment Directed msg implementation
//

type OrderAssignmentMsg struct {
	Button   driver.ClickEvent
	Assignee string
	Sender   string
}

func (m OrderAssignmentMsg) MsgType() string {
	return "OrderAssignmentMsg"
}

func (m OrderAssignmentMsg) GetSenderIp() string {
	return m.Sender
}

func (m OrderAssignmentMsg) GetRecieverIp() string {
	return m.Assignee
}

//
// Read confirmation Directed msg implementation
//

type ReadConfirmationMsg struct {
	Hash     string
	Reciever string
	Sender   string
}

func (m ReadConfirmationMsg) MsgType() string {
	return "ReadConfirmationMsg"
}

func (m ReadConfirmationMsg) GetSenderIp() string {
	return m.Sender
}

func (m ReadConfirmationMsg) GetRecieverIp() string {
	return m.Reciever
}
