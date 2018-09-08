package main

import (
	"fmt"
	"net/http"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

func publish(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	stream := strings.Split(r.FormValue("name"), "-")
	nom_user := stream[0]
	db_mu.Lock()
	query, err := db.Query("SELECT status FROM admin WHERE username = ?", nom_user)
	db_mu.Unlock()
	if err != nil {
		Warning.Println(err)
		http.Error(w, "Internal Server Error", 500)
		return
	}
	defer query.Close()
	for query.Next() {
		var status int
		err = query.Scan(&status)
		if err != nil {
			Warning.Println(err)
			continue
		}
		if r.FormValue("call") == "publish" && status == 1 {
			fmt.Fprintf(w, "Server OK")
			return
		} else {
			http.Error(w, "Internal Server Error", 500)
			return
		}
	}
	http.Error(w, "Internal Server Error", 500)
}

func onplay(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Internal Server Error", 500)
}
