package zrx

import (
	"encoding/hex"

	"github.com/0xProject/0x-mesh/zeroex"
	"github.com/ethereum/go-ethereum/common"
)

const AssetDataLength = int(36)
const AssetDataPrefixLength = int(4)

// EncodeERC20AssetData returns the encoded asset data for the token address
// Details: https://github.com/0xProject/0x-protocol-specification/blob/master/v3/v3-specification.md#assetdata
func EncodeERC20AssetData(address common.Address) []byte {
	prefixBytes, _ := hex.DecodeString(zeroex.ERC20AssetDataID)
	assetData := common.LeftPadBytes(address.Bytes(), AssetDataLength)
	for i := range assetData {
		if i < AssetDataPrefixLength {
			assetData[i] = prefixBytes[i]
		}
	}
	return assetData
}
