// Copyright Aeraki Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dubbo

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aeraki-framework/aeraki/test/e2e/util"
	"istio.io/pkg/log"
)

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	shutdown()
	os.Exit(code)
}

func setup() {
	util.CreateNamespace("dubbo", "")
	util.LabelNamespace("dubbo", "istio-injection=enabled", "")
	util.KubeApply("dubbo", "testdata/dubbo-sample.yaml", "")
	util.KubeApply("dubbo", "testdata/serviceentry.yaml", "")
	util.KubeApply("dubbo", "testdata/destinationrule.yaml", "")
}

func shutdown() {
	// leave the created application for debugging purpose
	/*util.KubeDelete("dubbo", "testdata/dubbo-sample.yaml", "")
	util.KubeDelete("dubbo", "testdata/serviceentry.yaml", "")
	util.KubeDelete("dubbo", "testdata/destinationrule.yaml", "")
	util.DeleteNamespace("dubbo", "")*/
}

func TestSidecarOutboundConfig(t *testing.T) {
	util.WaitForDeploymentsReady("dubbo", 10*time.Minute, "")
	consumerPod, _ := util.GetPodName("dubbo", "app=dubbo-sample-consumer", "")
	config, _ := util.PodExec("dubbo", consumerPod, "istio-proxy", "curl -s 127.0.0.1:15000/config_dump", false, "")
	config = strings.Join(strings.Fields(config), "")
	want := "{\n\"name\":\"envoy.filters.network.dubbo_proxy\",\n\"typed_config\":{\n\"@type\":\"type.googleapis.com/envoy.extensions.filters.network.dubbo_proxy.v3.DubboProxy\",\n\"stat_prefix\":\"outbound|20880||org.apache.dubbo.samples.basic.api.demoservice\",\n\"route_config\":[\n{\n\"name\":\"outbound|20880||org.apache.dubbo.samples.basic.api.demoservice\",\n\"interface\":\"org.apache.dubbo.samples.basic.api.DemoService\",\n\"routes\":[\n{\n\"match\":{\n\"method\":{\n\"name\":{\n\"safe_regex\":{\n\"google_re2\":{},\n\"regex\":\".*\"\n}\n}\n}\n},\n\"route\":{\n\"cluster\":\"outbound|20880||org.apache.dubbo.samples.basic.api.demoservice\"\n}\n}\n]\n}\n]\n}\n}\n]\n}"
	want = strings.Join(strings.Fields(want), "")
	log.Info(config)
	if !strings.Contains(config, want) {
		t.Error("cant't find dubbo proxy in the outbound listener of the envoy sidecar")
	}
}

func TestVersionRouting(t *testing.T) {
	util.WaitForDeploymentsReady("dubbo", 10*time.Minute, "")
	testVersion("v1", t)
	testVersion("v2", t)
}

func testVersion(version string, t *testing.T) {
	util.KubeApply("dubbo", "testdata/virtualservice-"+version+".yaml", "")
	defer util.KubeDelete("dubbo", "testdata/virtualservice-"+version+".yaml", "")

	log.Info("Waiting for rules to propagate ...")
	time.Sleep(1 * time.Minute)
	consumerPod, _ := util.GetPodName("dubbo", "app=dubbo-sample-consumer", "")
	for i := 0; i < 5; i++ {
		dubboResponse, _ := util.PodExec("dubbo", consumerPod, "dubbo-sample-consumer", "curl -s 127.0.0.1:9009/hello", false, "")
		want := "response from dubbo-sample-provider-" + version
		log.Info(dubboResponse)
		if !strings.Contains(dubboResponse, want) {
			t.Errorf("Version routing failed, want: %s, got %s", want, dubboResponse)
		}
	}
}

func TestPercentageRouting(t *testing.T) {
	util.WaitForDeploymentsReady("dubbo", 10*time.Minute, "")
	util.KubeApply("dubbo", "testdata/virtualservice-traffic-splitting.yaml", "")
	defer util.KubeDelete("dubbo", "testdata/virtualservice-traffic-splitting.yaml", "")

	log.Info("Waiting for rules to propagate ...")
	time.Sleep(1 * time.Minute)
	consumerPod, _ := util.GetPodName("dubbo", "app=dubbo-sample-consumer", "")
	v1 := 0
	for i := 0; i < 20; i++ {
		dubboResponse, _ := util.PodExec("dubbo", consumerPod, "dubbo-sample-consumer", "curl -s 127.0.0.1:9009/hello", false, "")
		responseV1 := "response from dubbo-sample-provider-v1"
		log.Info(dubboResponse)
		if strings.Contains(dubboResponse, responseV1) {
			v1++
		}
	}
	// The most accurate number should be 6, but the number may fall into a range around 6 since the sample is not big enough
	if v1 > 8 || v1 < 4 {
		t.Errorf("percentage traffic routing failed, want: %s got:%v ", "between 4 and 8", v1)
	}
}

func TestMethodRouting(t *testing.T) {
	util.WaitForDeploymentsReady("dubbo", 10*time.Minute, "")
	testMethodMatch("exact", t)
	testMethodMatch("prefix", t)
	testMethodMatch("regex", t)
}

func testMethodMatch(matchPattern string, t *testing.T) {
	util.KubeApply("dubbo", "testdata/virtualservice-method-"+matchPattern+".yaml", "")
	log.Info("Waiting for rules to propagate ...")
	time.Sleep(1*time.Minute + 30*time.Second)
	consumerPod, _ := util.GetPodName("dubbo", "app=dubbo-sample-consumer", "")
	for i := 0; i < 5; i++ {
		time.Sleep(3 * time.Second)
		dubboResponse, _ := util.PodExec("dubbo", consumerPod, "dubbo-sample-consumer", "curl -s 127.0.0.1:9009/hello", false, "")
		want := "response from dubbo-sample-provider-v2"
		log.Info(dubboResponse)
		if !strings.Contains(dubboResponse, want) {
			t.Errorf("method routing failed, want: %s, got %s", want, dubboResponse)
		}
	}
}
