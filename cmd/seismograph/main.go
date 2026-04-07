package main

import (
	"flag"
	"fmt"
	"github.com/stockyard-dev/stockyard-seismograph/internal/server"
	"github.com/stockyard-dev/stockyard-seismograph/internal/store"
	"log"
	"net/http"
	"os"
)

func main() {
	portFlag := flag.String("port", "", "")
	dataFlag := flag.String("data", "", "")
	flag.Parse()
	port := os.Getenv("PORT")
	if port == "" {
		port = "9700"
	}
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./seismograph-data"
	}
	if *portFlag != "" {
		port = *portFlag
	}
	if *dataFlag != "" {
		dataDir = *dataFlag
	}
	db, err := store.Open(dataDir)
	if err != nil {
		log.Fatalf("seismograph: %v", err)
	}
	defer db.Close()
	srv := server.New(db, server.DefaultLimits(), dataDir)
	fmt.Printf("\n  Seismograph — Self-hosted error tracker\n  Dashboard:  http://localhost:%s/ui\n  API:        http://localhost:%s/api\n  Questions? hello@stockyard.dev — I read every message\n\n", port, port)
	log.Printf("seismograph: listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, srv))
}
