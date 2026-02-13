package main

import (
	"fmt"

	api "github.com/innogames/serveradmin-go-client/adminapi"
)

func main() {
	var commitID int

	// Step 1: Check if object already exists
	fmt.Println("=== Checking for existing public_domain object ===")
	q, err := api.FromQuery("hostname=test.foo.com servertype=public_domain")
	checkErr(err)
	q.AddAttributes("dns_txt")

	publicURL, err := q.One()
	if err != nil {
		// Object doesn't exist, create it
		fmt.Println("=== Object not found, creating new public_domain object ===")
		publicURL, err = api.NewObject("public_domain")
		checkErr(err)

		// Set required attributes
		publicURL.Set("hostname", "test.foo.com")
		publicURL.Set("project", "admin")

		// Commit the new object
		commitID, err = publicURL.Commit()
		checkErr(err)
		fmt.Printf("Created public_url %s (commit ID: %d)\n", publicURL.GetString("hostname"), commitID)

		// Re-query to get the server-assigned object_id
		q, err = api.FromQuery("hostname=test.foo.com servertype=public_domain")
		checkErr(err)
		q.AddAttributes("dns_txt")
		publicURL, err = q.One()
		checkErr(err)
		fmt.Printf("Re-queried object_id: %d\n", publicURL.ObjectID())
	} else {
		fmt.Printf("Found existing object with object_id: %d\n", publicURL.ObjectID())
	}

	// Step 2: Set dns_txt to "foobar"
	fmt.Println("\n=== Setting dns_txt to foobar ===")
	publicURL.Set("dns_txt", []string{"foobar"})

	// Commit the update
	commitID, err = publicURL.Commit()
	checkErr(err)
	fmt.Printf("Set dns_txt to %v (commit ID: %d)\n", publicURL.Get("dns_txt"), commitID)

	// Step 3: Add a random string to dns_txt
	fmt.Println("\n=== Adding random string to dns_txt ===")
	publicURL.Set("dns_txt", []string{"foobar", "random_value_xyz123"})

	// Commit the second update
	commitID, err = publicURL.Commit()
	checkErr(err)
	fmt.Printf("Added to dns_txt, now: %v (commit ID: %d)\n", publicURL.Get("dns_txt"), commitID)

	// Step 4: Delete the object
	fmt.Println("\n=== Deleting object ===")
	publicURL.Delete()
	commitID, err = publicURL.Commit()
	checkErr(err)
	fmt.Printf("Deleted public_url (commit ID: %d)\n", commitID)

	fmt.Println("\n=== Complete ===")
}
