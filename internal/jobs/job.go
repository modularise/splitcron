package jobs

import "github.com/modularise/modularise/cmd/config"

type Job struct {
	Name   string
	Source string
	Branch string
	Config config.Splits
}

var KnownJobs = []*Job{
	&prometheusJob,
}
