package main

import "os"

import "github.com/tokenshift/mdweb"

func main() {
	mdweb.ProcessFiles(true, false, os.Args[1:]...)
}
