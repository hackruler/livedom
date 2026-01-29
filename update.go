package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/fatih/color"
)

// updateTool updates livedom to the latest version
func updateTool() {
	fmt.Println(color.New(color.FgCyan).Sprint("Updating livedom to the latest version..."))

	// First update the module to latest
	getCmd := exec.Command("go", "get", "-u", "github.com/hackruler/livedom@latest")
	getCmd.Stdout = os.Stdout
	getCmd.Stderr = os.Stderr
	getCmd.Run() // Run get, ignore errors

	// Then install the latest version
	cmd := exec.Command("go", "install", "github.com/hackruler/livedom@latest")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		fmt.Println(color.New(color.FgRed).Sprint("Error updating livedom:"), err)
		os.Exit(1)
	}

	fmt.Println(color.New(color.FgGreen).Sprint("✓ Successfully updated livedom!"))
}
