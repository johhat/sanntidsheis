package main

import (
	"encoding/json"
	"log"
	"strconv"
)

type Orderset map[int]FloorOrders

func (orderSet Orderset) MarshalJSON() ([]byte, error) {

	auxMap := make(map[string]FloorOrders)

	for key, val := range orderSet {
		auxMap[strconv.Itoa(key)] = val
	}

	return json.Marshal(auxMap)
}

func (orderSet *Orderset) UnmarshalJSON(data []byte) error {
	var err error

	auxMap := make(map[string]FloorOrders)
	err = json.Unmarshal(data, &auxMap)

	resultMap := make(Orderset)

	if err != nil {
		return err
	}

	var newKey int

	for key, val := range auxMap {

		newKey, err = strconv.Atoi(key)

		if err != nil {
			return err
		}

		resultMap[newKey] = val
	}

	*orderSet = resultMap

	return err
}

type FloorOrders map[int]bool

func (fo FloorOrders) MarshalJSON() ([]byte, error) {

	auxMap := make(map[string]bool)

	for key, val := range fo {
		auxMap[strconv.Itoa(key)] = val
	}

	return json.Marshal(auxMap)
}

func (fo *FloorOrders) UnmarshalJSON(data []byte) error {

	var err error

	auxMap := make(map[string]bool)
	err = json.Unmarshal(data, &auxMap)

	resultMap := make(FloorOrders)

	if err != nil {
		return err
	}

	var newKey int

	for key, val := range auxMap {

		newKey, err = strconv.Atoi(key)

		if err != nil {
			return err
		}

		resultMap[newKey] = val
	}

	*fo = resultMap

	return err
}

func main() {
	log.Println("Start of test")

	sourceMap := make(Orderset)
	resultMap := make(Orderset)

	sourceMap[0] = FloorOrders{
		0: true,
		1: false,
	}

	sourceMap[1] = FloorOrders{
		1: true,
		0: false,
	}

	log.Println("Sourcemap:", sourceMap)

	data, err1 := json.Marshal(sourceMap)

	if err1 != nil {
		log.Println("Error in marshal:", err1)
	}

	log.Println("JSON encoded sourceMap:", string(data))

	err2 := json.Unmarshal(data, &resultMap)

	if err2 != nil {
		log.Println("Error in marshal:", err2)
	}

	log.Println("Resultmap:", resultMap)
}
