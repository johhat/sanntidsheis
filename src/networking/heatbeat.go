package networking

import (
	"../com"
	"log"
	"time"
)

func udpSendHeartbeats(udpBroadcastMsg chan<- []byte, quit <-chan bool) {

	udpHeartbeatNum := 0
	udpHeatbeatTick := time.Tick(udpHeartbeatInterval)

	for {
		select {
		case <-udpHeatbeatTick:
			m := com.CreateHeartbeat(udpHeartbeatNum)
			w := com.WrapMessage(m)

			data, err := w.Encode()

			if err != nil {
				log.Println("Error when encoding Heartbeat. Err:", err, ". Message:", m)
			}

			udpBroadcastMsg <- data
			udpHeartbeatNum++
		case <-quit:
			return
		}
	}
}

func registerHeartbeat(heartbeats map[string]int, heartbeatNum int, sender string, connectionType string) {

	//TODO: Clear array on disconnect
	prev, ok := heartbeats[sender]

	if !ok {
		heartbeats[sender] = heartbeatNum
		return
	} else {
		heartbeats[sender] = heartbeatNum
	}

	switch {
	case prev > heartbeatNum:
		log.Printf("Delayed %s heartbeat from %s. Previous HB: %v Current HB: %v \n", connectionType, sender, prev, heartbeatNum)
	case prev == heartbeatNum:
		log.Printf("Duplicate %s heartbeat from %s. Previous HB: %v Current HB: %v \n", connectionType, sender, prev, heartbeatNum)
	case prev+1 != heartbeatNum:
		log.Printf("Missing %s heartbeat(s) from %s. Previous HB: %v Current HB: %v \n", connectionType, sender, prev, heartbeatNum)
	}
}
