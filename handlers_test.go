package main

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"io"
	"encoding/json"
	"bytes"
	"math/rand"
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
		err := testGetRequest("/"+recordType, expectedJson, http.StatusOK, true, true)
		if err != nil {
			t.Error(err)
		}
	}

	err := testGetRequest("/invalid", "", http.StatusNotFound, true, false)
	if err != nil {
		t.Error(err)
	}
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
		err := testGetRequest(path, expectedJson, http.StatusOK, true, true)
		if err != nil {
			t.Error(err)
		}
	}


	invalidPaths := []string{
		"/posts/-1",
		"/posts/9000",
		"/comments/-1",
		"/comments/9000",
	}

	for _, path := range invalidPaths {
		err := testGetRequest(path, "", http.StatusNotFound, true, false)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestPostRecord(t *testing.T) {
	databaseBeforeModification := serverData.Copy()
	defer func() {
		serverData = databaseBeforeModification
	}()

	arbitraryStringWithRandomNumber := func(text string) string {
		return fmt.Sprintf("%s %d", rand.Int())
	}

	titleString := func() string {
		return arbitraryStringWithRandomNumber("Title")
	}

	authorString := func() string {
		return arbitraryStringWithRandomNumber("Author")
	}

	bodyString := func() string {
		return arbitraryStringWithRandomNumber("Body")
	}

	// welcome to map[string]interface{} hell
	types := map[string][]map[string]interface{} {
		"posts": []map[string]interface{} {
			map[string]interface{} {
				"id": 2, // ID should be ignored
				"title": titleString(),
				"author": authorString(),
			},
			map[string]interface{} {
				"id": -1, // ID should be ignored
				"title": titleString(),
				"author": authorString(),
			},
			map[string]interface{} {
				"title": titleString(),
				"author": authorString(),
			},
			map[string]interface{} {
				"title": titleString(),
				"author": authorString(),
			},
		},
		"comments": []map[string]interface{} {
			map[string]interface{} {
				"id": 2, // ID should be ignored
				"body": bodyString(),
				"postId": rand.Int(),
			},
			map[string]interface{} {
				"id": -1,
				"body": bodyString(),
				"postId": rand.Int(),
			},
			map[string]interface{} {
				"body": bodyString(),
				"postId": rand.Int(),
			},
			map[string]interface{} {
				"body": bodyString(),
				"postId": rand.Int(),
			},
		},
	}

	for recordType, tests := range types {
		for _, test := range tests {
			testAsJson, err := json.Marshal(test)
			if err != nil {
				t.Error(err)
				continue
			}

			err = makePostRequest("/" + recordType, bytes.NewBuffer(testAsJson), http.StatusCreated)
			if err != nil {
				t.Error(err)
				return
			}

			test["id"] = maxIds[recordType]

			testAsJson, err = json.Marshal(test)
			if err != nil {
				t.Error(err)
				continue
			}

			err = testGetRequest(fmt.Sprintf("/%s/%d", recordType, maxIds[recordType]), string(testAsJson), http.StatusOK, false, true)
			if err != nil {
				t.Error(err)
			}
		}
	}
}

func TestPutRecord(t *testing.T) {

}

func TestDeleteRecord(t *testing.T) {

}

func TestGetAll(t *testing.T) {

}

func testGetRequest(path string, expectedJson string, expectedStatus int, useArray bool, compareBody bool) error {
	resp, err := http.Get("http://" + TestServerAddr + path)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != expectedStatus {
		return fmt.Errorf("Unexpected status code. Expected %d, got %d\n", expectedStatus, resp.StatusCode)
	}

	// Seems a little weird to have this as a parameter, but it prevents duplication of the above code
	if !compareBody {
		return nil
	}

	match, err := jsonResponseMatchesActual(resp, expectedJson, useArray)

	if !match {
		// TODO: make this print out the data mismatch? should probably just print the raw maps
		return fmt.Errorf("Data mismatch for path %s\n", path)
	}

	if err != nil {
		return err
	}

	return nil
}

func makePostRequest(path string, body io.Reader, expectedStatus int) error {
	// TODO: make the body type a parameter to support testing that body type is application/json? currently no
	// such check exists
	resp, err := http.Post("http://" + TestServerAddr + path, "application/json", body)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != expectedStatus {
		return fmt.Errorf("Unexpected status code. Expected %d, got %d\n", expectedStatus, resp.StatusCode)
	}

	return nil
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
