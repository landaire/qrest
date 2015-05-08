package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func handleDb(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	genericJsonResponse(w, r, data)
}

func genericJsonResponse(w http.ResponseWriter, r *http.Request, data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	jsonResponse(jsonData, w)
}

func jsonResponse(data []byte, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func readRequestData(r *http.Request) (returnData map[string]interface{}, err error) {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}

	returnData = make(map[string]interface{})

	err = json.Unmarshal(data, &returnData)

	return
}
