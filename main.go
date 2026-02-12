package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/innogames/serveradmin-go-client/adminapi"
)

// adminapi CLI entry point
func main() {
	var attributes string
	var orderBy string
	var onlyOne bool
	flag.StringVar(&attributes, "a", "hostname", "Attributes to fetch")
	flag.StringVar(&orderBy, "order", "", "Attributes to order by the result")
	flag.BoolVar(&onlyOne, "one", false, "Make sure exactly one server matches with the query")

	flag.Parse()

	query := flag.Arg(0)
	if query == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	q, err := adminapi.FromQuery(query)
	if err != nil {
		fmt.Println("Error parsing query:", err)
		os.Exit(1)
	}

	attributeList := strings.Split(attributes, ",")
	q.SetAttributes(attributeList...)
	q.OrderBy(orderBy)

	servers, err := q.All()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if onlyOne && len(servers) != 1 {
		fmt.Println("expected exactly one server object, got", len(servers))
		os.Exit(1)
	}

	for _, server := range servers {
		for _, arg := range attributeList {
			fmt.Printf("%v ", server.Get(arg))
		}
		fmt.Print("\n")
	}
}
