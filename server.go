package main

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/negroni"
	"github.com/julienschmidt/httprouter"
	nlogrus "github.com/meatballhat/negroni-logrus"
	"time"
	"sync"
	"fmt"
	"net/http"
	"strconv"
)

var (
	data   = make(map[string]interface{})
	dataMutex sync.RWMutex
	dirty  = false
	logger *logrus.Logger
)

func main() {
	logr := nlogrus.NewMiddleware()
	logger = logr.Logger

	parseJsonFile()

	port := ":" + os.Getenv("PORT")
	if port == ":" {
		port = ":3000"
	}

	router := httprouter.New()

	router.Handle("GET", "/db", handleDb)

	// set up our routes
	for key, value := range data {
		// Shadow these variables. If this isn't done, then the closures below will see
		// `value` and `key` as whatever they were in the last(?) iteration of the above for loop
		value := value
		key := key

		router.GET(fmt.Sprintf("/%s", key), func (w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
			genericJsonResponse(w, r, value)
		})

		for _, method := range []string{"GET", "PATCH", "PUT", "DELETE"} {
			method := method
			router.Handle(method, fmt.Sprintf("/%s/:id", key), func (w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
				rows, ok := value.([]interface{})
				if !ok {
					logger.Error("unknown type")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				idParam, _ := strconv.ParseFloat(ps.ByName("id"), 64)
				for i, row := range rows {
					rowMap, _ := row.(map[string]interface{})

					// Found the item
					if id, ok := rowMap["id"]; ok && id.(float64) == idParam {

						// The method type determines how we respond
						switch (method) {
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
					data[key] = append(rows, newData)
					dataMutex.Unlock()

					w.WriteHeader(http.StatusCreated)
				}
				w.WriteHeader(http.StatusNotFound)
			})
		}
	}

	go flushJson()

	n := negroni.Classic()
	n.Use(logr)
	n.UseHandler(router)
	n.Run(port)
}

// Parses the JSON file provided in the command arguments
func parseJsonFile() {
	if len(os.Args) != 2 {
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
