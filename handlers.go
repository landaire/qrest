package main
import (
	"net/http"
	"github.com/julienschmidt/httprouter"
	"encoding/json"
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
	w.Write(data)
	w.Header().Set("Content-Type", "application/json")
}
