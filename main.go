package main

import (
	"./driver"
	"log"
)

func main() {

	clickEventChan := make(chan driver.ClickEvent)
	sensorEventChan := make(chan int)

	go func() {
		for {
			select {
			case ev := <-clickEventChan:
				log.Println("Click event. Floor", ev.Floor, "Type", ev.Type)
			case sensor := <-sensorEventChan:
				log.Println("Floor sensor signal", sensor)
			}
		}
	}()

	driver.Init(clickEventChan, sensorEventChan)
	driver.BasicElevator()
}
