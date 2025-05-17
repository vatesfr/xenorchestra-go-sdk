package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/vatesfr/xenorchestra-go-sdk/pkg/config"
	v2 "github.com/vatesfr/xenorchestra-go-sdk/v2"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cfg, err := config.New()
	if err != nil {
		panic(err)
	}

	client, err := v2.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create XO client: %v", err)
	}

	vmService := client.VM()

	fmt.Println("Listing all VMs...")
	vms, err := vmService.List(ctx, map[string]any{"limit": 10})
	if err != nil {
		log.Fatalf("Failed to list VMs: %v", err)
	}

	fmt.Printf("Found %d VMs\n", len(vms))
	for i, vm := range vms {
		fmt.Printf("%d. %s (ID: %s, Power: %s)\n", i+1, vm.NameLabel, vm.ID, vm.PowerState)
	}

	if len(vms) > 0 {
		firstVM := vms[0]
		fmt.Printf("\nGetting details for VM: %s\n", firstVM.NameLabel)

		vmDetails, err := vmService.GetByID(ctx, firstVM.ID)
		if err != nil {
			log.Fatalf("Failed to get VM details: %v", err)
		}

		fmt.Printf("VM Details:\n")
		fmt.Printf("  Name: %s\n", vmDetails.NameLabel)
		fmt.Printf("  Description: %s\n", vmDetails.NameDescription)
		fmt.Printf("  Power State: %s\n", vmDetails.PowerState)
		fmt.Printf("  CPUs: %d\n", vmDetails.CPUs.Number)
	}
}
