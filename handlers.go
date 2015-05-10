package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"fmt"
	"strconv"
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
	for key, value := range data {
		// Shadow these variables. If this isn't done, then the closures below will see
		// `value` and `key` as whatever they were in the last(?) iteration of the above for loop
		value := value
		key := key
		rows, ok := value.([]interface{})

		if !ok {
			logger.Fatalln("unknown type")
		}

		router.POST(fmt.Sprintf("/%s", key), func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
			data, err := readRequestData(r)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			dataMutex.Lock()

			dirty = true
			rows = append(rows, data)
			data[key] = rows

			dataMutex.Unlock()
		})

		router.GET(fmt.Sprintf("/%s", key), func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
			genericJsonResponse(w, r, value)
		})

		for _, method := range []string{"GET", "PATCH", "PUT", "DELETE"} {
			method := method
			router.Handle(method, fmt.Sprintf("/%s/:id", key), func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
				idParam, _ := strconv.ParseFloat(ps.ByName("id"), 64)
				for i, row := range rows {
					rowMap, _ := row.(map[string]interface{})

					// Found the item
					if id, ok := rowMap["id"]; ok && id.(float64) == idParam {

						// The method type determines how we respond
						switch method {
						case "GET":
							genericJsonResponse(w, r, row)
							return
						case "PATCH":
							updatedData, err := readRequestData(r)
							if err != nil {
								w.WriteHeader(http.StatusBadRequest)
								return
							}

							dataMutex.Lock()
							for key, value := range updatedData {
								rowMap[key] = value
							}

							dirty = true

							// since value is a shadow copy, we need to update it as it's now stale
							value = data[key]

							dataMutex.Unlock()

							return
						case "PUT":
							updatedData, err := readRequestData(r)
							if err != nil {
								w.WriteHeader(http.StatusBadRequest)
								return
							}

							dataMutex.Lock()
							for key, _ := range rowMap {
								rowMap[key] = nil
							}

							for key, value := range updatedData {
								rowMap[key] = value
							}

							dirty = true

							// since value is a shadow copy, we need to update it as it's now stale
							value = data[key]

							dataMutex.Unlock()

							w.WriteHeader(http.StatusOK)

							return
						case "DELETE":
							dataMutex.Lock()
							data[key] = append(rows[:i], rows[i+1:]...)
							dirty = true

							// since value is a shadow copy, we need to update it as it's now stale
							value = data[key]

							dataMutex.Unlock()

							w.WriteHeader(http.StatusOK)
							return
						}
					}
				}

				if method == "PUT" {
					newData, err := readRequestData(r)
					if err != nil {
						w.WriteHeader(http.StatusBadRequest)
						return
					}

					dataMutex.Lock()
					dirty = true
					rows = append(rows, newData)
					data[key] = rows
					dataMutex.Unlock()

					w.WriteHeader(http.StatusCreated)
				}
				w.WriteHeader(http.StatusNotFound)
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
		genericJsonResponse(w, r, data)
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
	var data []byte
	data, err = ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}

	returnData = make(map[string]interface{})

	err = json.Unmarshal(data, &returnData)

	return
}
