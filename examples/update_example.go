package main

import (
	"context"
	"fmt"

	api "github.com/innogames/serveradmin-go-client/adminapi"
)

func singleObjectExample() {
	q, err := client.FromQuery("hostname=webserver01")
	checkErr(err)
	q.AddAttributes("backup_disabled", "tags")

	server, err := q.One(context.Background())
	checkErr(err)

	// Modify attributes
	server.Set("backup_disabled", true)

	// Get and update  multi-valued attribute as MultiAttr
	tags := server.GetMulti("tags")
	tags.Add("monitoring", "web")
	tags.Delete("old-tag")

	// Commit changes
	commitID, err := server.Commit(context.Background())
	checkErr(err)

	fmt.Printf("Updated server %s (commit %d)\n", server.GetString("hostname"), commitID)
}

func multiObjectExample() {
	q, err := client.FromQuery("environment=production state=online")
	checkErr(err)
	q.SetAttributes("hostname", "backup_disabled")

	servers, err := q.All(context.Background())
	checkErr(err)

	// Update all servers using batch Set()
	servers.Set("backup_disabled", false)

	// Commit all changes in a single API call
	commitID, err := servers.Commit(context.Background())
	checkErr(err)

	fmt.Printf("Updated %d servers (commit %d)\n", len(servers), commitID)
}

func createObjectExample() {
	// Create a new VM object — NewObject fetches defaults, sets attributes, commits,
	// and re-queries to populate object_id in a single call.
	newVM, err := client.NewObject(context.Background(), "vm", api.Attributes{
		"hostname":    "newserver.example.com",
		"environment": "development",
		"num_cpu":     4,
	})
	checkErr(err)

	fmt.Printf("Created new VM %s (object_id: %d)\n", newVM.GetString("hostname"), newVM.ObjectID())
}

func deleteObjectExample() {
	q, err := client.FromQuery("hostname=oldserver.example.com")
	checkErr(err)

	server, err := q.One(context.Background())
	checkErr(err)

	// Mark for deletion
	server.Delete()

	// Commit the deletion
	commitID, err := server.Commit(context.Background())
	checkErr(err)

	fmt.Printf("Deleted server (commit %d)\n", commitID)
}

func batchDeleteExample() {
	q, err := client.FromQuery("servertype=domain state=retired")
	checkErr(err)

	servers, err := q.All(context.Background())
	checkErr(err)

	// Delete ALL retired domains using batch Delete()
	servers.Delete()

	// Commit all deletions in a single API call
	commitID, err := servers.Commit(context.Background())
	checkErr(err)

	fmt.Printf("Deleted %d servers (commit %d)\n", len(servers), commitID)
}

func callAPIExample() {
	// Call a remote API function
	result, err := client.CallAPI(context.Background(), "ip", "get_free", map[string]any{"network": "internal"})
	checkErr(err)

	fmt.Printf("Free IP: %s\n", result)
}

func rollbackExample() {
	q, err := client.FromQuery("hostname=webserver01")
	checkErr(err)

	server, err := q.One(context.Background())
	checkErr(err)

	// Make some changes
	server.Set("hostname", "modified-name.local")
	fmt.Printf("Modified hostname: %s\n", server.GetString("hostname"))

	// Rollback the changes
	server.Rollback()
	fmt.Printf("After rollback: %s\n", server.GetString("hostname"))
}
