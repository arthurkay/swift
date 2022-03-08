package main

import (
	"fmt"
	"net/http"
	"petricoh/web"
)

func main() {
	routes := web.Routes()
	fmt.Printf("Application up and running")
	err := http.ListenAndServe(fmt.Sprintf(":%s", "8000"), routes)
	if err != nil {
		panic(err)
	}
}
