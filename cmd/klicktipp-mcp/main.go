package main

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/klicktipp/klicktipp-binary-mcp/internal/api"
	"github.com/klicktipp/klicktipp-binary-mcp/internal/config"
	"github.com/klicktipp/klicktipp-binary-mcp/internal/tools"
	mcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

var version = "dev"

func main() {
	executable, err := os.Executable()
	if err != nil {
		log.Fatalf("find executable: %v", err)
	}

	envFile := envPath(executable)
	env, envFileFound, err := config.LoadEnvWithSource(envFile)
	if err != nil {
		log.Fatalf("load env %q: %v", envFile, err)
	}

	cfg, err := config.FromEnv(env)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	cfg.EnvFilePath = envFile
	cfg.EnvFileFound = envFileFound

	client := api.NewClient(cfg)
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "klicktipp-mcp",
		Version: version,
	}, nil)

	tools.Register(server, client, cfg, env)

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

func envPath(executable string) string {
	return filepath.Join(filepath.Dir(executable), ".env")
}
