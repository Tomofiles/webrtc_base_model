package main

import (
	"encoding/json"
	"html/template"
	"net/http"
)

func web(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		t, _ := template.ParseFiles("sfu.html")
		checkError(t.Execute(w, nil))
	}
}

type serverinfo struct {
	URL        string `json:"url"`
	Credential string `json:"credential"`
	Username   string `json:"username"`
}

func getServerInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		payload := serverinfo{
			URL:        *serverURL,
			Credential: serverCredential,
			Username:   serverUname,
		}
		response, _ := json.Marshal(payload)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write(response)
	}
}
