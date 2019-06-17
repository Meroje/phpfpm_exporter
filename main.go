// Copyright 2017 Kumina, https://kumina.nl/
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	client_model "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"
	"gopkg.in/alecthomas/kingpin.v2"
)

func main() {
	var (
		listenAddress        = kingpin.Flag("web.listen-address", "Address on which to expose metrics and web interface.").Default(":9127").String()
		metricsPath          = kingpin.Flag("web.telemetry-path", "Path under which to expose metrics.").Default("/metrics").String()
		socketPaths          = kingpin.Flag("phpfpm.socket-paths", "Paths of the PHP-FPM sockets.").Strings()
		socketDirectories    = kingpin.Flag("phpfpm.socket-directories", "Path(s) of the directory where PHP-FPM sockets are located.").Strings()
		statusPath           = kingpin.Flag("phpfpm.status-path", "Path which has been configured in PHP-FPM to show status page.").Default("/status").String()
		scriptCollectorPaths = kingpin.Flag("phpfpm.script-collector-paths", "Paths of the PHP file whose output needs to be collected.").Strings()
	)

	log.AddFlags(kingpin.CommandLine)
	kingpin.Version(version.Print("phpfpm_exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	log.Infoln("Starting phpfpm_exporter exporter version: ", version.Info())
	log.Infoln("Build context", version.BuildContext())

	var sockets []string
	for _, socketDirectory := range *socketDirectories {
		filepath.Walk(socketDirectory, func(path string, info os.FileInfo, err error) error {
			if err == nil && info.Mode()&os.ModeSocket != 0 {
				sockets = append(sockets, path)
			}
			return nil
		})
	}

	for _, socket := range *socketPaths {
		sockets = append(sockets, socket)
	}

	exporter, err := NewPhpfpmExporter(sockets, *statusPath)
	if err != nil {
		log.Fatal(err)
	}
	prometheus.MustRegister(exporter)
	prometheus.MustRegister(version.NewCollector("phpfpm_exporter"))

	gatherer := prometheus.Gatherers{prometheus.DefaultGatherer}
	if len(*scriptCollectorPaths) != 0 {
		gatherer = append(gatherer,
			prometheus.GathererFunc(func() ([]*client_model.MetricFamily, error) {
				return CollectMetricsFromScript(sockets, *scriptCollectorPaths)
			}),
		)
	}

	log.Infoln("Listening on", *listenAddress)
	http.Handle(*metricsPath, promhttp.HandlerFor(gatherer, promhttp.HandlerOpts{ErrorHandling: promhttp.ContinueOnError}))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
			<head><title>PHP-FPM Exporter</title></head>
			<body>
			<h1>PHP-FPM Exporter</h1>
			<p><a href="` + *metricsPath + `">Metrics</a></p>
			</body>
			</html>`))
	})
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
