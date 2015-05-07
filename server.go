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

		router.Handle("GET", fmt.Sprintf("/%s", key), func (w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
			value := value
			genericJsonResponse(w, r, value)
		})

		router.Handle("GET", fmt.Sprintf("/%s/:id", key), func (w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
			rows, ok := value.([]interface{})
			if !ok {
				logger.Error("unknown type")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			idParam, _ := strconv.ParseFloat(ps.ByName("id"), 64)
			for _, row := range rows {
				rowMap, _ := row.(map[string]interface{})
				if id, ok := rowMap["id"]; ok && id.(float64) == idParam {
					genericJsonResponse(w, r, row)
				}
			}

			w.WriteHeader(http.StatusNotFound)
		})
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
			defer dataMutex.RUnlock()
			dirty = false

			jsonData, err := json.Marshal(data)
			if err != nil {
				logger.Error(err)
				continue
			}

			ioutil.WriteFile(filename, jsonData, 0755)
		}
	}
}
