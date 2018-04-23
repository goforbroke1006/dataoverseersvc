package repo

import (
	"database/sql"
	"time"
)

type User struct {
	Id    sql.NullInt64
	Name  sql.NullString
	Email sql.NullString
}

type MetalScanner struct {
	valid bool
	Value interface{}
}

func (scanner *MetalScanner) getBytes(src interface{}) []byte {
	if a, ok := src.([]uint8); ok {
		return a
	}
	return nil
}

func (scanner *MetalScanner) Scan(src interface{}) error {
	switch src.(type) {
	case int64:
		if value, ok := src.(int64); ok {
			scanner.Value = value
			scanner.valid = true
		}
	case float64:
		if value, ok := src.(float64); ok {
			scanner.Value = value
			scanner.valid = true
		}
	case bool:
		if value, ok := src.(bool); ok {
			scanner.Value = value
			scanner.valid = true
		}
	case string:
		value := scanner.getBytes(src)
		scanner.Value = string(value)
		scanner.valid = true
	case []byte:
		value := scanner.getBytes(src)
		scanner.Value = value
		scanner.valid = true
	case time.Time:
		if value, ok := src.(time.Time); ok {
			scanner.Value = value
			scanner.valid = true
		}
	case nil:
		scanner.Value = nil
		scanner.valid = true
	}
	return nil
}
