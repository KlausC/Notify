package main

import (
	"log"
	"os/exec"
)

func main() {
	cmd := exec.Command("sh", "-c", "sleep 1; exit 2")
	err := cmd.Start()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Waiting for command to finish...")
	err = cmd.Wait()
	log.Printf("Command finished with error: %v", err)
}
