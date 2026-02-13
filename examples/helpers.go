package main

import "log"

// checkErr is a helper function for examples that exits on error
func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
