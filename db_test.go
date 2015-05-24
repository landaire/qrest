package main
import (
	"testing"
	"io/ioutil"
	"os"
	"fmt"
	"reflect"
	"encoding/json"
)

const jsonTestData = `
{
  "posts": [
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
  ],
  "comments": [
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
  ]
}
`

func TestMain(t *testing.M) {
	tempFile, err := ioutil.TempFile("", "qrest")
	n, err := tempFile.Write([]byte(jsonTestData))
	tempFile.Close()

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
	} else {
		fmt.Fprintln(os.Stderr, "could not create temp file")
	}

	os.Exit(t.Run())
}

func TestParseJsonFile(t *testing.T) {
	// `TestMain` will have already called parseJsonFile for the initial setup,
	// so this is just a quick check to make sure that actually succeeded
	if maxIds["posts"] != 2 || maxIds["comments"] != 2 {
		t.Fatal("Failing TestParseJsonFile fails all tests")
	}
}

func TestItemTypes(t *testing.T) {
	itemTypes := serverData.ItemTypes()
	itemTypesFound := make(map[string]bool)
	expectedTypes := []string{"posts", "comments"}

	if len(itemTypes) != len(expectedTypes) {
		t.Errorf("Expected %d item types, got %d\n", len(expectedTypes), len(itemTypes))
	}

	for _, itemType := range expectedTypes {
		itemTypesFound[itemType] = false
	}

	for _, itemType := range itemTypes {
		itemTypesFound[itemType] = true
	}

	for itemType, found := range itemTypesFound {
		if !found {
			t.Errorf("Item type %s not found\n", itemType)
		}
	}
}

func TestFetchingRecord(t *testing.T) {
	type Record struct {
		Type string
		Id int64
		Expected map[string]interface{}
	}

	recordsToFetch := []Record{
		Record {
			Type: "posts",
			Id: 1,
			Expected: map[string]interface{}{
				"id": int64(1),
				"title": "Testing",
				"author": "Foo",
			},
		},
		Record {
			Type: "comments",
			Id: 2,
			Expected: map[string]interface{}{
				"id": int64(2),
				"body": "Testing Comment ID 2",
				"postId": int64(2),
			},
		},
	}

	for _, expectedRecord := range recordsToFetch {
		actualRecord, err := serverData.RecordWithId(expectedRecord.Type, expectedRecord.Id)

		if err != nil {
			t.Error("No error was expected:", err)
			continue
		}

		if len(actualRecord) != len(expectedRecord.Expected) {
			t.Errorf("Invalid number of columns in returned record. Expected %d, got %d", len(expectedRecord.Expected), len(actualRecord))
		}

		for key, expectedValue := range expectedRecord.Expected {
			actualValue, ok := actualRecord[key]

			if !ok {
				t.Error("invalid key", key)
				continue
			}



			if expectedValue != actualValue {
				expectedType := reflect.TypeOf(expectedValue)
				actualType := reflect.TypeOf(actualValue)

				// Special handling for json.Number
				if jsonNumber, ok := actualValue.(json.Number); ok {
					int64Number, err := jsonNumber.Int64()
					if err == nil && int64Number == expectedValue {
						continue
					}
				}
				t.Errorf("expected %#v of type %s, got %#v of type %s", expectedValue, expectedType, actualValue, actualType)
				continue
			}
		}
	}

	// Test getting a bad record
	errorMessage := "Should have gotten an error, none given"
	if _, err := serverData.RecordWithId("posts", -1); err == nil {
		t.Error(errorMessage)
	}

	if _, err := serverData.RecordWithId("non-existant", -1); err == nil {
		t.Error(errorMessage)
	}
}
