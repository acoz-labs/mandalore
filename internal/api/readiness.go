package api

import "github.com/acoz-labs/mandalore/internal/readiness"

var readinessOperations = []Operation{
	connectionOperation("connection_assess", "Assess selected local memory-integration metadata without executing programs, reading remembered content, network access or changes. Support, setup and scenario evidence are separate; optional follow-up guidance is not authorization to repair.", true, readiness.Assess),
}

func init() { readinessOperations[0].CLIOnly = true }
