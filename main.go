package main

import "fontget/cmd"

func main() {
	if err := cmd.Execute(); err != nil {
		// handle error, e.g. print and exit
	}
}
