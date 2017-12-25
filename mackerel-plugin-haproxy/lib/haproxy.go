package mphaproxy

import (
	"bufio"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"time"

	mp "github.com/mackerelio/go-mackerel-plugin"
)

var graphdef = map[string]mp.Graphs{
	"haproxy.total.sessions": {
		Label: "HAProxy Total Sessions",
		Unit:  "integer",
		Metrics: []mp.Metrics{
			{Name: "sessions", Label: "Sessions", Diff: true},
		},
	},
	"haproxy.total.bytes": {
		Label: "HAProxy Total Bytes",
		Unit:  "integer",
		Metrics: []mp.Metrics{
			{Name: "bytes_in", Label: "Bytes In", Diff: true},
			{Name: "bytes_out", Label: "Bytes Out", Diff: true},
		},
	},
	"haproxy.total.connection_errors": {
		Label: "HAProxy Total Connection Errors",
		Unit:  "integer",
		Metrics: []mp.Metrics{
			{Name: "connection_errors", Label: "Connection Errors", Diff: true},
		},
	},
}

// HAProxyPlugin mackerel plugin for haproxy
type HAProxyPlugin struct {
	URI      string
	Username string
	Password string
	Socket   string
}

// FetchMetrics interface for mackerelplugin
func (p HAProxyPlugin) FetchMetrics() (map[string]float64, error) {
	if p.Socket == "" {
		return p.fetchMetricsFromTCP()
	} else {
		return p.fetchMetricsFromSocket()
	}
}

func (p HAProxyPlugin) fetchMetricsFromTCP() (map[string]float64, error) {
	client := &http.Client{
		Timeout: time.Duration(5) * time.Second,
	}

	requestURI := p.URI + ";csv;norefresh"
	req, err := http.NewRequest("GET", requestURI, nil)
	if err != nil {
		return nil, err
	}
	if p.Username != "" {
		req.SetBasicAuth(p.Username, p.Password)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Request failed. Status: %s, URI: %s", resp.Status, requestURI)
	}

	return p.parseStats(resp.Body)
}

func (p HAProxyPlugin) fetchMetricsFromSocket() (map[string]float64, error) {
	client, err := net.Dial("unix", p.Socket)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	fmt.Fprintln(client, "show stat")

	return p.parseStats(bufio.NewReader(client))
}

func (p HAProxyPlugin) parseStats(statsBody io.Reader) (map[string]float64, error) {
	stat := make(map[string]float64)
	reader := csv.NewReader(statsBody)

	for {
		columns, err := reader.Read()
		if err == io.EOF {
			break
		}

		if len(columns) < 60 {
			return nil, errors.New("length of stats csv is too short (specified uri/socket may be wrong)")
		}

		if columns[1] != "BACKEND" {
			continue
		}

		var data float64

		data, err = strconv.ParseFloat(columns[7], 64)
		if err != nil {
			return nil, errors.New("cannot get values")
		}
		stat["sessions"] += data

		data, err = strconv.ParseFloat(columns[8], 64)
		if err != nil {
			return nil, errors.New("cannot get values")
		}
		stat["bytes_in"] += data

		data, err = strconv.ParseFloat(columns[9], 64)
		if err != nil {
			return nil, errors.New("cannot get values")
		}
		stat["bytes_out"] += data

		data, err = strconv.ParseFloat(columns[13], 64)
		if err != nil {
			return nil, errors.New("cannot get values")
		}
		stat["connection_errors"] += data
	}

	return stat, nil
}

// GraphDefinition interface for mackerelplugin
func (p HAProxyPlugin) GraphDefinition() map[string]mp.Graphs {
	return graphdef
}

// Do the plugin
func Do() {
	optURI := flag.String("uri", "", "URI")
	optScheme := flag.String("scheme", "http", "Scheme")
	optHost := flag.String("host", "localhost", "Hostname")
	optPort := flag.String("port", "80", "Port")
	optPath := flag.String("path", "/", "Path")
	optUsername := flag.String("username", "", "Username for Basic Auth")
	optPassword := flag.String("password", "", "Password for Basic Auth")
	optTempfile := flag.String("tempfile", "", "Temp file name")
	optSocket := flag.String("socket", "", "Unix Domain Socket")
	flag.Parse()

	var haproxy HAProxyPlugin
	if *optURI != "" {
		haproxy.URI = *optURI
	} else {
		haproxy.URI = fmt.Sprintf("%s://%s:%s%s", *optScheme, *optHost, *optPort, *optPath)
	}

	if *optUsername != "" {
		haproxy.Username = *optUsername
	}

	if *optPassword != "" {
		haproxy.Password = *optPassword
	}

	if *optSocket != "" {
		haproxy.Socket = *optSocket
	}

	helper := mp.NewMackerelPlugin(haproxy)
	helper.Tempfile = *optTempfile

	helper.Run()
}
