package envoyfilter

import (
	"bytes"
	"strconv"

	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/types"
	networking "istio.io/api/networking/v1alpha3"
	"istio.io/istio/pkg/config/host"
)

type Generator interface {
	Generate(service *networking.ServiceEntry) *networking.EnvoyFilter
}

type DubboGenerator struct {
}

func NewDubboGenerator() *DubboGenerator {
	return &DubboGenerator{}
}

func (*DubboGenerator) Generate(service *networking.ServiceEntry) *networking.EnvoyFilter {
	listenerName := service.GetAddresses()[0] + "_" + strconv.Itoa(int(service.Ports[0].Number))
	dubboProxy := buildDubboProxy(host.Name(service.Hosts[0]), int(service.Ports[0].Number))

	buf := &bytes.Buffer{}
	_ = (&jsonpb.Marshaler{OrigName: true}).Marshal(buf, dubboProxy)
	var out = &types.Struct{}
	(&jsonpb.Unmarshaler{AllowUnknownFields: false}).Unmarshal(buf, out)
	out.Fields["@type"] = &types.Value{Kind: &types.Value_StringValue{
		StringValue: "type.googleapis.com/envoy.extensions.filters.network.dubbo_proxy.v3.DubboProxy",
	}}

	Value := &types.Struct{
		Fields: map[string]*types.Value{
			"name": {
				Kind: &types.Value_StringValue{
					StringValue: "envoy.filters.network.dubbo_proxy",
				},
			},
			"typed_config": {
				Kind: &types.Value_StructValue{StructValue: out},
			},
		},
	}

	return &networking.EnvoyFilter{
		ConfigPatches: []*networking.EnvoyFilter_EnvoyConfigObjectPatch{
			&networking.EnvoyFilter_EnvoyConfigObjectPatch{
				ApplyTo: networking.EnvoyFilter_NETWORK_FILTER,
				Match: &networking.EnvoyFilter_EnvoyConfigObjectMatch{
					ObjectTypes: &networking.EnvoyFilter_EnvoyConfigObjectMatch_Listener{
						Listener: &networking.EnvoyFilter_ListenerMatch{
							Name: listenerName,
							FilterChain: &networking.EnvoyFilter_ListenerMatch_FilterChainMatch{
								Filter: &networking.EnvoyFilter_ListenerMatch_FilterMatch{
									Name: wellknown.TCPProxy,
								},
							},
						},
					},
				},
				Patch: &networking.EnvoyFilter_Patch{
					Operation: networking.EnvoyFilter_Patch_REMOVE,
					Value:     Value,
				},
			},
		},
	}
}
