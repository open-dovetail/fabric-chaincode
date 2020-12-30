package contract

import (
	"bytes"
	"encoding/json"
	"io/ioutil"

	"github.com/project-flogo/core/app"
	"github.com/project-flogo/core/app/resource"
	"github.com/project-flogo/flow/definition"
)

// AppConfig contains config of a Flogo app and its unmarshalled resource configs
type AppConfig struct {
	Config    *app.Config
	Resources map[string]*definition.DefinitionRep
}

// ReadAppConfig reads a Flogo app json file and returns app.Config
func ReadAppConfig(configFile string) (*AppConfig, error) {
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	config := &app.Config{}
	err = json.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}

	result := &AppConfig{Config: config}
	result.Resources = make(map[string]*definition.DefinitionRep)
	for _, r := range config.Resources {
		var def = &definition.DefinitionRep{}
		err := json.Unmarshal(r.Data, def)
		if err != nil {
			return nil, err
		}
		result.Resources[r.ID] = def
	}
	return result, nil
}

// marshal JSON w/o escaping condition chars &, <, >
func marshalNoEscape(data interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "   ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// WriteAppConfig serializes an app config and its resources
func WriteAppConfig(config *AppConfig, outFile string) error {
	// serializes resources
	config.Config.Resources = make([]*resource.Config, 0)
	for k, v := range config.Resources {
		data, err := marshalNoEscape(v)
		if err != nil {
			return err
		}
		res := &resource.Config{
			ID:   k,
			Data: data,
		}
		config.Config.Resources = append(config.Config.Resources, res)
	}

	result, err := marshalNoEscape(config.Config)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(outFile, result, 0644)
}
