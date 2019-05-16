// Copyright (c) 2019 The OpenSDS Authors.
//
//    Licensed under the Apache License, Version 2.0 (the "License"); you may
//    not use this file except in compliance with the License. You may obtain
//    a copy of the License at
//
//         http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
//    WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
//    License for the specific language governing permissions and limitations
//    under the License.
package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"sync"

	log "github.com/golang/glog"
	"github.com/opensds/opensds/contrib/drivers/lvm"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/yaml.v2"
)

const DefaultConfigFile = "/etc/opensds/metrics/lvm.yaml"
const DefaultPort = "8080"

// struct for lvm  collector that contains pointers
// to prometheus descriptors for each metric we expose.
type lvmCollector struct {
	mu sync.Mutex
	//volume metrics
	VolumeIOPS            *prometheus.Desc
	VolumeReadThroughput  *prometheus.Desc
	VolumeWriteThroughput *prometheus.Desc
	VolumeResponseTime    *prometheus.Desc
	VolumeServiceTime     *prometheus.Desc
	VolumeUtilization     *prometheus.Desc
	//Disk metrics
	DiskIOPS            *prometheus.Desc
	DiskReadThroughput  *prometheus.Desc
	DiskWriteThroughput *prometheus.Desc
	DiskResponseTime    *prometheus.Desc
	DiskServiceTime     *prometheus.Desc
	DiskUtilization     *prometheus.Desc
}

// constructor for lvm collector that
// initializes every descriptor and returns a pointer to the collector
func newLvmCollector() *lvmCollector {
	var labelKeys = []string{"device"}

	return &lvmCollector{
		VolumeIOPS: prometheus.NewDesc("lvm_volume_iops_tps",
			"Shows IOPS",
			labelKeys, nil,
		),
		VolumeReadThroughput: prometheus.NewDesc("lvm_volume_readThroughput_kbs",
			"Shows ReadThroughput",
			labelKeys, nil,
		),
		VolumeWriteThroughput: prometheus.NewDesc("lvm_volume_writeThroughput_kbs",
			"Shows ReadThroughput",
			labelKeys, nil,
		),
		VolumeResponseTime: prometheus.NewDesc("lvm_volume_responseTime_ms",
			"Shows ReadThroughput",
			labelKeys, nil,
		),
		VolumeServiceTime: prometheus.NewDesc("lvm_volume_serviceTime_ms",
			"Shows ServiceTime",
			labelKeys, nil,
		),
		VolumeUtilization: prometheus.NewDesc("lvm_volume_utilization_prcnt",
			"Shows Utilization in percentage",
			labelKeys, nil,
		),
		DiskIOPS: prometheus.NewDesc("lvm_volume_iops_tps",
			"Shows IOPS",
			labelKeys, nil,
		),
		DiskReadThroughput: prometheus.NewDesc("lvm_disk_readThroughput_kbs",
			"Shows ReadThroughput",
			labelKeys, nil,
		),
		DiskWriteThroughput: prometheus.NewDesc("lvm_disk_writeThroughput_kbs",
			"Shows ReadThroughput",
			labelKeys, nil,
		),
		DiskResponseTime: prometheus.NewDesc("lvm_disk_responseTime_ms",
			"Shows ReadThroughput",
			labelKeys, nil,
		),
		DiskServiceTime: prometheus.NewDesc("lvm_disk_serviceTime_ms",
			"Shows ServiceTime",
			labelKeys, nil,
		),
		DiskUtilization: prometheus.NewDesc("lvm_disk_utilization_prcnt",
			"Shows Utilization in percentage",
			labelKeys, nil,
		),
	}

}

// Describe function.
// It essentially writes all descriptors to the prometheus desc channel.
func (c *lvmCollector) Describe(ch chan<- *prometheus.Desc) {

	//Update this section with the each metric you create for a given collector
	ch <- c.VolumeIOPS
	ch <- c.VolumeReadThroughput
	ch <- c.VolumeWriteThroughput
	ch <- c.VolumeResponseTime
	ch <- c.VolumeServiceTime
	ch <- c.VolumeUtilization
	ch <- c.DiskIOPS
	ch <- c.DiskReadThroughput
	ch <- c.DiskWriteThroughput
	ch <- c.DiskResponseTime
	ch <- c.DiskServiceTime
	ch <- c.DiskUtilization
}

type Config struct {
	Type    string   `type`
	Devices []string `devices`
}

type Configs struct {
	Cfgs []*Config `resources`
}

// Collect implements required collect function for all promehteus collectors
func (c *lvmCollector) Collect(ch chan<- prometheus.Metric) {

	c.mu.Lock()
	defer c.mu.Unlock()

	//Implement logic here to determine proper metric value to return to prometheus
	//for each descriptor
	metricList := []string{"iops", "readThroughput", "writeThroughput", "responseTime", "serviceTime", "utilizationprcnt"}
	source, err := ioutil.ReadFile(DefaultConfigFile)
	if err != nil {
		log.Fatal("file "+DefaultConfigFile+"can't read", err)
	}
	var config Configs
	err1 := yaml.Unmarshal(source, &config)
	if err1 != nil {
		log.Fatalf("error: %v", err)
	}

	metricDriver := lvm.MetricDriver{}
	metricDriver.Setup()
	for _, resource := range config.Cfgs {
		switch resource.Type {
		case "volume":
			for _, volume := range resource.Devices {
				metricArray, _ := metricDriver.CollectMetrics(metricList, volume)
				for _, metric := range metricArray {
					lableVals := []string{metric.InstanceName}
					switch metric.Name {
					case "iops":
						ch <- prometheus.MustNewConstMetric(c.VolumeIOPS, prometheus.GaugeValue, metric.MetricValues[0].Value, lableVals...)
					case "readThroughput":
						ch <- prometheus.MustNewConstMetric(c.VolumeReadThroughput, prometheus.GaugeValue, metric.MetricValues[0].Value, lableVals...)
					case "writeThroughput":
						ch <- prometheus.MustNewConstMetric(c.VolumeWriteThroughput, prometheus.GaugeValue, metric.MetricValues[0].Value, lableVals...)
					case "responseTime":
						ch <- prometheus.MustNewConstMetric(c.VolumeResponseTime, prometheus.GaugeValue, metric.MetricValues[0].Value, lableVals...)
					case "serviceTime":
						ch <- prometheus.MustNewConstMetric(c.VolumeServiceTime, prometheus.GaugeValue, metric.MetricValues[0].Value, lableVals...)

					case "utilizationprcnt":
						ch <- prometheus.MustNewConstMetric(c.VolumeUtilization, prometheus.GaugeValue, metric.MetricValues[0].Value, lableVals...)

					}
				}
			}

		case "disk":
			for _, volume := range resource.Devices {
				metricArray, _ := metricDriver.CollectMetrics(metricList, volume)
				for _, metric := range metricArray {
					lableVals := []string{metric.Labels["device"]}
					switch metric.Name {
					case "iops":
						ch <- prometheus.MustNewConstMetric(c.DiskIOPS, prometheus.GaugeValue, metric.MetricValues[0].Value, lableVals...)
					case "readThroughput":
						ch <- prometheus.MustNewConstMetric(c.DiskReadThroughput, prometheus.GaugeValue, metric.MetricValues[0].Value, lableVals...)
					case "writeThroughput":
						ch <- prometheus.MustNewConstMetric(c.DiskWriteThroughput, prometheus.GaugeValue, metric.MetricValues[0].Value, lableVals...)
					case "responseTime":
						ch <- prometheus.MustNewConstMetric(c.DiskResponseTime, prometheus.GaugeValue, metric.MetricValues[0].Value, lableVals...)
					case "serviceTime":
						ch <- prometheus.MustNewConstMetric(c.DiskServiceTime, prometheus.GaugeValue, metric.MetricValues[0].Value, lableVals...)
					case "utilizationprcnt":
						ch <- prometheus.MustNewConstMetric(c.DiskUtilization, prometheus.GaugeValue, metric.MetricValues[0].Value, lableVals...)

					}
				}
			}
		}

	}

}
func validateCliArg(arg1 string) string {
	num, err := strconv.Atoi(arg1)
	if (err != nil) || (num > 65535) {

		fmt.Println("please enter a valid port number")
		os.Exit(1)
	}
	return arg1
}

// main function for lvm exporter
// lvm exporter is a independent process which user can start if required
func main() {

	var portNo string
	if len(os.Args) > 1 {
		portNo = validateCliArg(os.Args[1])
	} else {
		portNo = DefaultPort
	}

	//Create a new instance of the lvmcollector and
	//register it with the prometheus client.
	lvm := newLvmCollector()
	prometheus.MustRegister(lvm)

	//This section will start the HTTP server and expose
	//any metrics on the /metrics endpoint.
	http.Handle("/metrics", promhttp.Handler())
	log.Info("lvm exporter begining to serve on port :" + portNo)
	log.Fatal(http.ListenAndServe(":"+portNo, nil))
}
