package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	aiagentruntime "logic-grpc-service/internal/aiagent/runtime"
)

func main() {
	describe := flag.Bool("describe", false, "print the AI Agent service skeleton descriptor")
	check := flag.Bool("check", false, "validate the AI Agent service skeleton and exit")
	flag.Parse()

	descriptor, err := aiagentruntime.NewDescriptor()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ai-agent-service skeleton invalid: %v\n", err)
		os.Exit(1)
	}

	if *check {
		return
	}
	if *describe {
		printDescriptor(descriptor)
		return
	}

	fmt.Fprintln(os.Stderr, "ai-agent-service skeleton is compile-safe but intentionally unrouted.")
	fmt.Fprintln(os.Stderr, "Run with --describe for details or --check for validation.")
	os.Exit(2)
}

func printDescriptor(descriptor aiagentruntime.Descriptor) {
	fmt.Printf("name: %s\n", descriptor.Unit.Name)
	fmt.Printf("role: %s\n", descriptor.Unit.Role)
	fmt.Printf("command: %s\n", descriptor.Unit.Command)
	fmt.Printf("image: %s\n", descriptor.Unit.Image)
	fmt.Printf("config_prefix: %s\n", descriptor.Unit.ConfigPrefix)
	fmt.Printf("health: %s\n", descriptor.Unit.Health)
	fmt.Printf("cutover_mode: %s\n", descriptor.CutoverMode)
	fmt.Printf("traffic_enabled: %t\n", descriptor.TrafficEnabled)
	fmt.Printf("startup_mode: %s\n", descriptor.StartupMode)
	fmt.Printf("notes: %s\n", strings.Join(descriptor.Notes, "; "))
}
