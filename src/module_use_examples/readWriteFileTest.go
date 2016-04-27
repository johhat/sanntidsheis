package main

import(
	"../statetype"
	"fmt"
	"strconv"
	"os"
)

func main(){
	statetype.SaveInternalOrder(0)
	if _, err := os.Stat("internalOrder" + strconv.Itoa(0)); !os.IsNotExist(err) {
		fmt.Println(true)
	} else {
		fmt.Println(false)
	}
}