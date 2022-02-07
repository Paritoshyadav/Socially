package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/paritoshyadav/socialnetwork/internal/handler"
	"github.com/paritoshyadav/socialnetwork/internal/service"
	"github.com/paritoshyadav/socialnetwork/internal/service/codec"
)

func main() {
	var (
		port        = env("PORT", ":8000")
		databaseURL = env("DATABASE_URL", "postgresql://root@127.0.0.1:26257/socially?sslmode=disable") //add database name in link
		secrettoken = env("BRANCA_TOKEN", "supersecretkeyyoushouldnotcommit")
		origin      = env("ORIGIN", "http://localhost"+port)
	)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		log.Fatal("could not connect database ", err)
		return
	}

	err = db.Ping(ctx)

	if err != nil {
		log.Fatal("could not ping database", err)
		return
	}
	defer db.Close(context.Background())

	c := codec.New(secrettoken, service.TokenLifetime)

	s := service.New(db, c, origin)

	fmt.Println(s)
	defer func() {
		fmt.Println("db closed")

	}()

	h := handler.New(s)
	log.Println("Listing on port 8000...")
	err = http.ListenAndServe(port, h)
	if err != nil {
		log.Fatalf("could not listen or start server %v", err)
	}

}

func env(Key, fallbackValue string) string {
	s := os.Getenv(Key)
	if s == "" {
		return fallbackValue
	}
	return s
}
