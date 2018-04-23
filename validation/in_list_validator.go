package validation

import (
	"fmt"
)

type InListValidator struct {
	list []interface{}
}

func (v InListValidator) GetName() string {
	return "in-list"
}

func (v *InListValidator) Configure(cfg map[string]interface{}) error {
	if l, ok := cfg["list"]; ok {
		v.list = l.([]interface{})
		return nil
	} else {
		return fmt.Errorf("list parameter expected")
	}
}

func (v InListValidator) Check(name string, value interface{}) (bool, error) {
	for _, av := range v.list {
		if av == value {
			return true, nil
		}
	}
	return false, fmt.Errorf("%s : unexpected value %v", name, value)
}
