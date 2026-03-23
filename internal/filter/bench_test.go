package filter

import (
	"fmt"
	"testing"

	"github.com/edelwud/terraci/internal/discovery"
)

func BenchmarkMatchGlob_Simple(b *testing.B) {
	paths := []string{
		"platform/prod/us-east-1/vpc",
		"platform/stage/eu-west-1/eks",
		"payments/prod/ap-south-1/rds",
		"auth/dev/us-west-2/lambda",
	}
	pattern := "platform/*/*/vpc"

	b.ResetTimer()
	for b.Loop() {
		for _, p := range paths {
			matchGlob(pattern, p)
		}
	}
}

func BenchmarkMatchGlob_DoubleStar(b *testing.B) {
	paths := []string{
		"platform/prod/us-east-1/vpc",
		"platform/stage/eu-west-1/eks",
		"payments/prod/ap-south-1/vpc",
		"auth/dev/us-west-2/lambda",
	}
	pattern := "**/vpc"

	b.ResetTimer()
	for b.Loop() {
		for _, p := range paths {
			matchGlob(pattern, p)
		}
	}
}

func BenchmarkApply(b *testing.B) {
	modules := make([]*discovery.Module, 100)
	services := []string{"platform", "payments", "auth", "infra", "data"}
	envs := []string{"prod", "stage", "dev", "sandbox"}
	regions := []string{"us-east-1", "eu-west-1", "ap-south-1", "us-west-2"}
	names := []string{"vpc", "eks", "rds", "lambda", "s3"}

	for i := range 100 {
		modules[i] = discovery.TestModule(
			services[i%len(services)],
			envs[i%len(envs)],
			regions[i%len(regions)],
			names[i%len(names)],
		)
	}

	opts := Options{
		Excludes: []string{"**/sandbox/**"},
		Includes: []string{},
		Segments: map[string][]string{
			"service": {"platform", "payments"},
		},
	}

	b.ResetTimer()
	for b.Loop() {
		Apply(modules, opts)
	}
}

func BenchmarkApply_LargeModuleSet(b *testing.B) {
	for _, size := range []int{100, 500, 1000} {
		modules := make([]*discovery.Module, size)
		for i := range size {
			modules[i] = discovery.TestModule(
				fmt.Sprintf("svc%d", i%10),
				fmt.Sprintf("env%d", i%4),
				fmt.Sprintf("region%d", i%3),
				fmt.Sprintf("mod%d", i),
			)
		}

		opts := Options{
			Excludes: []string{"**/mod0"},
			Segments: map[string][]string{
				"service": {"svc0", "svc1", "svc2"},
			},
		}

		b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
			for b.Loop() {
				Apply(modules, opts)
			}
		})
	}
}
