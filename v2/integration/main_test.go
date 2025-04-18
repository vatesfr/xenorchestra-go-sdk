package integration

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/subosito/gotenv"
)

var (
	skipTeardown = flag.Bool("skip-teardown", false, "Skip teardown of resources")
)

func init() {
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current directory: %s\n", err)
		return
	}

	rootDir := filepath.Join(currentDir, "..", "..")
	envFile := filepath.Join(rootDir, ".env")

	fmt.Printf("Loading .env from: %s\n", envFile)
	err = gotenv.Load(envFile)
	if err != nil {
		fmt.Printf("Error loading .env file: %s\n", err)
	} else {
		fmt.Println("Successfully loaded .env file")

		fmt.Printf("XOA_URL: %s\n", os.Getenv("XOA_URL"))
		fmt.Printf("XOA_POOL: %s\n", os.Getenv("XOA_POOL"))
		fmt.Printf("XOA_TEMPLATE: %s\n", os.Getenv("XOA_TEMPLATE"))
		fmt.Printf("XOA_NETWORK: %s\n", os.Getenv("XOA_NETWORK"))
		fmt.Printf("XOA_INTEGRATION_TESTS: %s\n", os.Getenv("XOA_INTEGRATION_TESTS"))

		if os.Getenv("XOA_INTEGRATION_TESTS") == "" {
			os.Setenv("XOA_INTEGRATION_TESTS", "true")
			fmt.Println("Automatically enabled integration tests (XOA_INTEGRATION_TESTS=true)")
		}
	}
}

func TestMain(m *testing.M) {
	flag.Parse()

	if *skipTeardown {
		os.Setenv("XOA_SKIP_TEARDOWN", "true")
	}

	code := m.Run()

	os.Exit(code)
}
