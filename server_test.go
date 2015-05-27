package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

var (
	TestServerAddr = "localhost:"
)

// Setup function. We expect parseJsonFile to succeed before running all subsequent tests
//
func TestMain(m *testing.M) {
	// Setup for the JSON file
	tempFile, err := ioutil.TempFile("", "qrest")
	n, err := tempFile.Write([]byte(jsonTestData))
	tempFile.Close()

	JsonFilePath = tempFile.Name()

	if n != len([]byte(jsonTestData)) {
		fmt.Fprintln(os.Stderr, "invalid number of bytes written to temp file")
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err == nil {
		parseJsonFile(tempFile.Name())

		// Attempt to start the server on a (hopefully) unused port
		if port := os.Getenv("TEST_PORT"); port != "" {
			TestServerAddr += port
		} else {
			TestServerAddr += "3050"
		}

		logger.Out = ioutil.Discard
		go StartServer(TestServerAddr)
	} else {
		fmt.Fprintln(os.Stderr, "could not create temp file")
	}

	os.Exit(m.Run())
}
