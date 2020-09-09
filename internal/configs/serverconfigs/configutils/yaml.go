package configutils

import (
	"github.com/go-yaml/yaml"
	"io/ioutil"
)

func UnmarshalYamlFile(file string, ptr interface{}) error {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, ptr)
}
