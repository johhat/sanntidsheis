package networking

import (
	"../com"
	"./tcp"
	"log"
	"time"
)

const readConfirmationTimeout = 1 * time.Second

type expectReadConfirmationData struct {
	msg                  com.Message
	readConfirmationChan chan bool
	hash                 string
}

type awaitingConfirmationData struct {
	msg                  com.Message
	readConfirmationChan chan bool
	read                 chan bool
}

func sendReadConfirmationHandler(outgoingReadConfs <-chan com.ReadConfirmationMsg,
	tcpSendMsg chan<- tcp.RawMessage) {

	for msg := range outgoingReadConfs {

		w := com.WrapMessage(msg)

		data, err := w.Encode()

		if err != nil {
			log.Println("Error when encoding read confirmation msg. Err:", err, ". Message:", msg)
			continue
		}

		tcpSendMsg <- tcp.RawMessage{Data: data, Ip: msg.Reciever}
	}
}

func readConfirmationHandler(awaitConfirmation <-chan expectReadConfirmationData,
	registerConfirmationHash <-chan string) {

	//TODO: Lag et map for meldinger som har timet ut der meldinger slettes etter ett min?
	awaitingConfirmationMap := make(map[string]awaitingConfirmationData)
	timeout := make(chan string)

	for {
		select {
		case elem := <-awaitConfirmation:

			localReadConfirmChan := make(chan bool)

			awaitingConfirmationMap[elem.hash] = awaitingConfirmationData{
				msg:                  elem.msg,
				readConfirmationChan: elem.readConfirmationChan,
				read:                 localReadConfirmChan,
			}

			go func() {
				select {
				case <-time.After(readConfirmationTimeout):
					timeout <- elem.hash
					//Send resultat på kanalen manager har sendt til network-modul
					if elem.readConfirmationChan != nil {
						elem.readConfirmationChan <- false
						close(elem.readConfirmationChan)
					}
				case <-localReadConfirmChan:
					//Send resultat på kanalen manager har sendt til network-modul
					if elem.readConfirmationChan != nil {
						elem.readConfirmationChan <- true
						close(elem.readConfirmationChan) //TODO: Check if this is ok if we have several listeners
					}
				}
			}()

		case hash := <-registerConfirmationHash:

			val, ok := awaitingConfirmationMap[hash]

			if !ok {
				log.Println("readConfHandler: Tried to remove recieved message that was not present in map")
				continue
			}
			close(val.read)
			delete(awaitingConfirmationMap, hash)

		case hash := <-timeout:
			val, ok := awaitingConfirmationMap[hash]

			log.Println("Message timed out", val.msg)

			if !ok {
				log.Println("readConfHandler: Tried to remove timed out message that was not present in map")
				continue
			}
			close(val.read)
			delete(awaitingConfirmationMap, hash)
		}
	}
}
