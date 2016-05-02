package main

import (
	"fmt"
	"os"
	"strconv"
)

func main() {
	floor := 3
	file := "internalOrder" + strconv.Itoa(floor)
	if _, err := os.Stat(file); os.IsExist(err) {
		fmt.Println("Exists is positive 1")
		//os.Remove(file)
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		fmt.Println("NotExists is negative 1")
		//os.Remove(file)
	}
	_, _ = os.Create("internalOrder" + strconv.Itoa(floor))
	if _, err := os.Stat(file); os.IsExist(err) {
		fmt.Println("Exists is positive 2")
		//os.Remove(file)
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		fmt.Println("NotExists is negative 2")
		//os.Remove(file)
	}

}
