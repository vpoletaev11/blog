package main

import (
	"blog/handlers"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/gorilla/mux"
	_ "github.com/jackc/pgx/stdlib"
)

const (
	dbConnCheckTimeout = 1 * time.Second
	dbConnChecksCount  = 20
)

func main() {
	db, err := sqlx.Open("pgx", "postgres://postgres:password@172.17.0.1:8081/blog")
	if err != nil {
		log.Fatalf("error while open connection to database: %s", err.Error())
	}
	defer db.Close()

	for i := 1; db.Ping() != nil; i++ {
		log.Printf("database acccesabilty check failure")
		time.Sleep(dbConnCheckTimeout)
		if i == dbConnChecksCount {
			log.Fatal("database connection check timeout")
		}
	}

	router := mux.NewRouter()
	router.HandleFunc("/posts", handlers.AddPostJSON(db)).Methods("POST")
	router.HandleFunc("/posts", handlers.ListPostsJSON(db)).Methods("GET")

	fmt.Println("starting server on 8080 port")
	err = http.ListenAndServe(":8080", router)
	if err != nil {
		panic(err)
	}
}
