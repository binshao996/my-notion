package main

import "log"

func main() {
	log.Println("Worker started (noop for M0)")
	select {}
}
