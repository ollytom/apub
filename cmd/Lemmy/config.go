package main

import (
	"bufio"
	"os"
)

func readCreds(name string) (username, password string, err error) {
	f, err := os.Open(name)
	if err != nil {
		return "", "", err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Scan()
	username = sc.Text()
	sc.Scan()
	password = sc.Text()
	return username, password, sc.Err()
}
