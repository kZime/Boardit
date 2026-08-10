package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"backend/internal/database"

	"github.com/joho/godotenv"
)

func main() {
	direction := flag.String("direction", "up", "migration direction: up or down")
	flag.Parse()

	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Fatalf("load .env: %v", err)
	}
	db, err := database.OpenWithDSN(os.Getenv("DATABASE_DSN"))
	if err != nil {
		log.Fatal(err)
	}

	switch *direction {
	case "up":
		err = database.Migrate(db)
	case "down":
		err = database.RollbackLast(db)
	default:
		err = fmt.Errorf("unsupported direction %q", *direction)
	}
	if err != nil {
		log.Fatal(err)
	}
}
