// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.
//go:build linux

package nftables

import (
	nft "github.com/google/nftables"
)

type TableFamily = nft.TableFamily

const (
	TableFamilyINet   TableFamily = nft.TableFamilyINet
	TableFamilyIPv4   TableFamily = nft.TableFamilyIPv4
	TableFamilyIPv6   TableFamily = nft.TableFamilyIPv6
	TableFamilyARP    TableFamily = nft.TableFamilyARP
	TableFamilyNetdev TableFamily = nft.TableFamilyNetdev
	TableFamilyBridge TableFamily = nft.TableFamilyBridge
)
