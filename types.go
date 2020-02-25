package zrx

import "github.com/ethereum/go-ethereum/signer/core"

// ZeroExTestChainID is the chain ID of the 0x ganache snapshot network
const ZeroExTestChainID = 1337

// TypeEIP712Domain is the name of the EIP-712 domain type
const TypeEIP712Domain = "EIP712Domain"

// TypeZeroExTransaction is the name of the 0x transaction type
const TypeZeroExTransaction = "ZeroExTransaction"

// EIP712Types are the EIP-712 type definitions for the relevant 0x types and domain
var EIP712Types = core.Types{
	"EIP712Domain": {
		{
			Name: "name",
			Type: "string",
		},
		{
			Name: "version",
			Type: "string",
		},
		{
			Name: "chainId",
			Type: "uint256",
		},
		{
			Name: "verifyingContract",
			Type: "address",
		},
	},
	"ZeroExTransaction": {
		{
			Name: "salt",
			Type: "uint256",
		},
		{
			Name: "expirationTimeSeconds",
			Type: "uint256",
		},
		{
			Name: "gasPrice",
			Type: "uint256",
		},
		{
			Name: "signerAddress",
			Type: "address",
		},
		{
			Name: "data",
			Type: "bytes",
		},
	},
}
