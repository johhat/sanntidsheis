package state

import (
	"encoding/json"
	"strconv"
)

type FloorOrders map[int]bool

func (fo FloorOrders) MarshalJSON() ([]byte, error) {

	auxMap := make(map[string]bool)

	for key, val := range fo {
		auxMap[strconv.Itoa(key)] = val
	}

	return json.Marshal(auxMap)
}

func (fo *FloorOrders) UnmarshalJSON(data []byte) error {

	auxMap := make(map[string]bool)
	err := json.Unmarshal(data, &auxMap)
	resultMap := make(FloorOrders)

	if err != nil {
		return err
	}

	for key, val := range auxMap {
		newKey, err := strconv.Atoi(key)

		if err != nil {
			return err
		}

		resultMap[newKey] = val
	}

	*fo = resultMap

	return err
}
