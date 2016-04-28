package com

import (
	"encoding/json"
	"errors"
	"log"
	"math/rand"
	"strconv"
	"time"
)

type wrappedMessage struct {
	Msg     Message
	MsgType string
	MsgHash string
}

func (wrapped wrappedMessage) Encode() ([]byte, error) {
	return json.Marshal(wrapped)
}

func WrapMessage(message Message) wrappedMessage {

	var hash string
	data, err := json.Marshal(message)

	if err != nil {
		log.Println("Msg encode for hashing failed. Err:", err, ". Using fallback")
		hash = GetHashString([]byte(time.Now().String() + strconv.Itoa(rand.Int())))
	} else {
		hash = GetHashString(data)
	}

	w := wrappedMessage{
		Msg:     message,
		MsgType: message.MsgType(),
		MsgHash: hash,
	}

	return w
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
	case "ReadConfirmationMsg":
		temp := ReadConfirmationMsg{}
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

func DecodeWrappedMessage(data []byte, senderIp string) (Message, string, error) {

	var err error

	tempMap := make(map[string]*json.RawMessage)
	err = json.Unmarshal(data, &tempMap)

	if err != nil {
		log.Println("Error when decoding to tempmap. Err:", err, ". Data:", data)
		return nil, "", err
	}

	msgTypeJSON, msgTypeOk := tempMap["MsgType"]

	if !msgTypeOk {
		return nil, "", errors.New("Missing message type field")
	}

	msgJSON, msgOk := tempMap["Msg"]

	if !msgOk {
		return nil, "", errors.New("Missing message contents field")
	}

	msgHashJSON, msgHashOk := tempMap["MsgHash"]

	if !msgHashOk {
		return nil, "", errors.New("Missing message hash field")
	}

	var msgType string
	err = json.Unmarshal(*msgTypeJSON, &msgType)

	if err != nil {
		log.Println("Error when decoding msgType. Err:", err)
		return nil, "", err
	}

	var msgHash string
	err = json.Unmarshal(*msgHashJSON, &msgHash)

	if err != nil {
		log.Println("Error when decoding msgHash. Err:", err)
		return nil, "", err
	}

	var m Message

	m, err = unmarshallToMessage(msgJSON, msgType, senderIp)

	if err != nil {
		log.Println("Error when decoding msg contents. Err: ", err)
	}

	return m, msgHash, err
}
