package main

import (
	"net/http"
)

func main() {
	http.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		res.Write([]byte("hello\n"))
	})

	http.ListenAndServe(":8080", nil)
}
