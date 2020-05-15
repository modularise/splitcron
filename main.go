package main

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/modularise/modularise/cmd"
	"github.com/modularise/modularise/cmd/config"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/yaml.v3"

	"github.com/modularise/splitcron/internal/jobs"
	"github.com/modularise/splitcron/internal/logger"
)

var (
	dryRun  bool
	pubKey  string
	verbose bool
)

func main() {
	var log *zap.Logger
	cmd := cobra.Command{
		Use: os.Args[0],
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if !dryRun && !cmd.Flags().Changed("pub-key") {
				return errors.New("--pub-key flag must be set when not in dry-run mode")
			}

			zapCore := logger.NewSplitcronEncoder()
			zapEncoder := os.Stdout
			zapLevel := zap.InfoLevel
			if verbose {
				zapLevel = zap.DebugLevel
			}
			log = zap.New(zapcore.NewCore(zapCore, zapEncoder, zapLevel))

			return cobra.NoArgs(cmd, args)
		},
		Run: func(_ *cobra.Command, _ []string) {
			runJobs(log)
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Run all jobs but do not push any content to remote repositories.")
	cmd.Flags().StringVar(&pubKey, "pub-key", "", "Path at which the private key for SSH authentication can be found to push to repositories.")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "Print out verbose output to log files for each job.")

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runJobs(log *zap.Logger) {
	log.Info("Starting jobs.", zap.Int("job-count", len(jobs.KnownJobs)))

	wc := runtime.NumCPU()
	if wc > len(jobs.KnownJobs) {
		wc = len(jobs.KnownJobs)
	}
	if verbose {
		wc = 1 // Verbose logs are useless when multiple jobs output simultaneously.
	}
	log.Info("Running jobs in parallel.", zap.Int("parallism", wc))

	wg := sync.WaitGroup{}
	jc := make(chan *jobs.Job)
	for i := 0; i < wc; i++ {
		wg.Add(1)
		go jobRunner(log, &wg, jc)
	}

	for i := range jobs.KnownJobs {
		jc <- jobs.KnownJobs[i]
	}
	close(jc)
	wg.Wait()

	log.Info("Finished running all split jobs.")
}

func jobRunner(log *zap.Logger, wg *sync.WaitGroup, jobs <-chan *jobs.Job) {
	for j := range jobs {
		runJob(log, j)
	}
	wg.Done()
}

func runJob(log *zap.Logger, job *jobs.Job) {
	log = log.With(zap.String("job", job.Name))
	log.Info("Starting job.")

	wd, err := ioutil.TempDir("", "splitcron")
	if err != nil {
		log.Error("Could not create temporary storage.", zap.Error(err))
		return
	}

	_, err = git.PlainClone(wd, false, &git.CloneOptions{
		URL:           job.Source,
		ReferenceName: plumbing.NewBranchReferenceName(job.Branch),
		SingleBranch:  true,
		Depth:         1,
	})
	if err != nil {
		log.Error("Failed to clone source repository.", zap.Error(err))
		return
	}

	c := job.Config
	if dryRun {
		// Clean out any credentials for dry-runs
		c.Credentials = config.AuthConfig{}
	} else {
		for _, s := range c.Splits {
			// We should use SSH to push to split repositories.
			s.URL = strings.Replace(s.URL, "https://github.com/", "git@github.com:", 1)
		}
		c.Credentials.PubKey = &pubKey
	}

	cb, err := yaml.Marshal(&c)
	if err != nil {
		log.Error("Failed to marshal the modularise configuration.", zap.Error(err))
		return
	}
	cf := filepath.Join(wd, "modularise.yaml")
	if err = ioutil.WriteFile(cf, cb, 0644); err != nil {
		log.Error("Failed to write modularise configuration file.", zap.String("path", cf), zap.Error(err))
		return
	}

	mc := config.CLIConfig{
		ConfigFile: cf,
		DryRun:     dryRun,
		Verbose:    verbose,
	}
	if err = mc.CheckConfig(); err != nil {
		log.Error("Failed to validate the split configuration.", zap.String("path", cf), zap.Error(err))
		return
	}

	if err = cmd.RunSplit(&mc); err != nil {
		log.Error("Job failed.", zap.Error(err))
		os.Exit(1)
	}
}
