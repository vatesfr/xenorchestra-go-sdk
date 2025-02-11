package main

import (
	"fmt"
	"time"

	"github.com/vatesfr/xo-sdk-go/client"
)

func newClient() (client.XOClient, error) {
	duration, err := time.ParseDuration("10m")
	if err != nil {
		return nil, err
	}
	config := client.Config{
		// Use the websocket URL
		Url: "ws://xo-instance.example",
		// Use the following if you want to use basic auth
		// Username:           "admin",
		// Password:           "password",
		// XO Authentication token
		Token:              "token",
		InsecureSkipVerify: false,
		RetryMode:          client.Backoff,
		RetryMaxTime:       duration,
	}
	return client.NewClient(config)
}

func main() {
	xoClient, err := newClient()
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
	fmt.Printf("Created user id: %v", user.Id)

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
