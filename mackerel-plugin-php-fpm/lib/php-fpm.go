package mpphpfpm

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	mp "github.com/mackerelio/go-mackerel-plugin-helper"
)

// PhpFpmPlugin mackerel plugin
type PhpFpmPlugin struct {
	URL         string
	Prefix      string
	LabelPrefix string
	Timeout     uint
	FastCGI     bool
}

// PhpFpmStatus struct for PhpFpmPlugin mackerel plugin
type PhpFpmStatus struct {
	Pool               string `json:"pool"`
	ProcessManager     string `json:"process manager"`
	StartTime          uint64 `json:"start time"`
	StartSince         uint64 `json:"start since"`
	AcceptedConn       uint64 `json:"accepted conn"`
	ListenQueue        uint64 `json:"listen queue"`
	MaxListenQueue     uint64 `json:"max listen queue"`
	ListenQueueLen     uint64 `json:"listen queue len"`
	IdleProcesses      uint64 `json:"idle processes"`
	ActiveProcesses    uint64 `json:"active processes"`
	TotalProcesses     uint64 `json:"total processes"`
	MaxActiveProcesses uint64 `json:"max active processes"`
	MaxChildrenReached uint64 `json:"max children reached"`
	SlowRequests       uint64 `json:"slow requests"`
}

// MetricKeyPrefix interface for PluginWithPrefix
func (p PhpFpmPlugin) MetricKeyPrefix() string {
	return p.Prefix
}

// GraphDefinition interface for mackerelplugin
func (p PhpFpmPlugin) GraphDefinition() map[string]mp.Graphs {
	return map[string]mp.Graphs{
		"processes": {
			Label: p.LabelPrefix + " Processes",
			Unit:  "integer",
			Metrics: []mp.Metrics{
				{Name: "total_processes", Label: "Total Processes", Diff: false, Type: "uint64"},
				{Name: "active_processes", Label: "Active Processes", Diff: false, Type: "uint64"},
				{Name: "idle_processes", Label: "Idle Processes", Diff: false, Type: "uint64"},
			},
		},
		"max_active_processes": {
			Label: p.LabelPrefix + " Max Active Processes",
			Unit:  "integer",
			Metrics: []mp.Metrics{
				{Name: "max_active_processes", Label: "Max Active Processes", Diff: false, Type: "uint64"},
			},
		},
		"max_children_reached": {
			Label: p.LabelPrefix + " Max Children Reached",
			Unit:  "integer",
			Metrics: []mp.Metrics{
				{Name: "max_children_reached", Label: "Max Children Reached", Diff: false, Type: "uint64"},
			},
		},
		"queue": {
			Label: p.LabelPrefix + " Queue",
			Unit:  "integer",
			Metrics: []mp.Metrics{
				{Name: "listen_queue", Label: "Listen Queue", Diff: false, Type: "uint64"},
				{Name: "listen_queue_len", Label: "Listen Queue Len", Diff: false, Type: "uint64"},
			},
		},
		"max_listen_queue": {
			Label: p.LabelPrefix + " Max Listen Queue",
			Unit:  "integer",
			Metrics: []mp.Metrics{
				{Name: "max_listen_queue", Label: "Max Listen Queue", Diff: false, Type: "uint64"},
			},
		},
		"slow_requests": {
			Label: p.LabelPrefix + " Slow Requests",
			Unit:  "integer",
			Metrics: []mp.Metrics{
				{Name: "slow_requests", Label: "Slow Requests", Diff: false, Type: "uint64"},
			},
		},
	}
}

// FetchMetrics interface for mackerelplugin
func (p PhpFpmPlugin) FetchMetrics() (map[string]interface{}, error) {
	status, err := getStatus(p)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch PHP-FPM metrics: %s", err)
	}

	return map[string]interface{}{
		"total_processes":      status.TotalProcesses,
		"active_processes":     status.ActiveProcesses,
		"idle_processes":       status.IdleProcesses,
		"max_active_processes": status.MaxActiveProcesses,
		"max_children_reached": status.MaxChildrenReached,
		"listen_queue":         status.ListenQueue,
		"listen_queue_len":     status.ListenQueueLen,
		"max_listen_queue":     status.MaxListenQueue,
		"slow_requests":        status.SlowRequests,
	}, nil
}

func getStatus(p PhpFpmPlugin) (*PhpFpmStatus, error) {
	url := p.URL
	timeout := time.Duration(time.Duration(p.Timeout) * time.Second)
	client := http.Client{
		Timeout: timeout,
	}
	if p.FastCGI {
		client.Transport = &FastCGITransport{
			Timeout: timeout,
		}
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "mackerel-plugin-php-fpm")

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var status *PhpFpmStatus
	json.Unmarshal(body, &status)

	return status, nil
}

// Do the plugin
func Do() {
	optURL := flag.String("url", "http://localhost/status?json", "PHP-FPM status page URL")
	optPrefix := flag.String("metric-key-prefix", "php-fpm", "Metric key prefix")
	optLabelPrefix := flag.String("metric-label-prefix", "PHP-FPM", "Metric label prefix")
	optTimeout := flag.Uint("timeout", 5, "Timeout")
	optTempfile := flag.String("tempfile", "", "Temp file name")
	optFastCGI := flag.Bool("fcgi", false, "FastCGI mode")
	flag.Parse()

	p := PhpFpmPlugin{
		URL:         *optURL,
		Prefix:      *optPrefix,
		LabelPrefix: *optLabelPrefix,
		Timeout:     *optTimeout,
		FastCGI:     *optFastCGI,
	}
	helper := mp.NewMackerelPlugin(p)
	helper.Tempfile = *optTempfile

	helper.Run()
}
