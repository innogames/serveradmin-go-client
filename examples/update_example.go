package main

import (
	"fmt"
	"log"

	"github.com/innogames/serveradmin-go-client/adminapi"
)

// Example demonstrating how to update server objects
func main() {
	// Example 1: Update a single object
	singleObjectExample()

	// Example 2: Update multiple objects in batch
	multiObjectExample()

	// Example 3: Create a new object
	createObjectExample()

	// Example 4: Delete an object
	deleteObjectExample()

	// Example 5: Rollback changes
	rollbackExample()
}

func singleObjectExample() {
	q, err := adminapi.FromQuery("hostname=webserver01")
	if err != nil {
		log.Fatal(err)
	}

	server, err := q.One()
	if err != nil {
		log.Fatal(err)
	}

	// Modify attributes
	server.Set("backup_disabled", true)
	server.Set("comment", "Updated via Go client")

	// Commit changes
	commitID, err := server.Commit()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Updated server %s (commit %d)\n", server.GetString("hostname"), commitID)
}

func multiObjectExample() {
	q, err := adminapi.FromQuery("environment=production state=online")
	if err != nil {
		log.Fatal(err)
	}
	q.SetAttributes([]string{"hostname", "backup_disabled", "object_id"})

	servers, err := q.All()
	if err != nil {
		log.Fatal(err)
	}

	// Update all servers
	for _, server := range servers {
		server.Set("backup_disabled", false)
	}

	// Commit all changes in a single API call
	commitID, err := servers.Commit()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Updated %d servers (commit %d)\n", len(servers), commitID)
}

func createObjectExample() {
	// Create a new VM object
	newVM, err := adminapi.NewObject("vm")
	if err != nil {
		log.Fatal(err)
	}

	// Set required attributes
	newVM.Set("hostname", "newserver.example.com")
	newVM.Set("environment", "development")
	newVM.Set("num_cpu", 4)

	// Commit creates the object on the server
	commitID, err := newVM.Commit()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Created new VM %s (commit %d)\n", newVM.GetString("hostname"), commitID)
}

func deleteObjectExample() {
	q, err := adminapi.FromQuery("hostname=oldserver.example.com")
	if err != nil {
		log.Fatal(err)
	}

	server, err := q.One()
	if err != nil {
		log.Fatal(err)
	}

	// Mark for deletion
	server.Delete()

	// Commit the deletion
	commitID, err := server.Commit()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Deleted server (commit %d)\n", commitID)
}

func rollbackExample() {
	q, err := adminapi.FromQuery("hostname=webserver01")
	if err != nil {
		log.Fatal(err)
	}

	server, err := q.One()
	if err != nil {
		log.Fatal(err)
	}

	// Make some changes
	originalHostname := server.GetString("hostname")
	server.Set("hostname", "modified-name.local")
	fmt.Printf("Modified hostname: %s\n", server.GetString("hostname"))

	// Rollback the changes
	server.Rollback()
	fmt.Printf("After rollback: %s\n", server.GetString("hostname"))

	// Check commit state
	fmt.Printf("Commit state: %s\n", server.CommitState()) // Should be "consistent"

	if server.GetString("hostname") != originalHostname {
		log.Fatal("Rollback failed!")
	}
}
