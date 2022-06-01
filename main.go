package main

import (
	"YP_GO/cmd/agent"
	"YP_GO/cmd/server"
)

func main() {
	go server.RunServer()
	agent.RunAgent()
}
