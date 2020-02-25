package zrx

import (
	"github.com/0xProject/0x-mesh/ethereum/signer"
	"github.com/0xProject/0x-mesh/zeroex"
)

// ECSignatureLength is the length, in bytes, of a ECSignature
const ECSignatureLength = 66

// ECSignatureToBytes converts a 0x ECSignature to it's bytes representation
// Ideally this would be a method on *signer.ECSignature
func ECSignatureToBytes(ecSignature *signer.ECSignature) []byte {
	signature := make([]byte, ECSignatureLength)
	signature[0] = ecSignature.V
	copy(signature[1:33], ecSignature.R[:])
	copy(signature[33:65], ecSignature.S[:])
	signature[65] = byte(zeroex.EthSignSignature)
	return signature
}
