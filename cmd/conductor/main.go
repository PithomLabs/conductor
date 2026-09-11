package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/PithomLabs/conductor/internal/adapter/solvent"
	"github.com/PithomLabs/conductor/internal/api"
	"github.com/PithomLabs/conductor/internal/governance"
	"github.com/PithomLabs/conductor/internal/mcp"
	"github.com/PithomLabs/conductor/internal/service"
	"github.com/PithomLabs/conductor/internal/store"
	"github.com/PithomLabs/conductor/internal/web"
)

func main() {
	mode := flag.String("mode", "api", "Server mode: 'api' for HTTP server, 'mcp' for MCP stdio server, 'web' for UI")
	addr := flag.String("addr", ":8080", "HTTP listen address (for API/web mode)")
	dbPath := flag.String("db", "conductor.db", "SQLite database path")
	flag.Parse()

	credentials := parseCredentials(os.Getenv("CONDUCTOR_API_KEY"))

	db, err := store.Open(*dbPath)
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}
	defer db.Close()

	// Wire GovernanceService with provider routing
	govService := service.NewGovernanceService()
	govService.RegisterProvider("null", &governance.NullReader{})

	if solventURL := os.Getenv("SOLVENT_URL"); solventURL != "" {
		solventKey := os.Getenv("SOLVENT_API_KEY")
		client := solvent.NewClient(solventURL, solventKey)
		adapter := solvent.NewAdapter(client)
		govService.RegisterProvider("solvent", adapter)
		log.Printf("Solvent governance provider registered: %s", solventURL)
	}

	switch *mode {
	case "api":
		if len(credentials) == 0 {
			log.Fatal("CONDUCTOR_API_KEY required for API mode (format: key1:actor-1,key2:actor-2)")
		}
		server := api.NewServer(*addr, db, govService, credentials)

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigCh
			fmt.Println("\nshutting down...")
			os.Exit(0)
		}()

		if err := server.ListenAndServe(); err != nil {
			log.Fatalf("server error: %v", err)
		}

	case "mcp":
		// MCP agent identity is deterministic: explicit env var or fixed default.
		// Do NOT derive from credential map iteration (non-deterministic).
		agentID := os.Getenv("CONDUCTOR_MCP_AGENT_ID")
		if agentID == "" {
			agentID = "agent-mcp"
		}
		server := mcp.NewServer(db, govService, agentID)

		fmt.Fprintln(os.Stderr, "Conductor MCP server running")
		if err := server.Run(); err != nil {
			log.Fatalf("mcp server error: %v", err)
		}

	case "web":
		// Default web binding is loopback (127.0.0.1) for trusted-local POC.
		// Remote exposure requires an authenticated deployment boundary in the future.
		if *addr == ":8080" {
			*addr = "127.0.0.1:8080"
		}
		server := web.NewServer(*addr, db, govService)

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigCh
			fmt.Println("\nshutting down...")
			os.Exit(0)
		}()

		if err := server.ListenAndServe(); err != nil {
			log.Fatalf("web server error: %v", err)
		}

	default:
		log.Fatalf("unknown mode: %s (use 'api', 'mcp', or 'web')", *mode)
	}
}

func parseCredentials(raw string) map[string]string {
	creds := make(map[string]string)
	if raw == "" {
		return creds
	}
	for _, pair := range strings.Split(raw, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		parts := strings.SplitN(pair, ":", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			continue
		}
		creds[parts[0]] = parts[1]
	}
	return creds
}
