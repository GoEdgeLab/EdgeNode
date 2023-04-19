// Copyright 2022 Liuxiangchao iwind.liu@gmail.com. All rights reserved.
//go:build linux

package nftables

import nft "github.com/google/nftables"

type SetDataType = nft.SetDatatype

var (
	TypeInvalid     = nft.TypeInvalid
	TypeVerdict     = nft.TypeVerdict
	TypeNFProto     = nft.TypeNFProto
	TypeBitmask     = nft.TypeBitmask
	TypeInteger     = nft.TypeInteger
	TypeString      = nft.TypeString
	TypeLLAddr      = nft.TypeLLAddr
	TypeIPAddr      = nft.TypeIPAddr
	TypeIP6Addr     = nft.TypeIP6Addr
	TypeEtherAddr   = nft.TypeEtherAddr
	TypeEtherType   = nft.TypeEtherType
	TypeARPOp       = nft.TypeARPOp
	TypeInetProto   = nft.TypeInetProto
	TypeInetService = nft.TypeInetService
	TypeICMPType    = nft.TypeICMPType
	TypeTCPFlag     = nft.TypeTCPFlag
	TypeDCCPPktType = nft.TypeDCCPPktType
	TypeMHType      = nft.TypeMHType
	TypeTime        = nft.TypeTime
	TypeMark        = nft.TypeMark
	TypeIFIndex     = nft.TypeIFIndex
	TypeARPHRD      = nft.TypeARPHRD
	TypeRealm       = nft.TypeRealm
	TypeClassID     = nft.TypeClassID
	TypeUID         = nft.TypeUID
	TypeGID         = nft.TypeGID
	TypeCTState     = nft.TypeCTState
	TypeCTDir       = nft.TypeCTDir
	TypeCTStatus    = nft.TypeCTStatus
	TypeICMP6Type   = nft.TypeICMP6Type
	TypeCTLabel     = nft.TypeCTLabel
	TypePktType     = nft.TypePktType
	TypeICMPCode    = nft.TypeICMPCode
	TypeICMPV6Code  = nft.TypeICMPV6Code
	TypeICMPXCode   = nft.TypeICMPXCode
	TypeDevGroup    = nft.TypeDevGroup
	TypeDSCP        = nft.TypeDSCP
	TypeECN         = nft.TypeECN
	TypeFIBAddr     = nft.TypeFIBAddr
	TypeBoolean     = nft.TypeBoolean
	TypeCTEventBit  = nft.TypeCTEventBit
	TypeIFName      = nft.TypeIFName
	TypeIGMPType    = nft.TypeIGMPType
	TypeTimeDate    = nft.TypeTimeDate
	TypeTimeHour    = nft.TypeTimeHour
	TypeTimeDay     = nft.TypeTimeDay
	TypeCGroupV2    = nft.TypeCGroupV2
)
