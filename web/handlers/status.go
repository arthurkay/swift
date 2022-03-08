package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"petricoh/operation"
	"petricoh/web/models"
)

func Status(w http.ResponseWriter, r *http.Request) {
	l, err := operation.Connect()
	if err != nil {
		fmt.Printf("%v", err)
	}
	data, er := ioutil.ReadAll(r.Body)
	if er != nil {
		fmt.Printf("Error: %v", er)
	}
	dataStruct := models.Domain{}
	e := json.Unmarshal(data, &dataStruct)
	if e != nil {
		fmt.Printf("%v", e)
	}
	operation.DomainState(dataStruct.Name, l)
	fmt.Fprintf(w, "Hello World")
}

func Reboot(w http.ResponseWriter, r *http.Request) {}

func ShutDown(w http.ResponseWriter, r *http.Request) {}

func StartUp(w http.ResponseWriter, r *http.Request) {}

func Install(w http.ResponseWriter, r *http.Request) {}
