package validation

import (
	"fmt"
	"strconv"
)

type InRangeValidator struct {
	min, max float64
}

func (v InRangeValidator) GetName() string {
	return "in-range"
}

func (v *InRangeValidator) Configure(cfg map[string]interface{}) error {
	if min, ok := cfg["min"]; ok {
		//f, err := strconv.ParseFloat(min.(string), 64)
		//if nil != err {
		//	return fmt.Errorf("min parameter invalid value")
		//}
		//v.min = f
		v.min = float64(min.(int))
	} else {
		return fmt.Errorf("min parameter expected")
	}
	if max, ok := cfg["max"]; ok {
		//f, err := strconv.ParseFloat(max.(string), 64)
		//if nil != err {
		//	return fmt.Errorf("max parameter invalid value")
		//}
		//v.max = f
		v.max = float64(max.(int))
	} else {
		return fmt.Errorf("max parameter expected")
	}
	return nil
}

func (v InRangeValidator) Check(name string, value interface{}) (bool, error) {
	var val float64 = 0
	switch value.(type) {
	case string:
		f, err := strconv.ParseFloat(value.(string), 64)
		if nil == err {
			val = f
		}
	case int64:
		val = float64(value.(int64))
	}

	if val < v.min || v.max < val {
		return false, fmt.Errorf("%s : value %v is not in range %f..%f",
			name, val, v.min, v.max)
	}

	return true, nil
}
