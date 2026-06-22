package main

import (
	"context"
	"log"

	api "github.com/innogames/serveradmin-go-client/adminapi"
)

// checkErr is a helper function for examples that exits on error
func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// client is a shared example client. Replace BaseURL/Token with your own, or use
// api.NewClientFromEnv() to configure it from the SERVERADMIN_* environment.
var client, _ = api.NewClient(api.Config{
	BaseURL: "https://serveradmin.example.com",
	Token:   "your-token",
})

func main() {
	ctx := context.Background()
	var commitID int

	// Step 1: Check if object already exists
	log.Println("=== Checking for existing public_domain object ===")
	q, err := client.FromQuery("hostname=test.foo.com servertype=public_domain")
	checkErr(err)
	q.AddAttributes("dns_txt")

	publicURL, err := q.One(ctx)
	if err != nil {
		// Object doesn't exist, create it
		log.Println("=== Object not found, creating new public_domain object ===")
		publicURL, err = client.NewObject(ctx, "public_domain", api.Attributes{
			"hostname": "test.foo.com",
			"project":  "admin",
			"dns_txt":  api.MultiAttr{},
		})
		checkErr(err)
		log.Printf("Created public_url %s (object_id: %d)\n", publicURL.GetString("hostname"), publicURL.ObjectID())
	} else {
		log.Printf("Found existing object with object_id: %d\n", publicURL.ObjectID())
	}

	// Step 2: Set dns_txt to "foobar"
	log.Println("\n=== Setting dns_txt to foobar ===")
	dnsTxt := publicURL.GetMulti("dns_txt")
	dnsTxt.Clear()
	dnsTxt.Add("foobar")
	publicURL.Set("dns_txt", dnsTxt)

	// Commit the update
	commitID, err = publicURL.Commit(ctx)
	checkErr(err)
	log.Printf("Set dns_txt to %v (commit ID: %d)\n", publicURL.Get("dns_txt"), commitID)

	// Step 3: Add a random string to dns_txt
	log.Println("\n=== Adding random string to dns_txt ===")
	dnsTxt = publicURL.GetMulti("dns_txt")
	dnsTxt.Add("random_value_xyz123")
	publicURL.Set("dns_txt", dnsTxt)

	// Commit the second update
	commitID, err = publicURL.Commit(ctx)
	checkErr(err)
	log.Printf("Added to dns_txt, now: %v (commit ID: %d)\n", publicURL.Get("dns_txt"), commitID)

	// Step 4: Delete the object
	log.Println("\n=== Deleting object ===")
	publicURL.Delete()
	commitID, err = publicURL.Commit(ctx)
	checkErr(err)
	log.Printf("Deleted public_url (commit ID: %d)\n", commitID)

	log.Println("\n=== Complete ===")
}
