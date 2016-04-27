package statetype

import (
	driver "../simdriver"
	"encoding/json"
	"strconv"
)

type Orderset map[driver.BtnType]FloorOrders

func (orderSet Orderset) MarshalJSON() ([]byte, error) {

	auxMap := make(map[string]FloorOrders)

	for key, val := range orderSet {
		auxMap[strconv.Itoa(int(key))] = val
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

		resultMap[driver.BtnType(newKey)] = val
	}

	*orderSet = resultMap

	return err
}
