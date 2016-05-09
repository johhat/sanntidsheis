package networking

import (
	"../com"
	"log"
	"time"
)

func udpSendHeartbeats(udpBroadcastMsg chan<- []byte) chan<- bool {

	quit := make(chan bool)

	go func() {
		udpHeartbeatNum := 0
		udpHeatbeatTick := time.Tick(udpHeartbeatInterval)

		for {
			select {
			case <-udpHeatbeatTick:
				msg := com.CreateHeartbeat(udpHeartbeatNum)

				data, err := com.EncodeMessage(msg)

				if err != nil {
					log.Println("Error when encoding Heartbeat. Err:", err, ". Message:", msg)
				}

				udpBroadcastMsg <- data
				udpHeartbeatNum++
			case <-quit:
				return
			}
		}
	}()
	return quit
}

func registerHeartbeat(heartbeats map[string]int, heartbeatNum int, sender string, connectionType string) {

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
