package zrx

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

const TestERC20TokenAddress = "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
const TestERC20TokenAssetData = "0xf47261b0000000000000000000000000c02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"

func TestEncodeAssetData(t *testing.T) {
	assetData := hexutil.Encode(EncodeERC20AssetData(common.HexToAddress(TestERC20TokenAddress)))
	assert.Equal(t, TestERC20TokenAssetData, assetData)
}
