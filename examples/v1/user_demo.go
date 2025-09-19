package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/vatesfr/xenorchestra-go-sdk/client"
)

func main() {
	// Create custom logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	config := client.GetConfigFromEnv()
	xoClient, err := client.NewClientWithLogger(config, logger)
	if err != nil {
		panic(err)
	}

	user, err := xoClient.CreateUser(client.User{
		Email:    "golang-client-test",
		Password: "password",
	})
	if err != nil {
		panic(err)
	}
	fmt.Printf("Created user id: %v\n", user.Id)

	user, err = xoClient.GetUser(client.User{Email: "golang-client-test"})
	if err != nil {
		panic(err)
	}
	fmt.Println("User found: ", user)

	err = xoClient.DeleteUser(client.User{Id: user.Id})
	if err != nil {
		panic(err)
	}
	fmt.Println("User deleted")
}
