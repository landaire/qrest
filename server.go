package main

import (
	"os"

	"fmt"
	"net/http"
	"strconv"
	"sync"
	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/negroni"
	"github.com/julienschmidt/httprouter"
	nlogrus "github.com/meatballhat/negroni-logrus"
)

var (
	data      = make(map[string]interface{})
	dataMutex sync.RWMutex
	dirty     = false
	logger    *logrus.Logger
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

	setupRoutes(&router)

	go flushJson()

	n := negroni.Classic()
	n.Use(logr)
	n.UseHandler(router)
	n.Run(port)
}
