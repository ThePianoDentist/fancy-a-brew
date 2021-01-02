package main

import (
	"fmt"
	"os"

	"github.com/ThePianoDentist/fancy-a-brew/app"

	"go.uber.org/zap"
)

func main() {
	lgr, _ := zap.NewProduction()
	defer lgr.Sync()
	fmt.Println("APP_DB_PASSWORD:")
	fmt.Println(os.Getenv("APP_DB_PASSWORD"))
	a := app.NewApp(
		lgr,
		os.Getenv("APP_DB_USERNAME"),
		os.Getenv("APP_DB_PASSWORD"),
		os.Getenv("APP_DB_NAME"),
	)

	a.Run("0.0.0.0:8081")

	fmt.Println("vim-go")
}
