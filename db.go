package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"time"
)

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

	jsonData, err := ioutil.ReadAll(file)
	if err != nil {
		logger.Fatalln(err)
	}

	err = json.Unmarshal(jsonData, &data)
	if err != nil {
		logger.Fatalln(err)
	}
}

// Flushes the in-memory data to the JSON file
func flushJson() {
	filename := os.Args[1]

	for {
		// Flush every 30 seconds
		<-time.After(30 * time.Second)

		if dirty {
			dataMutex.RLock()
			dirty = false

			jsonData, err := json.Marshal(data)
			dataMutex.RUnlock()
			if err != nil {
				logger.Error(err)
				continue
			}

			ioutil.WriteFile(filename, jsonData, 0755)
		}
	}
}
