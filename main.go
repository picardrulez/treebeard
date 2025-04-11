package main

import (
	"flag"
	"io"
	"log"
	"net/http"
	"os"
)

// global vars
var VERSION string = "v0.0.1"
var LOGFILE string = "/var/log/treebeard"

func main() {
	//set up logging
	var _, err = os.Stat(LOGFILE)
	if os.IsNotExist(err) {
		var file, err = os.Create(LOGFILE)
		checkError(err)
		defer file.Close()
	}
	f, err := os.OpenFile(LOGFILE, os.O_WRONLY|os.O_APPEND, 0644)
	checkError(err)
	defer f.Close()
	log.SetOutput(f)

	//read config
	var config = ReadConfig()

	//handle user flags
	bind := flag.String("bind", config.Bind, "port to bind to")
	flag.Parse()

	//handlers
	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/update", updateHandler)

	log.Println("Treebeard ", VERSION, " Started")
	http.ListenAndServe(":"+*bind, nil)
}

// http root handler
func rootHandler(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "Treebeard "+VERSION)
}

func checkError(err error) {
	if err != nil {
		log.Println(err.Error())
	}
}
