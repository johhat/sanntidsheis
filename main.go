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
				log.Println(ev)
			case sensor := <-sensorEventChan:
				log.Println("Floor sensor signal", sensor)
			}
		}
	}()

	driver.Init(clickEventChan, sensorEventChan)
	driver.BasicElevator()
}
