package main

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestGetAllRecordsOfType(t *testing.T) {
	types := map[string]string{
		"posts": `[
					{
					  "id": 1,
					  "title": "Testing",
					  "author": "Foo"
					},
					{
					  "id": 2,
					  "title": "Testing Post ID 2",
					  "author": "Bar"
					}
				  ]`,
		"comments": `[
						{
						  "id": 1,
						  "body": "Testing",
						  "postId": 1
						},
						{
						  "id": 2,
						  "body": "Testing Comment ID 2",
						  "postId": 2
						}
					  ]`,
	}

	for recordType, expectedJson := range types {
		testGetRequest(t, "/"+recordType, expectedJson, http.StatusOK, true, false)
	}

	testGetRequest(t, "/invalid", "", http.StatusNotFound, true, false)
}

func TestGetRecord(t *testing.T) {
	paths := map[string]string{
		"/posts/2": `{
					  "id": 2,
					  "title": "Testing Post ID 2",
					  "author": "Bar"
					}`,
		"/comments/1": `{
						  "id": 1,
						  "body": "Testing",
						  "postId": 1
						}`,
	}

	for path, expectedJson := range paths {
		testGetRequest(t, path, expectedJson, http.StatusOK, true, false)
	}


	invalidPaths := []string{
		"/posts/-1",
		"/posts/9000",
		"/comments/-1",
		"/comments/9000",
	}

	for _, path := range invalidPaths {
		testGetRequest(t, path, "", http.StatusNotFound, true, false)
	}
}

func TestPostRecord(t *testing.T) {

}

func TestPutRecord(t *testing.T) {

}

func TestDeleteRecord(t *testing.T) {

}

func TestGetAll(t *testing.T) {

}

func testGetRequest(t *testing.T, path string, expectedJson string, expectedStatus int, useArray bool, compareBody bool) {
	resp, err := http.Get("http://" + TestServerAddr + path)

	if err != nil {
		t.Error(err)
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != expectedStatus {
		t.Errorf("Unexpected status code. Expected %d, got %d\n", expectedStatus, resp.StatusCode)
	}

	// Seems a little weird to have this as a parameter, but it prevents duplication of the above code
	if !compareBody {
		return
	}

	match, err := jsonResponseMatchesActual(resp, expectedJson, useArray)

	if !match {
		// TODO: make this print out the data mismatch? should probably just print the raw maps
		t.Error("Data mismatch for path", path)
	}

	if err != nil {
		t.Error(err)
	}
}

func jsonResponseMatchesActual(resp *http.Response, expected string, useArray bool) (bool, error) {
	var (
		expectedData interface{}
		actualData   interface{}
	)

	if useArray {
		expectedData = make(map[string]interface{})
	} else {
		expectedData = []interface{}{}
	}

	if contentType := resp.Header.Get("Content-Type"); contentType != "application/json" {
		return false, fmt.Errorf("Unexpected Content-Type %s\n", contentType)
	}

	if useArray {
		actualData = make(map[string]interface{})
	} else {
		actualData = []interface{}{}
	}

	err := decodeJson(strings.NewReader(expected), &expectedData)

	if err != nil {
		return false, err
	}

	err = decodeJson(resp.Body, &actualData)

	if err != nil {
		return false, err
	}

	return reflect.DeepEqual(expectedData, actualData), nil
}
