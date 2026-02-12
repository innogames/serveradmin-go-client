package main

import (
	"fmt"
	"log"

	api "github.com/innogames/serveradmin-go-client/adminapi"
)

func singleObjectExample() {
	q, err := api.FromQuery("hostname=webserver01")
	if err != nil {
		log.Fatal(err)
	}
	q.AddAttributes("backup_disabled")

	server, err := q.One()
	if err != nil {
		log.Fatal(err)
	}

	// Modify attributes
	server.Set("backup_disabled", true)

	// Commit changes
	commitID, err := server.Commit()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Updated server %s (commit %d)\n", server.GetString("hostname"), commitID)
}

func multiObjectExample() {
	q, err := api.FromQuery("environment=production state=online")
	if err != nil {
		log.Fatal(err)
	}
	q.SetAttributes("hostname", "backup_disabled")

	servers, err := q.All()
	if err != nil {
		log.Fatal(err)
	}

	// Update all servers using batch Set()
	servers.Set("backup_disabled", false)

	// Commit all changes in a single API call
	commitID, err := servers.Commit()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Updated %d servers (commit %d)\n", len(servers), commitID)
}

func createObjectExample() {
	// Create a new VM object
	newVM, err := api.NewObject("vm")
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
	q, err := api.FromQuery("hostname=oldserver.example.com")
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

func batchDeleteExample() {
	q, err := api.FromQuery("state=decommissioned")
	if err != nil {
		log.Fatal(err)
	}

	servers, err := q.All()
	if err != nil {
		log.Fatal(err)
	}

	// Delete ALL decommissioned servers using batch Delete()
	servers.Delete()

	// Commit all deletions in a single API call
	commitID, err := servers.Commit()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Deleted %d servers (commit %d)\n", len(servers), commitID)
}

func rollbackExample() {
	q, err := api.FromQuery("hostname=webserver01")
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
