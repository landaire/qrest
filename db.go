package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"os/signal"
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
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	write := func() {
		if dirty {
			dataMutex.RLock()
			dirty = false

			jsonData, err := json.Marshal(data)
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
