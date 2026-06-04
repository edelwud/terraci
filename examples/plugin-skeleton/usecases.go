package skeleton

import (
	"context"
	"fmt"

	"github.com/edelwud/terraci/pkg/ci"
	"github.com/edelwud/terraci/pkg/plugin"
)

// Request contains command flags and other command-scoped inputs.
type Request struct {
	Consume bool
}

// Result is the typed usecase output consumed by output.go.
type Result struct {
	Producer *ProducerResult
	Consumer *ConsumerResult
}

// ProducerResult is the raw result persisted by the producer flow.
type ProducerResult struct {
	Greeting   string `json:"greeting"`
	WorkDir    string `json:"work_dir"`
	ServiceDir string `json:"service_dir"`
}

// ConsumerResult is the read-side summary for other producer reports.
type ConsumerResult struct {
	Reports []ConsumedReport
}

// ConsumedReport is a small consumer-facing projection of a ci.Report.
type ConsumedReport struct {
	Producer string
	Status   ci.ReportStatus
	Summary  string
	Sections []ConsumedSection
}

// ConsumedSection describes one decoded render-ready section.
type ConsumedSection struct {
	Title  string
	Blocks int
	Error  string
}

// Run executes the producer or consumer usecase.
func Run(ctx context.Context, runtime Runtime, req Request) (*Result, error) {
	if req.Consume {
		result, err := consumeReports(ctx, runtime)
		if err != nil {
			return nil, err
		}
		return &Result{Consumer: result}, nil
	}

	result := produceResult(runtime)
	if err := publishProducerArtifacts(ctx, runtime, result); err != nil {
		return nil, err
	}
	return &Result{Producer: result}, nil
}

func produceResult(runtime Runtime) *ProducerResult {
	greeting := "Hello from skeleton!"
	if runtime.Config != nil && runtime.Config.Greeting != "" {
		greeting = runtime.Config.Greeting
	}
	return &ProducerResult{
		Greeting:   greeting,
		WorkDir:    runtime.WorkDir,
		ServiceDir: runtime.ServiceDir,
	}
}

func publishProducerArtifacts(ctx context.Context, runtime Runtime, result *ProducerResult) error {
	publication, err := ci.NewArtifactPublication(ci.ArtifactPublicationOptions{
		Producer: pluginName,
		Writer:   runtime.Reports,
		Results:  result,
		BuildReport: func() (*ci.Report, error) {
			run, err := plugin.NewArtifactRun(runtime.AppContext, plugin.ArtifactRunOptions{
				Producer: pluginName,
			})
			if err != nil {
				return nil, fmt.Errorf("build artifact run: %w", err)
			}
			return buildReport(result, run)
		},
	})
	if err != nil {
		return err
	}
	return ci.PublishArtifacts(ctx, publication)
}
