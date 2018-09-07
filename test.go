package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

//var db_mu sync.Mutex
var c = make(chan int, 1)

func main() {
	os.Remove("/tmp/testlock.db")
	os.Remove("/tmp/testlock-shm.db")
	os.Remove("/tmp/testlock-wal.db")

	var db *sql.DB
	if len(os.Args) != 2 { //no param
		printUsage()
		return
	}
	switch os.Args[1] {
	case "1":
		db = openDB()
		initDB(db)
	case "0":
		dbLoc := openDB()
		initDB(dbLoc)
		dbLoc.Close()
	default:
		printUsage()
		return
	}

	ch := make(chan bool)
	c <- 1 // Put the initial value into the channel
	go writer(db)
	go reader(db)
	<-ch
}

func printUsage() {
	fmt.Printf("usage: dbtest mode\n 0:multiple connection; 1:single connection\n")
}

func initDB(db *sql.DB) {
	sSql := `
                CREATE TABLE counters (
                    id      INTEGER,
                    intro         VARCHAR (255)
                )`
	db.Exec(sSql)
}

func openDB() *sql.DB {
	db, err := sql.Open("sqlite3", "file:/tmp/testlock.db")
	if err != nil {
		log.Printf("open error %s", err)
		return nil
	}
	db.Exec("PRAGMA journal_mode=WAL")
	return db
}

func writer(db *sql.DB) {
	var dbLoc *sql.DB
	if db == nil {
		dbLoc = openDB()
	} else {
		dbLoc = db
	}
	sSql := `
                INSERT INTO counters (
                        id,intro
                ) VALUES(
                        ?,?
                )`
	dbLoc.Exec(sSql, 1, "")
	i := 0
	sSql = `
        update counters set id = ?
        `
	for {
		<-c // Grab the ticket
		_, err := dbLoc.Exec(sSql, i)
		c <- 1 // Give it back
		i += 1
		fmt.Printf("Counter: %d\r", i)
		if err != nil {
			log.Printf("db error in writer %s", err)
			os.Exit(1)
		}
	}
}

func reader(db *sql.DB) {
	var dbLoc *sql.DB
	if db == nil {
		dbLoc = openDB()
	} else {
		dbLoc = db
	}
	sSql := `
        select * from counters
        `
	for {
		<-c // Grab the ticket
		_, err := dbLoc.Exec(sSql)
		c <- 1 // Give it back
		if err != nil {
			log.Printf("db error in reader %s", err)
			os.Exit(1)
		}
	}
}
