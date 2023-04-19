// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.
//go:build linux

package nftables

import nft "github.com/google/nftables"

type ChainPolicy = nft.ChainPolicy

// Possible ChainPolicy values.
const (
	ChainPolicyDrop   = nft.ChainPolicyDrop
	ChainPolicyAccept = nft.ChainPolicyAccept
)
