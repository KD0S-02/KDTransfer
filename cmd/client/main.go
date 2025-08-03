package main

import (
	"fmt"
	"os"

	"github.com/KD0S-02/KDTransfer/internal/cli"
)

func main() {
	cli := cli.NewCLI()
	if err := cli.Run(); err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
}
