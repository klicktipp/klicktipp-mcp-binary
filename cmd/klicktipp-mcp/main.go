package main

import (
	"context"
	"log"

	"github.com/klicktipp/klicktipp-binary-mcp/internal/api"
	"github.com/klicktipp/klicktipp-binary-mcp/internal/config"
	"github.com/klicktipp/klicktipp-binary-mcp/internal/tools"
	mcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	env, err := config.LoadEnv(".env")
	if err != nil {
		log.Fatalf("load env: %v", err)
	}

	cfg, err := config.FromEnv(env)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	client := api.NewClient(cfg)
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "klicktipp-mcp",
		Version: "0.2.0-go-sdk",
	}, nil)

	tools.Register(server, client, cfg, env)

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
