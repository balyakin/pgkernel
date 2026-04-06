package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/balyakin/pgkernel/internal/checker"
	"github.com/balyakin/pgkernel/internal/checks"
	"github.com/balyakin/pgkernel/internal/detect"
	"github.com/balyakin/pgkernel/internal/output"
	"github.com/balyakin/pgkernel/internal/policy"
	"github.com/spf13/cobra"
)

type checkOptions struct {
	format      string
	pgConfig    string
	severity    string
	only        string
	exclude     string
	failOn      string
	baseline    string
	compareWith string
	profile     string
	share       bool
	noColor     bool
	quiet       bool
}

func main() {
	if err := buildRootCommand().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(3)
	}
}

func buildRootCommand() *cobra.Command {
	opts := checkOptions{}
	rootCmd := &cobra.Command{
		Use:   "pgkernel",
		Short: "PostgreSQL and Linux kernel health checker",
		Long:  "One command. Full diagnosis. Zero guesswork.",
	}

	checkCmd := &cobra.Command{
		Use:   "check",
		Short: "Run kernel + PostgreSQL checks",
		Run: func(cmd *cobra.Command, args []string) {
			exitCode := runCheck(opts)
			os.Exit(exitCode)
		},
	}

	checkCmd.Flags().StringVar(&opts.format, "format", "pretty", "Output format: pretty, json, markdown")
	checkCmd.Flags().StringVar(&opts.pgConfig, "pg-config", "", "Path to postgresql.conf (auto-detect when empty)")
	checkCmd.Flags().StringVar(&opts.severity, "severity", "all", "Minimum severity to show: all, warn, crit")
	checkCmd.Flags().StringVar(&opts.only, "only", "", "Run only selected checks or categories")
	checkCmd.Flags().StringVar(&opts.exclude, "exclude", "", "Exclude checks or categories")
	checkCmd.Flags().StringVar(&opts.failOn, "fail-on", "crit", "CI fail threshold: warn or crit")
	checkCmd.Flags().StringVar(&opts.baseline, "baseline", "", "Path to baseline JSON report")
	checkCmd.Flags().StringVar(&opts.compareWith, "compare-with", "", "Path to previous JSON report for regression detection")
	checkCmd.Flags().StringVar(&opts.profile, "profile", "auto", "Runtime profile: auto, baremetal, vm, container, managed")
	checkCmd.Flags().BoolVar(&opts.share, "share", false, "Print share-ready markdown snippet with top risks and fixes")
	checkCmd.Flags().BoolVar(&opts.noColor, "no-color", false, "Disable colored output")
	checkCmd.Flags().BoolVar(&opts.quiet, "quiet", false, "Exit code only mode")

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print pgkernel version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("pgkernel %s\n", checker.ToolVersion)
		},
	}

	rootCmd.AddCommand(checkCmd, versionCmd)
	return rootCmd
}

func runCheck(opts checkOptions) int {
	if err := validateOptions(opts); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 3
	}

	state, systemInfo, postgresInfo := detect.CollectState(detect.CollectOptions{
		Profile:      opts.profile,
		PGConfigPath: opts.pgConfig,
	})

	enabledChecks := policy.ApplyCheckFilter(checks.All(), opts.only, opts.exclude)
	runner := checker.NewRunner(enabledChecks)
	results := runner.Run(state)
	runtimeError := runner.HasRuntimeError()

	report := checker.NewReport(state.Profile, systemInfo, postgresInfo, results)

	regressionsOnly := opts.compareWith != "" || opts.baseline != ""
	if opts.baseline != "" {
		baselineReport, err := policy.LoadReport(opts.baseline)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to load baseline report: %v\n", err)
			return 3
		}
		report.Regressions = append(report.Regressions, policy.DetectRegressions(results, baselineReport)...)
	}
	if opts.compareWith != "" {
		comparisonReport, err := policy.LoadReport(opts.compareWith)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to load comparison report: %v\n", err)
			return 3
		}
		report.Regressions = append(report.Regressions, policy.DetectRegressions(results, comparisonReport)...)
	}
	report.Regressions = dedupeRegressions(report.Regressions)

	exitCode := policy.DetermineExitCode(results, report.Regressions, opts.failOn, runtimeError, regressionsOnly)
	report.Summary = checker.BuildSummary(results)
	report.Summary.ExitCode = exitCode

	if opts.quiet {
		if exitCode != 0 {
			fmt.Fprintf(os.Stderr, "pgkernel: exit=%d warnings=%d criticals=%d regressions=%d\n", exitCode, report.Summary.Warnings, report.Summary.Criticals, len(report.Regressions))
		}
		return exitCode
	}

	rendered, err := output.Render(opts.format, report, output.RenderOptions{
		NoColor:        opts.noColor,
		SeverityFilter: opts.severity,
		Share:          opts.share,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 3
	}

	fmt.Println(rendered)
	return exitCode
}

func validateOptions(opts checkOptions) error {
	if opts.failOn != "warn" && opts.failOn != "crit" {
		return errors.New("invalid --fail-on value: use warn or crit")
	}
	if opts.severity != "all" && opts.severity != "warn" && opts.severity != "crit" {
		return errors.New("invalid --severity value: use all, warn, or crit")
	}
	format := strings.ToLower(opts.format)
	if format != "pretty" && format != "json" && format != "markdown" {
		return errors.New("invalid --format value: use pretty, json, or markdown")
	}
	return nil
}

func dedupeRegressions(items []checker.Regression) []checker.Regression {
	if len(items) <= 1 {
		return items
	}
	seen := make(map[string]struct{}, len(items))
	unique := make([]checker.Regression, 0, len(items))
	for _, item := range items {
		key := fmt.Sprintf("%s:%s:%s", item.ID, item.Previous, item.Current)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, item)
	}
	return unique
}
