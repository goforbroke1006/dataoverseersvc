package validation

import (
	"fmt"
	"reflect"
)

var list = map[string]Validator{}

func Register(validator Validator) (error) {
	if _, ok := list[validator.GetName()]; ok {
		fmt.Errorf(
			"validator with name \"%s\" already registered",
			validator.GetName(),
		)
	}
	list[validator.GetName()] = validator
	return nil
}

type Validator interface {
	GetName() string
	Configure(cfg map[string]interface{}) (error)
	Check(name string, value interface{}) (bool, error)
}

type subject struct {
	validator Validator
	fields    []string
}

type ValidationHub struct {
	subjects []subject
}

func (hub *ValidationHub) Setup(
	vType string,
	columns []string,
	params map[string]interface{},
) {
	if validator, ok := list[vType]; ok {
		nV := reflect.New(
			reflect.ValueOf(validator).Elem().Type(),
		).Interface().(Validator)
		nV.Configure(params)
		hub.subjects = append(hub.subjects, subject{
			validator: nV,
			fields:    columns,
		})
	}
	fmt.Errorf("validator with name \"%s\" not found", vType)
}

func (hub *ValidationHub) Validate(name string, val interface{}) (bool, error) {
	for _, s := range hub.subjects {
		for _, c := range s.fields {
			if name == c {
				return s.validator.Check(name, val)
			}
		}
	}
	return true, nil
}
