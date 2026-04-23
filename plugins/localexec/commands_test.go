package localexec

import (
	"strings"
	"testing"

	"github.com/edelwud/terraci/pkg/plugin/plugintest"
)

func TestPlugin_Commands_Registration(t *testing.T) {
	t.Parallel()

	appCtx := plugintest.NewAppContext(t, t.TempDir())
	cmds := (&Plugin{}).Commands(appCtx)
	if len(cmds) != 1 {
		t.Fatalf("Commands() returned %d commands, want 1", len(cmds))
	}

	root := cmds[0]
	if root.Use != "local-exec" {
		t.Fatalf("command.Use = %q, want local-exec", root.Use)
	}
	if root.Example == "" {
		t.Fatal("root command example should be set")
	}
	if !strings.Contains(root.Long, `logging "no modules to process"`) {
		t.Fatalf("root command long help should describe empty target behavior:\n%s", root.Long)
	}
	if !strings.Contains(root.Long, "summary plugin produced summary-report.json") {
		t.Fatalf("root command long help should describe summary report contract:\n%s", root.Long)
	}
	for _, wanted := range []string{"local-exec plan --changed-only", "local-exec plan --filter environment=stage", "local-exec run --filter environment=stage --parallelism 2"} {
		if !strings.Contains(root.Example, wanted) {
			t.Fatalf("root command example missing %q:\n%s", wanted, root.Example)
		}
	}

	planCmd, _, err := root.Find([]string{"plan"})
	if err != nil {
		t.Fatalf("Find(plan) error = %v", err)
	}
	if planCmd == nil || planCmd.Use != "plan" {
		t.Fatalf("plan command = %#v, want plan", planCmd)
	}
	if planCmd.Example == "" {
		t.Fatal("plan command example should be set")
	}
	if !strings.Contains(planCmd.Long, `logging "no modules to process"`) {
		t.Fatalf("plan command long help should describe empty target behavior:\n%s", planCmd.Long)
	}
	if !strings.Contains(planCmd.Long, "summary-report.json") {
		t.Fatalf("plan command long help should describe summary report rendering:\n%s", planCmd.Long)
	}
	for _, wanted := range []string{"local-exec plan --changed-only", "--include 'platform/*' --exclude '*/test/*'"} {
		if !strings.Contains(planCmd.Example, wanted) {
			t.Fatalf("plan command example missing %q:\n%s", wanted, planCmd.Example)
		}
	}

	runCmd, _, err := root.Find([]string{"run"})
	if err != nil {
		t.Fatalf("Find(run) error = %v", err)
	}
	if runCmd == nil || runCmd.Use != "run" {
		t.Fatalf("run command = %#v, want run", runCmd)
	}
	if runCmd.Example == "" {
		t.Fatal("run command example should be set")
	}
	if !strings.Contains(runCmd.Long, `logging "no modules to process"`) {
		t.Fatalf("run command long help should describe empty target behavior:\n%s", runCmd.Long)
	}
	if !strings.Contains(runCmd.Long, "summary-report.json") {
		t.Fatalf("run command long help should describe summary report rendering:\n%s", runCmd.Long)
	}
	for _, wanted := range []string{"local-exec run --module platform/stage/eu-central-1/vpc", "local-exec run --filter environment=stage --parallelism 2"} {
		if !strings.Contains(runCmd.Example, wanted) {
			t.Fatalf("run command example missing %q:\n%s", wanted, runCmd.Example)
		}
	}

	applyCmd, _, err := root.Find([]string{"apply"})
	if err == nil && applyCmd != nil && applyCmd.Use == "apply" {
		t.Fatal("apply command should not be registered")
	}

	if root.PersistentFlags().Lookup("dry-run") != nil {
		t.Fatal("dry-run flag should not be registered on root command")
	}
	if planCmd.Flags().Lookup("dry-run") != nil {
		t.Fatal("dry-run flag should not be registered on plan command")
	}
	if runCmd.Flags().Lookup("dry-run") != nil {
		t.Fatal("dry-run flag should not be registered on run command")
	}
}
