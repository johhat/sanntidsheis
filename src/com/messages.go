package com

import (
	"../driver"
	"../state"
)

type Message interface {
	MsgType() string
	GetSenderIp() string
}


type DirectedMessage interface {
	GetRecieverIp() string
	Message
}


type OrderEventMsg struct {
	Button   driver.ClickEvent
	NewState state.State
	Sender   string
}

func (m OrderEventMsg) MsgType() string {
	return "OrderEventMsg"
}

func (m OrderEventMsg) GetSenderIp() string {
	return m.Sender
}


type SensorEventMsg struct {
	Type     EventType
	NewState state.State
	Sender   string
}

func (m SensorEventMsg) MsgType() string {
	return "SensorEventMsg"
}

func (m SensorEventMsg) GetSenderIp() string {
	return m.Sender
}


type InitialStateMsg struct {
	NewState state.State
	Sender   string
}

func (m InitialStateMsg) MsgType() string {
	return "InitialStateMsg"
}

func (m InitialStateMsg) GetSenderIp() string {
	return m.Sender
}


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
