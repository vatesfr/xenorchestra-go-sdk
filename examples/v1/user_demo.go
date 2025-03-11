package main

import (
	"fmt"

	"github.com/vatesfr/xenorchestra-go-sdk/client"
)

func newClient() (client.XOClient, error) {
	config := client.GetConfigFromEnv()
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
