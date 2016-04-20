package main

import (
	"../networking/messageInterface"
	"fmt"
)

func main() {
	fmt.Println("Wecome to this test")
	m := messageInterface.MockMessage{Number: 10, Text: "Hello!"}
	fmt.Println(m)
	fmt.Println("In json this becomes", m.Encode())
	fmt.Println("In text this translates to", string(m.Encode()))
	m2 := messageInterface.MockMessage{}
	m2.Decode(m.Encode())
	fmt.Println("Which decodes to:", m)

	fmt.Println("Now we try to decode a general message")
	messageInterface.DecodeMessage(m.Encode())
}
