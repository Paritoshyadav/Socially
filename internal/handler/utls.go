package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-playground/validator/v10"
)

func response(w http.ResponseWriter, v interface{}, statusCode int) {

	b, err := json.Marshal(v)
	os.Stdout.Write(b)
	if err != nil {
		responseError(w, fmt.Errorf("could not marshal response: %v", err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(b)
}

func responseError(res http.ResponseWriter, err error) {
	log.Println(err)
	http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func ValidateInput(s interface{}) error {

	val := validator.New()

	return val.Struct(s)

}
