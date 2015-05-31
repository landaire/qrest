package main

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"time"
)

var (
	serverData    = make(BackingData)
	dataMutex     sync.RWMutex
	dirty         = false
	maxIds        = make(map[string]int64)
	ErrorNotFound = errors.New("Item not present in data set")
	JsonFilePath  string
)

func init() {
	if len(os.Args) > 1 {
		JsonFilePath = os.Args[1]
	}
}

type BackingData map[string]interface{}

// recordIndex returns the index of a record within the `BackingData[itemType]` array
//
func (b BackingData) recordIndex(itemType string, id int64) (int, error) {
	rows, err := b.ItemType(itemType)

	if err != nil {
		return -1, err
	}

	for i, row := range rows {
		rowMap, _ := row.(map[string]interface{})

		currentId, ok := rowMap["id"].(int64)

		if !ok {
			logger.Errorf("ID either not present for record at index %i or it's unknown type\n", i)
			continue
		}

		if err != nil {
			logger.Errorln(err)
			continue
		}

		// Found the item
		if currentId == id {
			return i, nil
		}
	}

	return -1, ErrorNotFound
}

// RecordWithId will return a record with the provided ID. If no such record exists, err
// will be set to ErrorNotFound
//
func (b BackingData) RecordWithId(itemType string, id int64) (map[string]interface{}, error) {
	rows, err := b.ItemType(itemType)

	if err != nil {
		return nil, err
	}

	index, err := b.recordIndex(itemType, id)
	if err != nil {
		return nil, err
	}

	rowMap := rows[index].(map[string]interface{})

	return rowMap, nil
}

func (b BackingData) DeleteRecord(itemType string, id int64) error {
	records, err := b.ItemType(itemType)
	if err != nil {
		return err
	}

	index, err := b.recordIndex(itemType, id)
	if err != nil {
		return err
	}

	b[itemType] = append(records[:index], records[index+1:]...)

	return nil
}

func (b BackingData) ItemType(itemType string) ([]interface{}, error) {
	value, ok := b[itemType]

	if !ok {
		return nil, ErrorNotFound
	}

	return value.([]interface{}), nil
}

func (b BackingData) ItemTypes() []string {
	itemTypes := make([]string, 0, len(serverData))

	for key, _ := range b {
		itemTypes = append(itemTypes, key)
	}

	return itemTypes
}

func (b BackingData) AddRecord(itemType string, record map[string]interface{}) {
	items, _ := b.ItemType(itemType)
	b[itemType] = append(items, record)
}

func (b BackingData) Copy() BackingData {
	data := make(BackingData)

	for key, value := range b {
		data[key] = copyInterfaceType(value)
	}

	return data
}

func copyInterfaceType(value interface{}) interface{} {
	switch value.(type) {
	case map[string]interface{}:
		mapValue := value.(map[string]interface{})

		mapValueCopy := make(map[string]interface{})

		for key, value := range mapValue {
			mapValueCopy[key] = copyInterfaceType(value)
		}

		return mapValueCopy
	case []interface{}:
		valueArray := value.([]interface{})
		valueArrayCopy := make([]interface{}, len(valueArray))
		copy(valueArrayCopy, valueArray)
		for i, value := range valueArrayCopy {
			valueArray[i] = copyInterfaceType(value)
		}

		return valueArrayCopy
	case json.Number:
		jsonNumber := value.(json.Number)
		number, _ := jsonNumber.Int64()

		return json.Number(strconv.FormatInt(number, 10))
	default:
		return value
	}
}

// Parses the JSON file provided in the command arguments
//
func parseJsonFile(fname string) {
	file, err := os.Open(fname)
	if err != nil {
		logger.Fatalln(err)
	}

	defer file.Close()

	err = decodeJson(file, &serverData)
	if err != nil {
		logger.Fatalln(err)
	}

	// Get the highest IDs
	for _, itemType := range serverData.ItemTypes() {
		rows, _ := serverData.ItemType(itemType)
		for _, record := range rows {
			record := record.(map[string]interface{})
			id, ok := record["id"].(int64)

			if !ok {
				continue
			}

			if err != nil {
				continue
			}

			if max, ok := maxIds[itemType]; id > max || !ok {
				maxIds[itemType] = id
			}
		}
	}
}

// Flushes the in-memory data to the JSON file
func flushJson(filename string) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	write := func() {
		if !dirty {
			return
		}

		dataMutex.RLock()
		dirty = false

		jsonData, err := json.Marshal(serverData)
		dataMutex.RUnlock()
		if err != nil {
			logger.Error(err)
			return
		}

		ioutil.WriteFile(filename, jsonData, 0755)
	}

	// Flush loop
	for {
		select {
		case <-c:
			write()
			return
		// Flush every 30 seconds
		case <-time.After(30 * time.Second):
			write()
		}
	}
}

// Makes decoding JSON less repetitive (no need to create the decoder, call UseNumber(), etc.)
//
func decodeJson(r io.Reader, data interface{}) error {
	decoder := json.NewDecoder(r)
	decoder.UseNumber()

	err := decoder.Decode(data)

	if err != nil {
		return err
	}

	// Convert all json.Number to int64
	if dataMap, ok := data.(*map[string]interface{}); ok {
		convertMapNumbers(*dataMap)
	}

	if backingData, ok := data.(*BackingData); ok {
		backingDataAsMap := (*map[string]interface{})(backingData)
		convertMapNumbers(*backingDataAsMap)
	}

	return nil
}

func convertMapNumbers(data map[string]interface{}) {
	for key, value := range data {
		data[key] = convertMapType(value)
	}
}

func convertMapType(value interface{}) interface{} {
	switch value.(type) {
	case map[string]interface{}:
		dataMap := value.(map[string]interface{})
		convertMapNumbers(dataMap)

		return dataMap
	case []interface{}:
		valueArray := value.([]interface{})
		for i, value := range valueArray {
			newValue := convertMapType(value)
			valueArray[i] = newValue
		}

		return valueArray
	case json.Number:
		number := value.(json.Number)
		numberAsInt, _ := number.Int64()

		return numberAsInt
	default:
		return value
	}

}
