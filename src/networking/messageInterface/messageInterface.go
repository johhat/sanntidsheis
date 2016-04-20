package messageInterface

import (
	"encoding/json"
	"fmt"
)

//Interface

type Message interface {
	Encode()
	Decode()
}

func DecodeMessage(encodedMsg []byte) {

	tempMap := make(map[string]*json.RawMessage)

	json.Unmarshal(encodedMsg, &tempMap)

	for key, value := range tempMap {
		fmt.Println(key, value)
	}
}

//Mock format

type MockMessage struct {
	Number int
	Text   string
}

func (m MockMessage) Encode() []byte {
	bytes, _ := json.Marshal(m)
	return bytes
}

func (m *MockMessage) Decode(data []byte) {
	json.Unmarshal(data, m)
}

func (m MockMessage) String() string {
	return fmt.Sprintf("This mock message has number %v and text %s \n", m.Number, m.Text)
}

//Heartbeat format

type Heartbeat struct {
	Msg string
}

func (m Heartbeat) Encode() []byte {
	bytes, _ := json.Marshal(m)
	return bytes
}

func (m *Heartbeat) Decode(data []byte) {
	json.Unmarshal(data, m)
}
