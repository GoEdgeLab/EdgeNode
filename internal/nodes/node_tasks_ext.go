// Copyright 2023 GoEdge CDN goedge.cdn@gmail.com. All rights reserved. Official site: https://goedge.cn .
//go:build !plus

package nodes

import "github.com/TeaOSLab/EdgeNode/internal/rpc"

func (this *Node) execScriptsChangedTask() error {
	// stub
	return nil
}

func (this *Node) execUAMPolicyChangedTask(rpcClient *rpc.RPCClient) error {
	// stub
	return nil
}

func (this *Node) execHTTPCCPolicyChangedTask(rpcClient *rpc.RPCClient) error {
	// stub
	return nil
}

func (this *Node) execHTTP3PolicyChangedTask(rpcClient *rpc.RPCClient) error {
	// stub
	return nil
}

func (this *Node) execHTTPPagesPolicyChangedTask(rpcClient *rpc.RPCClient) error {
	// stub
	return nil
}

func (this *Node) execNetworkSecurityPolicyChangedTask(rpcClient *rpc.RPCClient) error {
	// stub
	return nil
}
