package main

import (
	"encoding/json"
	"net/http"

	"fmt"
	"strconv"

	"github.com/julienschmidt/httprouter"
)

// addDynamicRoutes will dynamically add RESTful routes to the router. Routes are added based off of the keys that
// are present in the parsed JSON file. For instance, if a JSON file is set up like:
//
//    {
//        "posts": [ { "id": 1, "title": "Foo" } ]
//    }
//
// The following routes will be created:
//
//    POST /posts (creates a new post record)
//    GET /posts (returns all post records)
//    GET /posts/:id (returns a specific record)
//    PUT /posts/:id (creates or updates a record with the specified ID)
//    PATCH /posts/:id (updates a record with the specified ID)
//    DELETE /posts/:id (deletes the specified record)
//
//
func addDynamicRoutes(router *httprouter.Router) {
	// set up our routes
	for _, itemType := range serverData.ItemTypes() {
		// Shadow these variables. If this isn't done, then the closures below will see
		// `value` and `key` as whatever they were in the last(?) iteration of the above for loop
		itemType := itemType

		router.POST(fmt.Sprintf("/%s", itemType), func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
			data, err := readRequestData(r)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			dataMutex.Lock()

			dirty = true
			serverData.AddRecord(itemType, data)

			dataMutex.Unlock()
		})

		router.GET(fmt.Sprintf("/%s", itemType), func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
			items, _ := serverData.ItemType(itemType)
			genericJsonResponse(w, r, items)
		})

		for _, method := range []string{"GET", "PATCH", "PUT", "DELETE"} {
			method := method
			router.Handle(method, fmt.Sprintf("/%s/:id", itemType), func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
				idParam, _ := strconv.ParseInt(ps.ByName("id"), 10, 64)

				record, err := serverData.RecordWithId(itemType, idParam)

				if err != nil {
					if err == ErrorNotFound {
						w.WriteHeader(http.StatusNotFound)
					} else {
						w.WriteHeader(http.StatusInternalServerError)
					}
				}

				// The method type determines how we respond
				switch method {
				case "GET":
					genericJsonResponse(w, r, record)
					return
				case "PATCH":
					updatedData, err := readRequestData(r)
					if err != nil {
						w.WriteHeader(http.StatusBadRequest)
						return
					}

					dataMutex.Lock()
					for key, value := range updatedData {
						record[key] = value
					}

					dirty = true

					dataMutex.Unlock()

					return
				case "PUT":
					updatedData, err := readRequestData(r)
					if err != nil {
						w.WriteHeader(http.StatusBadRequest)
						return
					}

					dataMutex.Lock()

					for key, _ := range record {
						record[key] = nil
					}

					for key, value := range updatedData {
						record[key] = value
					}

					dirty = true

					dataMutex.Unlock()

					w.WriteHeader(http.StatusOK)
					return
				case "DELETE":
					dataMutex.Lock()

					dirty = true
					serverData.DeleteRecord(itemType, idParam)

					dataMutex.Unlock()

					w.WriteHeader(http.StatusOK)
					return
				}

				// If it's not found, then this request acts as a POST
				if method == "PUT" {
					newData, err := readRequestData(r)
					if err != nil {
						w.WriteHeader(http.StatusBadRequest)
						return
					}

					if _, hasId := newData["id"]; !hasId {
						newData["id"] = idParam
					}

					dataMutex.Lock()
					dirty = true

					serverData.AddRecord(itemType, newData)

					dataMutex.Unlock()

					w.WriteHeader(http.StatusCreated)
					return
				}

			})
		}
	}
}

// addStaticRoutes adds all routes which are present regardless of the JSON file's data. These include
//
//    GET /db (returns the entire DB as a JSON structure)
//
//
func addStaticRoutes(router *httprouter.Router) {
	router.GET("/db", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		genericJsonResponse(w, r, serverData)
	})
}

// genericJsonResponse writes a generic JSON response and handles any errors which may occur
// when marshalling the data
//
func genericJsonResponse(w http.ResponseWriter, r *http.Request, data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

// readRequestData parses the JSON body of a request
//
func readRequestData(r *http.Request) (returnData map[string]interface{}, err error) {
	returnData = make(map[string]interface{})

	decoder := json.NewDecoder(r.Body)
	decoder.UseNumber()

	// don't handle err here since it's returned
	err = decoder.Decode(&returnData)

	return
}
