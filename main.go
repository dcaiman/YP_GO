package main

import (
	"YP_GO_devops/cmd/agent"
	"YP_GO_devops/cmd/server"
)

func main() {
	go agent.RunAgent()
	server.RunServer()
}
