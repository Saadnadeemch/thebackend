package main

import (
	"log"
	"net/http"
	"time"

	"backend/router"
	utils "backend/utils"

	"github.com/rs/cors"
)

func main() {
	go func() {
		for {
			if err := utils.DeleteFilesOlderThan("downloads", 24); err != nil {
				log.Printf("❌ Cleanup error: %v", err)
			}
			time.Sleep(1 * time.Hour)
		}
	}()

	r := router.SetupRouter()

	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	}).Handler(r)

	log.Println("🚀 Server running at http://localhost:8080")
	if err := http.ListenAndServe(":8080", corsHandler); err != nil {
		log.Fatalf("❌ Server failed: %v", err)
	}
}
