package main

import (
	"context"
	"log"
	"time"

	api "github.com/innogames/serveradmin-go-client/adminapi"
)

// clientExample shows the recommended entry point: an explicit, per-instance
// Client constructed from a Config. No environment variables are read, and the
// client is safe for concurrent use, so a single process can hold several
// clients pointing at different targets.
func clientExample() {
	client, err := api.NewClient(api.Config{
		BaseURL: "https://serveradmin.example.com",
		Token:   "your-token",
		Timeout: 10 * time.Second,
	})
	checkErr(err)

	ctx := context.Background()

	q, err := client.FromQuery("hostname=webserver01 environment=production")
	checkErr(err)

	servers, err := q.All(ctx)
	checkErr(err)

	log.Printf("Found %d servers using the client API\n", len(servers))
}

func stringQueryExample() {
	// Simple string-based query
	q, err := client.FromQuery("hostname=webserver01 environment=production")
	checkErr(err)

	servers, err := q.All(context.Background())
	checkErr(err)

	log.Printf("Found %d servers using string query\n", len(servers))
}

func simpleFilterExample() {
	// Create query programmatically with simple filters passed directly
	q := client.NewQuery(api.Filters{
		"environment": "production",
		"state":       "online",
		"num_cpu":     api.LessThanOrEquals(4),
	})
	q.SetAttributes("hostname", "num_cpu", "memory")

	servers, err := q.All(context.Background())
	checkErr(err)

	log.Printf("Found %d production servers with 8 CPUs\n", len(servers))
}

func regexpFilterExample() {
	// Use Regexp filter to match hostnames
	q := client.NewQuery(api.Filters{
		"hostname":    api.Regexp("^web.*\\.example\\.com$"),
		"environment": "production",
	})

	servers, err := q.All(context.Background())
	checkErr(err)

	log.Printf("Found %d web servers matching pattern\n", len(servers))
}

func anyAnyFilterExample() {
	// Use Any filter to match multiple possible values
	q := client.NewQuery(api.Filters{
		"game_world": api.GreaterThan(1),
		"state":      api.Any("online", "maintenance"),
	})

	servers, err := q.All(context.Background())
	checkErr(err)
	log.Printf("Found %d servers:", len(servers))
}

func nestedFilterExample() {
	// Complex nested filters: servers that don't match certain patterns
	q := client.NewQuery(api.Filters{})

	// Find servers where hostname is NOT matching any of these patterns
	q.AddFilter("hostname", api.Not(api.Any(
		api.Regexp("^test.*"),
		api.Regexp("^dev.*"),
		api.Regexp("^tmp.*"),
	)))

	// Environment must be production or staging, but not empty
	q.AddFilter("environment", api.Any("production", "staging"))

	servers, err := q.All(context.Background())
	checkErr(err)

	log.Printf("Found %d servers with complex nested filters\n", len(servers))
}

func combinedFilterExample() {
	q := client.NewQuery(api.Filters{})

	q.AddFilter("servertype", "server")
	q.AddFilter("environment", "production")
	q.AddFilter("state", api.Any("online", "deploy_online"))
	q.AddFilter("num_cpu", api.Any(4, 8, 16))

	// Must NOT be marked for decommission
	q.AddFilter("decommission", api.Not(true))

	// Must have a hostname that doesn't start with "legacy"
	q.AddFilter("hostname", api.Not(api.Regexp("^legacy.*")))

	// Must have non-empty project assignment
	q.AddFilter("project", api.NotEmpty())

	// Only fetch required attributes
	q.SetAttributes(
		"hostname",
		"environment",
		"num_cpu",
		"memory",
		"state",
		"project",
		"object_id",
	)

	servers, err := q.All(context.Background())
	checkErr(err)

	log.Printf("Found %d servers suitable for migration:\n", len(servers))
	for _, server := range servers {
		log.Printf("  - %s: %v CPUs, %v GB RAM, project: %s\n",
			server.GetString("hostname"),
			server.Get("num_cpu"),
			server.Get("memory"),
			server.GetString("project"),
		)
	}
}

func multiAttrExample() {
	// Fetch a server with multi-valued attributes
	q, err := client.FromQuery("hostname=webserver01")
	checkErr(err)

	server, err := q.One(context.Background())
	checkErr(err)

	// Get tags as MultiAttr
	tags := server.GetMulti("tags")

	// Use MultiAttr convenience methods
	tags.Add("monitoring", "web") // Add tags (web is duplicate, won't be added)
	tags.Delete("old-tag")        // Remove old tag

	if tags.Contains("monitoring") {
		log.Println("Server has monitoring tag")
	}

	// Set back to ServerObject and commit
	checkErr(server.Set("tags", []string(tags)))
	commitID, err := server.Commit(context.Background())
	checkErr(err)

	log.Printf("Updated tags for %s (commit %d)\n", server.GetString("hostname"), commitID)
}
