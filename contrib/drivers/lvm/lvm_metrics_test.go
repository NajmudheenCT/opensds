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
package lvm

import (
	"fmt"
	"github.com/opensds/opensds/pkg/model"
	"github.com/opensds/opensds/pkg/utils/config"
	"github.com/opensds/opensds/pkg/utils/exec"
	"reflect"
	"testing"
)

func TestMetricDriverSetup(t *testing.T) {
	var d = &MetricDriver{}

	if err := d.Setup(); err != nil {
		t.Errorf("Setup lvm metric  driver failed: %+v\n", err)
	}

}

func TestCollectMetrics(t *testing.T) {

	metricList := []string{"IOPS", "ReadThroughput", "WriteThroughput", "ResponseTime", "ServiceTime", "UtilizationPercentage"}
	var metricDriver = &MetricDriver{}
	metricDriver.Setup()
	metricArray, err := metricDriver.CollectMetrics(metricList, "sda")
	if err != nil {
		t.Errorf("collectMetrics call to lvm driver failed: %+v\n", err)
	}
	for _, m := range metricArray {
		t.Log(*m)
	}

}
type MetricFakeExecuter struct {
	RespMap map[string]*MetricFakeResp
}

type MetricFakeResp struct {
	out string
	err error
}
func (f *MetricFakeExecuter) Run(name string, args ...string) (string, error) {
	var cmd = name
	if name == "env" {
		cmd = args[1]
	}
	v, ok := f.RespMap[cmd]
	if !ok {
		return "", fmt.Errorf("can find specified op: %s", args[1])
	}
	return v.out, v.err
}
func NewMetricFakeExecuter(respMap map[string]*MetricFakeResp) exec.Executer {
	return &MetricFakeExecuter{RespMap: respMap}
}

func TestCollectMetrics1(t *testing.T) {
	var fd = &MetricDriver{}
	config.CONF.OsdsDock.Backends.LVM.ConfigPath = "testdata/lvm.yaml"
	fd.Setup()

	respMap := map[string]*MetricFakeResp{
		"sar": { `05:26:43  IST       DEV       tps     rkB/s     wkB/s   areq-sz    aqu-sz     await     svctm     %util
			05:26:44      loop0      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00
			05:26:44      loop1      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00
			05:26:44      loop2      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00
			05:26:44      loop3      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00
			05:26:44      loop4      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00
			05:26:44      loop5      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00
			05:26:44      loop6      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00
			05:26:44      loop7      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00
			05:26:44        sda      3.16      0.00    134.74     42.67      0.01      2.67      4.00      1.26
			05:26:44      loop8      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00
			05:26:44      loop9      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00
			05:26:44      loop10      0.00      0.00      0.00      0.00      0.00      0.00      0.00      0.00`, nil},
	}
	fd.cli.RootExecuter = NewMetricFakeExecuter(respMap)
	fd.cli.BaseExecuter = NewMetricFakeExecuter(respMap)
	metricMap:= map[string]float64{"IOPS":3.16,"ReadThroughput":0.00,"WriteThroughput":134.74, "ResponseTime":2.67, "ServiceTime":4.00, "UtilizationPercentage":1.26}
	metricToUnitMap:= map[string]string{"IOPS":"tps","ReadThroughput":"KB/s","WriteThroughput":"KB/s", "ResponseTime":"ms", "ServiceTime":"ms", "UtilizationPercentage":"%"}
	var tempMetricArray []*model.MetricSpec
	metricsList := []string{"IOPS", "ReadThroughput", "WriteThroughput", "ResponseTime", "ServiceTime", "UtilizationPercentage"}
	for _, element := range metricsList {
		val:= metricMap[element]
		//Todo: See if association  is required here, resource discovery could fill this information
		associatorMap := make(map[string]string)
		associatorMap["device"] = "sda"
		metricValue := &model.Metric{
			Timestamp: 123456,
			Value:     val,
		}
		metricValues := make([]*model.Metric, 0)
		metricValues = append(metricValues, metricValue)

		metric := &model.MetricSpec{
			InstanceID:   "sda",
			InstanceName: "sda",
			Job:          "OpenSDS",
			Labels:       associatorMap,
			//Todo Take Componet from Post call, as of now it is only for volume
			Component: "Volume",
			Name:      element,
			//Todo : Fill units according to metric type
			Unit: metricToUnitMap[element],
			//Todo : Get this information dynamically ( hard coded now , as all are direct values
			AggrType:     "",
			MetricValues: metricValues,
		}
		tempMetricArray = append(tempMetricArray, metric)
	}
	expectedMetrics:=tempMetricArray
	retunMetrics, err := fd.CollectMetrics(metricsList,"sda")
	if err != nil {
		t.Error("Failed to create volume:", err)
	}

	if !reflect.DeepEqual(retunMetrics, expectedMetrics) {
		t.Errorf("Expected %+v, got %+v\n", expectedMetrics, retunMetrics)
		printMetricSpec(expectedMetrics)
		fmt.Println("returned metrics")
		printMetricSpec(retunMetrics)
	}
}

func printMetricSpec(m []*model.MetricSpec){
	for _,p := range m{
		fmt.Printf("%+v\n", p)
		for _,v := range p.MetricValues{
			fmt.Printf("%+v\n", v)
		}
	}


}