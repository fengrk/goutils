package common

import (
	"encoding/json"
	"github.com/bitly/go-simplejson"
	"github.com/frkhit/logger"
	"io/ioutil"
)

// ref: https://www.oschina.net/code/snippet_197499_22659
func JsonLoads(filename string) (content map[string]interface{}, err error) {
	fileBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	
	if err := json.Unmarshal(fileBytes, &content); err != nil {
		logger.Errorln("Unmarshal: ", err.Error())
		return nil, err
	}
	
	return content, nil
}

func JsonDumps(filename string, content map[string]interface{}, indent string) (error) {
	contentJson, err := json.MarshalIndent(content, "", indent)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, contentJson, 0644)
}

func JsonSimpleLoads(filename string) (content *simplejson.Json, err error) {
	fileBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return simplejson.NewJson(fileBytes)
}

func JsonSimpleDumps(filename string, content *simplejson.Json, indent string) (error) {
	contentJson, err := json.MarshalIndent(content, "", indent)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, contentJson, 0644)
}

func NewJsonFromMap(info map[string]interface{}) (result *simplejson.Json, err error) {
	infoBytes, err := json.Marshal(info)
	
	if err != nil {
		return nil, err
	}
	return simplejson.NewJson(infoBytes)
}

func NewJsonFromMapArray(info []map[string]interface{}) (result *simplejson.Json, err error) {
	infoBytes, err := json.Marshal(info)
	
	if err != nil {
		return nil, err
	}
	return simplejson.NewJson(infoBytes)
}

func NewMapFromJson(info *simplejson.Json) (result map[string]interface{}, err error) {
	infoBytes, err := json.Marshal(info)
	if err != nil {
		return nil, err
	}
	
	if err := json.Unmarshal(infoBytes, &result); err != nil {
		logger.Errorln("Unmarshal: ", err.Error())
		return nil, err
	}
	
	return result, nil
}

func NewMapFromJsonArray(info *simplejson.Json) (result []interface{}, err error) {
	infoBytes, err := json.Marshal(info)
	if err != nil {
		return nil, err
	}
	
	if err := json.Unmarshal(infoBytes, &result); err != nil {
		logger.Errorln("Unmarshal: ", err.Error())
		return nil, err
	}
	
	return result, nil
}
