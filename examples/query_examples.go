package main

import (
	"fmt"

	api "github.com/innogames/serveradmin-go-client/adminapi"
)

func stringQueryExample() {
	// Simple string-based query
	q, err := api.FromQuery("hostname=webserver01 environment=production")
	checkErr(err)

	servers, err := q.All()
	checkErr(err)

	fmt.Printf("Found %d servers using string query\n", len(servers))
}

func simpleFilterExample() {
	// Create query programmatically with simple filters passed directly
	q := api.NewQuery(api.Filters{
		"environment": "production",
		"state":       "online",
		"num_cpu":     8,
	})
	q.SetAttributes("hostname", "num_cpu", "memory")

	servers, err := q.All()
	checkErr(err)

	fmt.Printf("Found %d production servers with 8 CPUs\n", len(servers))
}

func regexpFilterExample() {
	// Use Regexp filter to match hostnames
	q := api.NewQuery(api.Filters{
		"hostname":    api.Regexp("^web.*\\.example\\.com$"),
		"environment": "production",
	})

	servers, err := q.All()
	checkErr(err)

	fmt.Printf("Found %d web servers matching pattern\n", len(servers))
}

func anyAllFilterExample() {
	// Use Any filter to match multiple possible values
	q := api.NewQuery(api.Filters{
		"game_world": api.Any(1, 2, 3),
		"state":      api.Any("online", "maintenance"),
	})

	servers, err := q.All()
	checkErr(err)

	fmt.Printf("Found %d servers in game worlds 1, 2, or 3\n", len(servers))

	// Use All filter to match all conditions
	q2 := api.NewQuery(api.Filters{
		"tags": api.All("backup", "critical"),
	})

	servers2, err := q2.All()
	checkErr(err)

	fmt.Printf("Found %d servers with both 'backup' and 'critical' tags\n", len(servers2))
}

func notFilterExample() {
	// Use Not with Empty to find servers with non-empty values
	q := api.NewQuery(api.Filters{
		"backup_disabled": false,
		"comment":         api.NotEmpty(),
	})

	servers, err := q.All()
	checkErr(err)

	fmt.Printf("Found %d servers with backup_disabled and comment set\n", len(servers))

	// Use Not with specific value
	q2 := api.NewQuery(api.Filters{})
	q2.AddFilter("environment", api.Not("development"))

	servers2, err := q2.All()
	checkErr(err)

	fmt.Printf("Found %d non-development servers\n", len(servers2))
}

func nestedFilterExample() {
	// Complex nested filters: servers that don't match certain patterns
	q := api.NewQuery(api.Filters{})

	// Find servers where hostname is NOT matching any of these patterns
	q.AddFilter("hostname", api.Not(api.Any(
		api.Regexp("^test.*"),
		api.Regexp("^dev.*"),
		api.Regexp("^tmp.*"),
	)))

	// Environment must be production or staging, but not empty
	q.AddFilter("environment", api.Any("production", "staging"))

	servers, err := q.All()
	checkErr(err)

	fmt.Printf("Found %d servers with complex nested filters\n", len(servers))
}

func combinedFilterExample() {
	// Real-world example: Find suitable servers for migration
	q := api.NewQuery(api.Filters{})

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

	servers, err := q.All()
	checkErr(err)

	fmt.Printf("Found %d servers suitable for migration:\n", len(servers))
	for _, server := range servers {
		fmt.Printf("  - %s: %v CPUs, %v GB RAM, project: %s\n",
			server.GetString("hostname"),
			server.Get("num_cpu"),
			server.Get("memory"),
			server.GetString("project"),
		)
	}
}
