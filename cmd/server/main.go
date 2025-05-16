package main

import (
	"log"
	"sales-analytics/internal/di"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	srv, err := di.Init()
	if err != nil {
		log.Fatal(err)
	}
	log.Fatal(srv.ListenAndServe())
}
