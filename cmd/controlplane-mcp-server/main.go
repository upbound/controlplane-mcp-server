/*
Package main is the main entrypoint into the mcp server.
*/
package main

import (
	"fmt"

	"github.com/alecthomas/kong"
	"github.com/mark3labs/mcp-go/server"
	"go.uber.org/zap/zapcore"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/crossplane/function-sdk-go/logging"

	"github.com/upbound/controlplane-mcp-server/internal/tool"
)

const (
	version = "0.0.1"
	name    = "controlplane-mcp-server"
	desc    = "Upbound ControlPlane MCP Server"
)

// Command contains all the options for the runnable.
type Command struct {
	Debug   bool `default:"false" env:"DEBUG"    help:"Run with debug logging."   name:"debug"    short:"d"`
	DevMode bool `default:"false" env:"DEV_MODE" help:"Enables logging dev mode." name:"dev-mode"`

	Port       string `default:":8080" help:"Address to listen on."`
	Kubeconfig string `default:""      help:"Location of the kubeconfig to use for the API clients. Default is to use the incluster config."`
}

func main() {
	cmd := Command{}

	kongCtx := kong.Parse(&cmd,
		kong.Name(name),
		kong.Description(desc),
		kong.UsageOnError(),
	)

	// initialize a new MCP server.
	s := server.NewMCPServer(
		desc,
		version,
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)

	// specify logging options
	zapOpts := []zap.Opts{}
	if cmd.Debug {
		zapOpts = append(zapOpts, zap.Level(zapcore.DebugLevel))
	}
	if cmd.DevMode {
		zapOpts = append(zapOpts, zap.UseDevMode(true))
	}

	zl := zap.New(zapOpts...)
	nzl := zl.WithName(name)
	log := logging.NewLogrLogger(nzl)
	ctrllog.SetLogger(nzl)

	cfg, err := ctrl.GetConfig()
	kongCtx.FatalIfErrorf(err, "failed to retrieve Kubeconfig")

	clientset, err := kubernetes.NewForConfig(cfg)
	kongCtx.FatalIfErrorf(err, "failed to construct clientset")

	ts := tool.NewServer(clientset)
	s.AddTool(tool.GetPodLogs(), ts.GetPodLogsHander)

	ss := server.NewStreamableHTTPServer(s)

	log.Info(fmt.Sprintf("Streamable HTTP server starting at http://localhost%s/mcp", cmd.Port))
	kongCtx.FatalIfErrorf(ss.Start(cmd.Port), "failed to start SSE server")
}
