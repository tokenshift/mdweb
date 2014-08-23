package main

import "os"

import "github.com/tokenshift/mdweb"

func main() {
	mdweb.ProcessFiles(false, true, os.Args[1:]...)
}
