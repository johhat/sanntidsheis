package networking

import (
	"../com"
	"log"
	"time"
)

const readConfirmationTimeout = 1 * time.Second

type msgAndHash struct {
	msg              com.Message
	readConfirmation chan bool
	hash             string
}

type awaitingConfirmationTuple struct {
	msg              com.Message
	readConfirmation chan bool
	read             chan bool
}

func readConfirmationHandler(awaitConfirmation <-chan msgAndHash, registerConfirmationHash <-chan string) {

	//TODO: Lag et map for meldinger som har timet ut der meldinger slettes etter ett min?
	awaitingConfirmationMap := make(map[string]awaitingConfirmationTuple)
	timeout := make(chan string)

	for {
		select {
		case elem := <-awaitConfirmation:

			localReadConfirmChan := make(chan bool)

			awaitingConfirmationMap[elem.hash] = awaitingConfirmationTuple{
				msg:              elem.msg,
				readConfirmation: elem.readConfirmation,
				read:             localReadConfirmChan,
			}

			go func() {
				select {
				case <-time.After(readConfirmationTimeout):
					timeout <- elem.hash
					if elem.readConfirmation != nil {
						elem.readConfirmation <- false
						close(elem.readConfirmation)
					}
				case <-localReadConfirmChan:
					if elem.readConfirmation != nil {
						elem.readConfirmation <- true
						close(elem.readConfirmation) //TODO: Check if this is ok if we have several listeners
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
