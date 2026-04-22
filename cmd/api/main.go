package main

import (
	"log"

	"github.com/PhonkersBase/base-api2/internal/server"
)

func main() {
	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
}
