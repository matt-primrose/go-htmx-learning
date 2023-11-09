package main

import (
	"embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jrapoport/chestnut"
	"github.com/jrapoport/chestnut/encryptor/aes"
	"github.com/jrapoport/chestnut/encryptor/crypto"
	"github.com/jrapoport/chestnut/storage/bolt"
	"github.com/jritsema/go-htmx-starter/internal"
	"github.com/jritsema/go-htmx-starter/internal/certificates"
	"github.com/jritsema/go-htmx-starter/internal/dashboard"
	"github.com/jritsema/go-htmx-starter/internal/devices"
	"github.com/jritsema/go-htmx-starter/internal/profiles"
	"github.com/jritsema/gotoolbox"
)

var (
	//go:embed css/output.css
	css embed.FS
)

func main() {

	//exit process immediately upon sigterm
	handleSigTerms()

	//add routes
	router := http.NewServeMux()

	router.Handle("/css/output.css", http.FileServer(http.FS(css)))
	// Open the my.db data file in your current directory.
	// It will be created if it doesn't exist.

	// use nutsdb for storage
	store := bolt.NewStore("my.db")

	textSecret := crypto.NewManagedSecret("amtid", "P@ssw0rd")
	// use AES256-CFB for encryption
	opt := chestnut.WithAES(crypto.Key256, aes.CFB, textSecret)

	cn := chestnut.NewChestnut(store, opt)
	if err := cn.Open(); err != nil {
		log.Fatal(err)
	}

	defer cn.Close()
	// db, err := bbolt.Open("my.db", 0600, nil)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// defer db.Close()

	_ = devices.NewDevices(cn, router)
	_ = certificates.NewCertificates(router)
	_ = profiles.NewProfiles(router)
	_ = dashboard.NewDashboard(router)

	_ = internal.NewIndex(router)

	//logging/tracing
	nextRequestID := func() string {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	logger := log.New(os.Stdout, "http: ", log.LstdFlags)
	middleware := internal.Tracing(nextRequestID)(internal.Logging(logger)(router))

	port := gotoolbox.GetEnvWithDefault("PORT", "8080")
	logger.Println("listening on http://localhost:" + port)
	if err := http.ListenAndServe("localhost:"+port, middleware); err != nil {
		logger.Println("http.ListenAndServe():", err)
		os.Exit(1)
	}
}

func handleSigTerms() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("received SIGTERM, exiting")
		os.Exit(1)
	}()
}
