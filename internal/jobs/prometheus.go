package jobs

import "github.com/modularise/modularise/cmd/config"

var prometheusJob = Job{
	Name:   "prometheus",
	Source: "https://github.com/prometheus/prometheus",
	Branch: "master",
	Config: config.Splits{
		Splits: map[string]*config.Split{
			"tsdb": {
				ModulePath: "github.com/modularise/prometheus-tsdb",
				URL:        "https://github.com/modularise/prometheus-tsdb",
				Branch:     "master",
				Includes: []string{
					"pkg/labels",
					"storage",
					"tsdb",
				},
				Excludes: []string{
					"storage/fanout",
					"storage/remote",
				},
			},
			"promql": {
				ModulePath: "github.com/modularise/prometheus-promql",
				URL:        "https://github.com/modularise/prometheus-promql",
				Branch:     "master",
				Includes: []string{
					"promql",
					"util/stats",
				},
			},
			"discovery": {
				ModulePath: "github.com/modularise/prometheus-discovery",
				URL:        "https://github.com/modularise/prometheus-discovery",
				Branch:     "master",
				Includes: []string{
					"discovery",
					"util/treecache",
				},
			},
		},
	},
}
