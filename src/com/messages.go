package com

import (
	driver "../simdriver"
	s "../statetype"
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
