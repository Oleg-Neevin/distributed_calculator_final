package main

import (
	"github.com/Oleg-Neevin/distributed_calculator_final/internal/agent"
	"github.com/Oleg-Neevin/distributed_calculator_final/internal/orchestrator"
)

func main() {
	go agent.StartAgent()
	orchestrator.RunOrchestrator()

}
