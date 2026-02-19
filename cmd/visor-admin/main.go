package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"visor/internal/levelup"
)

func main() {
	projectRoot := flag.String("project-root", ".", "visor project root")
	flag.Parse()

	args := flag.Args()
	if len(args) < 2 || args[0] != "levelup" {
		printUsage()
		os.Exit(2)
	}

	sub := args[1]
	switch sub {
	case "list":
		statuses, err := levelup.List(*projectRoot)
		if err != nil {
			log.Fatalf("list level-ups: %v", err)
		}
		for _, s := range statuses {
			state := "disabled"
			if s.Enabled {
				state = "enabled"
			}
			if s.DisplayName != "" {
				fmt.Printf("%s (%s): %s\n", s.Name, s.DisplayName, state)
				continue
			}
			fmt.Printf("%s: %s\n", s.Name, state)
		}
	case "enable":
		names := splitNames(args[2:])
		if err := levelup.Enable(*projectRoot, names); err != nil {
			log.Fatalf("enable level-up: %v", err)
		}
		fmt.Printf("enabled: %s\n", strings.Join(names, ", "))
	case "disable":
		names := splitNames(args[2:])
		if err := levelup.Disable(*projectRoot, names); err != nil {
			log.Fatalf("disable level-up: %v", err)
		}
		fmt.Printf("disabled: %s\n", strings.Join(names, ", "))
	default:
		printUsage()
		os.Exit(2)
	}
}

func splitNames(args []string) []string {
	joined := strings.Join(args, ",")
	if strings.TrimSpace(joined) == "" {
		return nil
	}
	parts := strings.Split(joined, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

func printUsage() {
	fmt.Println("usage:")
	fmt.Println("  visor-admin [-project-root .] levelup list")
	fmt.Println("  visor-admin [-project-root .] levelup enable <name>[,<name>...]")
	fmt.Println("  visor-admin [-project-root .] levelup disable <name>[,<name>...]")
}
