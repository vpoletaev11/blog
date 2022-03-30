package main

import (
	"github.com/jmoiron/sqlx"

	_ "github.com/lib/pq"
)

func main() {
	db, err := sqlx.Open("postgres", "user=postgres password=password dbname=blog sslmode=disable")
	if err != nil {
		panic(err)
	}
	defer db.Close()

}
