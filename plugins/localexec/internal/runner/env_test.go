package runner

import (
	"reflect"
	"testing"
)

func TestMergeEnvAppliesLaterValuesLast(t *testing.T) {
	t.Parallel()

	got := mergeEnv(
		map[string]string{"A": "base", "B": "base"},
		nil,
		map[string]string{"B": "override", "C": "override"},
	)
	want := map[string]string{"A": "base", "B": "override", "C": "override"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mergeEnv() = %#v, want %#v", got, want)
	}
}

func TestEnvMapToListSortsKeys(t *testing.T) {
	t.Parallel()

	got := envMapToList(map[string]string{"B": "2", "A": "1"})
	want := []string{"A=1", "B=2"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("envMapToList() = %v, want %v", got, want)
	}
}

func TestRewriteTerraciCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		command  string
		selfPath string
		want     string
	}{
		{
			name:     "empty self path leaves command unchanged",
			command:  "terraci summary",
			selfPath: "",
			want:     "terraci summary",
		},
		{
			name:     "exact terraci command",
			command:  "terraci",
			selfPath: "/tmp/terraci",
			want:     "/tmp/terraci",
		},
		{
			name:     "terraci subcommand",
			command:  "terraci summary --local",
			selfPath: "/tmp/terraci",
			want:     "/tmp/terraci summary --local",
		},
		{
			name:     "non terraci command unchanged",
			command:  "xterraci build",
			selfPath: "/tmp/terraci",
			want:     "xterraci build",
		},
		{
			name:     "terraci prefix without separator unchanged",
			command:  "terraci-summary",
			selfPath: "/tmp/terraci",
			want:     "terraci-summary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := rewriteTerraciCommand(tt.command, tt.selfPath); got != tt.want {
				t.Fatalf("rewriteTerraciCommand() = %q, want %q", got, tt.want)
			}
		})
	}
}
