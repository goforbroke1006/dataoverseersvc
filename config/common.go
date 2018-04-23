package config

import (
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
)

type connection struct {
	Driver   string `yaml:""`
	Host     string `yaml:""`
	Port     int    `yaml:""`
	Name     string `yaml:""`
	Username string `yaml:"user"`
	Password string `yaml:"pass"`
}

type Task struct {
	Query   string `yaml:"query"`
	FieldId string `yaml:"fieldId"`
	Subjects []struct {
		Type    string                 `yaml:"type"`
		Columns []string               `yaml:"columns"`
		Params  map[string]interface{} `yaml:"params"`
	} `yaml:"subjects"`
}

type Configuration struct {
	AdminEmail string `yaml:"admin_email"`
	Connection connection `yaml:"connection"`
	Timeout    int64      `yaml:"timeout"`
	Tasks      []Task     `yaml:"tasks"`
	Mailer struct {
		Type     string
		Host     *string `yaml:""`
		Port     *int    `yaml:""`
		Username string  `yaml:"user"`
		Password string  `yaml:"pass"`
	} `yaml:"mailer"`
	Redis struct {
		Host     string `yaml:""`
		Port     int    `yaml:""`
		Password string  `yaml:"pass"`
	} `yaml:"redis"`
}

func LoadConfig(filename string) (*Configuration, error) {
	fi, err := os.OpenFile(filename, os.O_RDONLY, 0777)
	if nil != err {
		return nil, err
	}
	data, err := ioutil.ReadAll(fi)
	if nil != err {
		return nil, err
	}
	c := &Configuration{}
	err = yaml.Unmarshal(data, c)
	if nil != err {
		return nil, err
	}
	return c, nil
}
