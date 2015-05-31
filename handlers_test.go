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
		serverData = databaseBeforeModification.Copy()
	}()

	arbitraryStringWithRandomNumber := func(text string) string {
		return fmt.Sprintf("%s %d", text, rand.Int())
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
				"postId": rand.Int63(),
			},
			map[string]interface{} {
				"id": -1,
				"body": bodyString(),
				"postId": rand.Int63(),
			},
			map[string]interface{} {
				"body": bodyString(),
				"postId": rand.Int63(),
			},
			map[string]interface{} {
				"body": bodyString(),
				"postId": rand.Int63(),
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

			err = makeRequest("POST", "/" + recordType, bytes.NewBuffer(testAsJson), []int{http.StatusCreated})
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
	databaseBeforeModification := serverData.Copy()
	defer func() {
		serverData = databaseBeforeModification
	}()


	arbitraryStringWithRandomNumber := func(text string) string {
		return fmt.Sprintf("%s %d", text, rand.Int())
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
				"id": 2,
				"title": titleString(),
				"author": authorString(),
			},
			map[string]interface{} {
				"id": 1000,
				"title": titleString(),
				"author": authorString(),
			},
		},
		"comments": []map[string]interface{} {
			map[string]interface{} {
				"id": 2, // ID should be ignored
				"body": bodyString(),
				"postId": rand.Int63(),
			},
			map[string]interface{} {
				"id": 1000,
				"body": bodyString(),
				"postId": rand.Int63(),
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

			acceptableStatuses := []int{http.StatusCreated, http.StatusOK}
			err = makeRequest("PUT", fmt.Sprintf("/%s/%d", recordType, test["id"]), bytes.NewBuffer(testAsJson), acceptableStatuses)
			if err != nil {
				t.Error(err)
				return
			}

			testAsJson, err = json.Marshal(test)
			if err != nil {
				t.Error(err)
				continue
			}

			err = testGetRequest(fmt.Sprintf("/%s/%d", recordType, int64(test["id"].(int))), string(testAsJson), http.StatusOK, false, true)
			if err != nil {
				t.Error(err)
			}
		}
	}
}

func TestDeleteRecord(t *testing.T) {
	databaseBeforeModification := serverData.Copy()
	defer func() {
		serverData = databaseBeforeModification
	}()

	err := makeRequest("DELETE", "/posts/1", strings.NewReader(""), []int{http.StatusOK})

	if err != nil {
		t.Error(err)
		return
	}

	err = testGetRequest("/posts/1", "", http.StatusNotFound, false, false)
	if err != nil {
		t.Error(err)
		return
	}
}

func TestGetAll(t *testing.T) {
	err := testGetRequest("/db", jsonTestData, http.StatusOK, false, true)
	if err != nil {
		t.Error(err)
		return
	}
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

	match, err, expected, actual := jsonResponseMatchesActual(resp, expectedJson, useArray)

	if err != nil {
		return err
	}

	if !match {
		// TODO: make this print out the data mismatch? should probably just print the raw maps
		return fmt.Errorf("Data mismatch for path %s.\n Expected:\n%#v\n\ngot\n%#v", path, expected, actual)
	}


	return nil
}

// Makes a request at `path` with the given method and body. `acceptableStatuses` are any status that is acceptable
//
func makeRequest(method string, path string, body io.Reader, acceptableStatuses []int) error {
	// TODO: make the body type a parameter to support testing that body type is application/json? currently no
	// such check exists
	req, err := http.NewRequest(method, "http://" + TestServerAddr + path, body)

	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	statusPresent := false

	for _, status := range acceptableStatuses {
		if resp.StatusCode == status {
			statusPresent = true
			break
		}
	}

	if !statusPresent {
		return fmt.Errorf("Unexpected status code. Expected any of %v, got %d\n", acceptableStatuses, resp.StatusCode)
	}

	return nil
}

func jsonResponseMatchesActual(resp *http.Response, expected string, useArray bool) (bool, error, interface{}, interface{}) {
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
		return false, fmt.Errorf("Unexpected Content-Type %s\n", contentType), nil, nil
	}

	if useArray {
		actualData = make(map[string]interface{})
	} else {
		actualData = []interface{}{}
	}

	err := decodeJson(strings.NewReader(expected), &expectedData)

	if err != nil {
		return false, err, nil, nil
	}

	err = decodeJson(resp.Body, &actualData)

	if err != nil {
		return false, err, nil, nil
	}

	return reflect.DeepEqual(expectedData, actualData), nil, expectedData, actualData
}
