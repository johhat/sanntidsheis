package main

import (
	"../networking/messageInterface"
	"fmt"
)

func main() {
	fmt.Println("Wecome to this test")

	s := messageInterface.State{1, 2, "Hello from state struct"}

	m := messageInterface.MockMessage{Number: 10, Text: "Hello!", MockState: s}
	fmt.Println(m)
	fmt.Println("In json this becomes", m.Encode())
	fmt.Println("In text this translates to", string(m.Encode()))
	m2 := messageInterface.MockMessage{}
	m2.Decode(m.Encode())
	fmt.Println("Which decodes to:", m2)

	fmt.Println("Now we try to wrap the message")
	wrap := messageInterface.WrappedMessage{}
	wrap.Wrap(m)
	fmt.Println("The wrapped data is", wrap)
	fmt.Println("In json this is", string(wrap.Encode()))
	wrap2 := messageInterface.WrappedMessage{}
	wrap2.Decode(wrap.Encode())
	fmt.Println("This decodes to", wrap2)
}
