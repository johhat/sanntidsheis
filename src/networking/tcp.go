package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

const (
	CONNECTION_port = ":3333"
	CONNECTION_type = "tcp"
)

func tcpSend() {

}

func tcpRecv() {

}

func tcpConnect() {

}

func TcpWorker() {

}

func Connect(host string) {

	conn, err := net.Dial(CONNECTION_type, net.JoinHostPort(host, CONNECTION_port))

	if err == nil {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Text to send: ")
		text, _ := reader.ReadString('n')

		fmt.Fprintf(conn, text + "\n")
		
	}

	status, err := bufio.NewReader(conn).ReadString('\n')
}

func Listen() {
	ln, err := net.Listen(CONNECTION_TYPE, CONNECTION_port)

	if err != nil {
		// handle error
	}

	i := 0

	for {
		conn, err := ln.Accept()
		if err == nil {
			message, _ := bufio.NewReader(conn).ReadString('\n')
			fmt.Print("Message Received:", string(message))
			newmessage:= "This is a new message with id " + string(i) + "\n"
			conn.Write([]byte(newmessage)
		} else {
			fmt.Println("Woops something went wrong indeed")
		}
	}
}

func main() {

	ifat, _ := net.InterfaceAddrs()
	interfaces, _ := net.Interfaces()

	fmt.Println(ifat)
	fmt.Println(interfaces)
}
