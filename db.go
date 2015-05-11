package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"os/signal"
	"sync"
	"time"
)

var (
	serverData    = make(BackingData)
	dataMutex     sync.RWMutex
	dirty         = false
	maxIds		  = make(map[string]int64)
	ErrorNotFound = errors.New("Item not present in data set")
)

type BackingData map[string]interface{}

func (b BackingData) recordIndex(itemType string, id int64) (int, error) {
	rows, err := b.ItemType(itemType)

	if err != nil {
		return -1, err
	}

	for i, row := range rows {
		rowMap, _ := row.(map[string]interface{})

		currentId, ok := rowMap["id"].(json.Number)
		currentIdAsInt, err := currentId.Int64()

		if !ok {
			logger.Errorf("ID either not present for record %i or it's unknown type\n", i)
			continue
		}

		if err != nil {
			logger.Errorln(err)
			continue
		}

		// Found the item
		if currentIdAsInt == id {
			return i, nil
		}
	}

	return -1, ErrorNotFound
}

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

// Parses the JSON file provided in the command arguments
func parseJsonFile() {
	if len(os.Args) != 2 {
		logger.Println(os.Args)
		logger.Fatalln("Invalid number of arguments")
	}

	filename := os.Args[1]
	file, err := os.Open(filename)
	if err != nil {
		logger.Fatalln(err)
	}

	defer file.Close()

	decoder := json.NewDecoder(file)
	// decode as json.Number type instead of float64
	decoder.UseNumber()

	err = decoder.Decode(&serverData)
	if err != nil {
		logger.Fatalln(err)
	}

	// Get the highest IDs
	for _, itemType := range serverData.ItemTypes() {
		rows, _ := serverData.ItemType(itemType)
		for _, record := range rows {
			record := record.(map[string]interface{})
			id, ok := record["id"].(json.Number)

			if !ok {
				continue
			}

			idAsInt, err := id.Int64()

			if err != nil {
				continue
			}

			if max, ok := maxIds[itemType]; idAsInt > max || !ok {
				maxIds[itemType] = idAsInt + 1
			}
		}
	}
}

// Flushes the in-memory data to the JSON file
func flushJson() {
	filename := os.Args[1]
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	write := func() {
		if dirty {
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
	}

	for {
		select {
		case sig := <-c:
			write()
			if sig == os.Interrupt {
				os.Exit(0)
			}
		// Flush every 30 seconds
		case <-time.After(30 * time.Second):
			write()
		}
	}
}
