package main

import (
	"blog/handlers"
	"fmt"
	"net/http"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/gorilla/mux"
	_ "github.com/jackc/pgx/stdlib"
)

func main() {
	db, err := sqlx.Open("pgx", "postgres://postgres:password@172.17.0.1:8081/blog")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	time.Sleep(30 * time.Second)
	err = db.Ping()
	if err != nil {
		panic(err)
	}

	router := mux.NewRouter()
	router.HandleFunc("/posts", handlers.AddPostJSON(db)).Methods("POST")
	router.HandleFunc("/posts", handlers.ListPostsJSON(db)).Methods("GET")

	http.Handle("/", router)

	fmt.Println("starting server on 8080 port")
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}
