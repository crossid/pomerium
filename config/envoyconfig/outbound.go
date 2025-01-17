package envoyconfig

import (
	"fmt"
	"strconv"

	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	envoy_config_listener_v3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	envoy_config_route_v3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	envoy_http_connection_manager "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/pomerium/pomerium/config"
)

func (b *Builder) buildOutboundListener(cfg *config.Config) (*envoy_config_listener_v3.Listener, error) {
	outboundPort, err := strconv.Atoi(cfg.OutboundPort)
	if err != nil {
		return nil, fmt.Errorf("invalid outbound port: %w", err)
	}

	filter, err := b.buildOutboundHTTPConnectionManager()
	if err != nil {
		return nil, fmt.Errorf("error building outbound http connection manager filter: %w", err)
	}

	li := &envoy_config_listener_v3.Listener{
		Name: "outbound-ingress",
		Address: &envoy_config_core_v3.Address{
			Address: &envoy_config_core_v3.Address_SocketAddress{
				SocketAddress: &envoy_config_core_v3.SocketAddress{
					Address: "127.0.0.1",
					PortSpecifier: &envoy_config_core_v3.SocketAddress_PortValue{
						PortValue: uint32(outboundPort),
					},
				},
			},
		},
		FilterChains: []*envoy_config_listener_v3.FilterChain{{
			Name:    "outbound-ingress",
			Filters: []*envoy_config_listener_v3.Filter{filter},
		}},
	}
	return li, nil
}

func (b *Builder) buildOutboundHTTPConnectionManager() (*envoy_config_listener_v3.Filter, error) {
	rc, err := b.buildOutboundRouteConfiguration()
	if err != nil {
		return nil, err
	}

	tc := marshalAny(&envoy_http_connection_manager.HttpConnectionManager{
		CodecType:  envoy_http_connection_manager.HttpConnectionManager_AUTO,
		StatPrefix: "grpc_egress",
		// limit request first byte to last byte time
		RequestTimeout: &durationpb.Duration{
			Seconds: 15,
		},
		RouteSpecifier: &envoy_http_connection_manager.HttpConnectionManager_RouteConfig{
			RouteConfig: rc,
		},
		HttpFilters: []*envoy_http_connection_manager.HttpFilter{{
			Name: "envoy.filters.http.router",
		}},
	})

	return &envoy_config_listener_v3.Filter{
		Name: "envoy.filters.network.http_connection_manager",
		ConfigType: &envoy_config_listener_v3.Filter_TypedConfig{
			TypedConfig: tc,
		},
	}, nil
}

func (b *Builder) buildOutboundRouteConfiguration() (*envoy_config_route_v3.RouteConfiguration, error) {
	return b.buildRouteConfiguration("grpc", []*envoy_config_route_v3.VirtualHost{{
		Name:    "grpc",
		Domains: []string{"*"},
		Routes:  b.buildOutboundRoutes(),
	}})
}

func (b *Builder) buildOutboundRoutes() []*envoy_config_route_v3.Route {
	type Def struct {
		Cluster  string
		Prefixes []string
	}
	defs := []Def{
		{
			Cluster: "pomerium-authorize",
			Prefixes: []string{
				"/envoy.service.auth.v3.Authorization/",
			},
		},
		{
			Cluster: "pomerium-databroker",
			Prefixes: []string{
				"/databroker.DataBrokerService/",
				"/directory.DirectoryService/",
				"/registry.Registry/",
			},
		},
		{
			Cluster: "pomerium-control-plane-grpc",
			Prefixes: []string{
				"/",
			},
		},
	}
	var routes []*envoy_config_route_v3.Route
	for _, def := range defs {
		for _, prefix := range def.Prefixes {
			routes = append(routes, &envoy_config_route_v3.Route{
				Name: def.Cluster,
				Match: &envoy_config_route_v3.RouteMatch{
					PathSpecifier: &envoy_config_route_v3.RouteMatch_Prefix{Prefix: prefix},
					Grpc:          &envoy_config_route_v3.RouteMatch_GrpcRouteMatchOptions{},
				},
				Action: &envoy_config_route_v3.Route_Route{
					Route: &envoy_config_route_v3.RouteAction{
						ClusterSpecifier: &envoy_config_route_v3.RouteAction_Cluster{
							Cluster: def.Cluster,
						},
						// disable the timeout to support grpc streaming
						Timeout:     durationpb.New(0),
						IdleTimeout: durationpb.New(0),
					},
				},
			})
		}
	}
	return routes
}
