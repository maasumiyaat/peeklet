package main

// Command hashpw prints a bcrypt hash of the owner password, for use as the
// OwnerPasswordHash stack parameter.
//
//	go run ./cmd/hashpw "my-strong-password"

import (
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	if len(os.Args) != 2 || os.Args[1] == "" {
		fmt.Fprintln(os.Stderr, "usage: hashpw <password>")
		os.Exit(2)
	}
	h, err := bcrypt.GenerateFromPassword([]byte(os.Args[1]), bcrypt.DefaultCost)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	fmt.Println(string(h))
}
