package com

import (
	"encoding/json"
	"errors"
	"log"
)

type wrappedMessage struct {
	Msg     Message
	MsgType string
}

func (wrapped wrappedMessage) encode() ([]byte, error) {
	return json.Marshal(wrapped)
}

func wrapMessage(message Message) wrappedMessage {
	return wrappedMessage{
		Msg:     message,
		MsgType: message.MsgType(),
	}
}

func EncodeMessage(message Message) ([]byte, error) {
	return wrapMessage(message).encode()
}

func unmarshallToMessage(msgJSON *json.RawMessage, msgType, senderIp string) (Message, error) {

	var err error
	var m Message

	switch msgType {
	case "Heartbeat":
		temp := Heartbeat{}
		err = json.Unmarshal(*msgJSON, &temp)
		temp.Sender = senderIp
		m = temp
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
		temp.Sender = senderIp
		m = temp
	default:
		return nil, errors.New("Error in decode - type field not known")
	}

	return m, err
}

func DecodeMessage(data []byte, senderIp string) (Message, error) {

	var err error

	tempMap := make(map[string]*json.RawMessage)
	err = json.Unmarshal(data, &tempMap)

	if err != nil {
		log.Println("Error when decoding to tempmap. Err:", err, ". Data:", string(data))
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
		log.Println("Error when decoding msgType. Err:", err)
		return nil, err
	}

	var m Message

	m, err = unmarshallToMessage(msgJSON, msgType, senderIp)

	if err != nil {
		log.Println("Error when decoding msg contents. Err: ", err)
	}

	return m, err
}
