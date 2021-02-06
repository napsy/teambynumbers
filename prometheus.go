package main

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	promLeadTime = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "teambynumbers_lead_time",
		Help: "Number of done items in a week",
	},
		[]string{"team"})
	promCycleTime = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "teambynumbers_cycle_time",
		Help: "Time in days for a task to be done",
	},
		[]string{"team"})
	promBugsReported = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "teambynumbers_bugs_reported",
		Help: "Number of bugs reported",
	},
		[]string{"team"})
	promBugsSquased = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "teambynumbers_bugs_squashed",
		Help: "Number of bugs solved",
	},
		[]string{"team"})
	promDeployCount = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "teambynumbers_deploy_count",
		Help: "Number of deploys within a week",
	},
		[]string{"team"})
	promMemberCount = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "teambynumbers_member_count",
		Help: "Team member count",
	},
		[]string{"team"})
)

// updatePrometheusStatus will set the gauges to the
// latest reported data.
func updatePrometheusStatus(db *recordDb) {
	teams := make(map[string]bool)
	// Records are already sorted from newest to oldest, so we
	// only have to remember which team we already processed.
	for i := range db.records {
		record := db.records[i]
		if _, ok := teams[db.records[i].Team]; ok {
			continue
		}
		promLeadTime.WithLabelValues(record.Team).Set(record.LeadTime)
		promCycleTime.WithLabelValues(record.Team).Set(record.CycleTime)
		promMemberCount.WithLabelValues(record.Team).Set(float64(record.MemberCount))
		promBugsReported.WithLabelValues(record.Team).Set(float64(record.BugsReported))
		promBugsSquased.WithLabelValues(record.Team).Set(float64(record.BugsSquashed))
		promDeployCount.WithLabelValues(record.Team).Set(float64(record.DeployCount))
		teams[record.Team] = true
	}
}

func prometheusUpdater(db *recordDb, interval time.Duration) {
	go func() {
		for {
			updatePrometheusStatus(db)
			time.Sleep(interval)
		}
	}()
}
