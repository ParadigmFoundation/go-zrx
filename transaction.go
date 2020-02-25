package zrx

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/0xProject/0x-mesh/ethereum"
	"github.com/0xProject/0x-mesh/ethereum/signer"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/signer/core"

	"github.com/ParadigmFoundation/go-eth"
)

// ZeroExProtocolName is the EIP-712 domain name of the 0x protocol
const ZeroExProtocolName = "0x Protocol"

// ZeroExProtocolVersion is the EIP-712 domain version of the 0x protocol
const ZeroExProtocolVersion = "3.0.0"

// Transaction represents 0x transaction (see ZEIP-18)
type Transaction struct {
	Salt                  *big.Int
	ExpirationTimeSeconds *big.Int
	GasPrice              *big.Int
	SignerAddress         common.Address
	Data                  []byte

	hash *common.Hash
}

// MarshalJSON implements json.Marshaler
func (tx *Transaction) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{
		"salt":                  tx.Salt.String(),
		"expirationTimeSeconds": tx.ExpirationTimeSeconds.String(),
		"gasPrice":              tx.GasPrice.String(),
		"signerAddress":         strings.ToLower(tx.SignerAddress.Hex()),
		"data":                  hexutil.Encode(tx.Data),
	})
}

// UnmarshalJSON implements json.Unmarshaler
func (tx *Transaction) UnmarshalJSON(data []byte) error {
	var raw transactionJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if err := tx.fromJSON(&raw); err != nil {
		return err
	}

	return nil
}

// ComputeHashForChainID calculates the 0x transaction hash for the provided chain ID.
// See https://github.com/0xProject/0x-protocol-specification/blob/master/v3/v3-specification.md#hashing-a-transaction
func (tx *Transaction) ComputeHashForChainID(chainID int) (common.Hash, error) {
	if tx.hash != nil {
		return *tx.hash, nil
	}

	contractAddresses, err := ethereum.GetContractAddressesForChainID(chainID)
	if err != nil {
		return common.Hash{}, err
	}

	evmChainID := math.NewHexOrDecimal256(int64(chainID))
	domain := core.TypedDataDomain{
		Name:              ZeroExProtocolName,
		Version:           ZeroExProtocolVersion,
		ChainId:           evmChainID,
		VerifyingContract: contractAddresses.Exchange.Hex(),
	}

	typedData := core.TypedData{
		Types:       EIP712Types,
		PrimaryType: TypeZeroExTransaction,
		Domain:      domain,
		Message:     tx.Map(),
	}

	domainSeparator, err := typedData.HashStruct(TypeEIP712Domain, typedData.Domain.Map())
	if err != nil {
		return common.Hash{}, err
	}

	typedDataHash, err := typedData.HashStruct(typedData.PrimaryType, typedData.Message)
	if err != nil {
		return common.Hash{}, err
	}

	rawData := []byte(fmt.Sprintf("\x19\x01%s%s", string(domainSeparator), string(typedDataHash)))
	hashBytes := eth.Keccak256(rawData)
	hash := common.BytesToHash(hashBytes)
	tx.hash = &hash
	return hash, nil
}

// Map returns the transaction as an un-typed map (useful when hashing)
func (tx *Transaction) Map() map[string]interface{} {
	return map[string]interface{}{
		"salt":                  tx.Salt.String(),
		"expirationTimeSeconds": tx.ExpirationTimeSeconds.String(),
		"gasPrice":              tx.GasPrice.String(),
		"signerAddress":         tx.SignerAddress.Hex(),
		"data":                  tx.Data,
	}
}

// ResetHash returns the cached transaction hash to nil
func (tx *Transaction) ResetHash() {
	tx.hash = nil
}

// set a 0x transaction values from a JSON representation
func (tx *Transaction) fromJSON(ztx *transactionJSON) error {
	salt, ok := new(big.Int).SetString(ztx.Salt, 10)
	if !ok {
		return errors.New(`unable to unmarshal value for "salt"`)
	}

	expirationTimeSeconds, ok := new(big.Int).SetString(ztx.ExpirationTimeSeconds, 10)
	if !ok {
		return errors.New(`unable to unmarshal value for "expirationTimeSeconds"`)
	}

	gasPrice, ok := new(big.Int).SetString(ztx.GasPrice, 10)
	if !ok {
		return errors.New(`unable to unmarshal value for "gasPrice"`)
	}

	tx.Salt = salt
	tx.ExpirationTimeSeconds = expirationTimeSeconds
	tx.GasPrice = gasPrice
	tx.SignerAddress = common.HexToAddress(ztx.SignerAddress)

	if ztx.Data[:2] == "0x" {
		tx.Data = common.Hex2Bytes(ztx.Data[2:])
	} else {
		tx.Data = common.Hex2Bytes(ztx.Data)
	}

	data, err := hexutil.Decode(ztx.Data)
	if err != nil {
		return err
	}

	tx.Data = data
	return nil
}

// SignedTransaction represents a signed 0x transaction
type SignedTransaction struct {
	Transaction

	Signature []byte
}

// MarshalJSON implements json.Marshaler
func (stx *SignedTransaction) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{
		"salt":                  stx.Salt.String(),
		"expirationTimeSeconds": stx.ExpirationTimeSeconds.String(),
		"gasPrice":              stx.GasPrice.String(),
		"signerAddress":         strings.ToLower(stx.SignerAddress.Hex()),
		"data":                  hexutil.Encode(stx.Data),
		"signature":             hexutil.Encode(stx.Signature),
	})
}

// UnmarshalJSON implements json.Unmarshaler
func (stx *SignedTransaction) UnmarshalJSON(data []byte) error {
	var raw transactionJSON
	var rawStx signedTransactionJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if err := json.Unmarshal(data, &rawStx); err != nil {
		return err
	}

	if err := stx.fromJSON(&raw); err != nil {
		return err
	}

	sig, err := hexutil.Decode(rawStx.Signature)
	if err != nil {
		return err
	}

	stx.Signature = sig
	return nil
}

// used to assist in un-marshalling 0x transactions
type transactionJSON struct {
	Salt                  string `json:"salt"`
	ExpirationTimeSeconds string `json:"expirationTimeSeconds"`
	GasPrice              string `json:"gasPrice"`
	SignerAddress         string `json:"signerAddress"`
	Data                  string `json:"data"`
}

// used to assist in un-marshalling 0x transactions
type signedTransactionJSON struct {
	transactionJSON
	Signature string `json:"signature"`
}

// SignTransaction signs the 0x transaction with the supplied Signer
func SignTransaction(signer signer.Signer, tx *Transaction, chainID int) (*SignedTransaction, error) {
	hash, err := tx.ComputeHashForChainID(chainID)
	if err != nil {
		return nil, err
	}

	ecSignature, err := signer.EthSign(hash.Bytes(), tx.SignerAddress)
	if err != nil {
		return nil, err
	}

	return &SignedTransaction{
		Transaction: *tx,
		Signature:   ECSignatureToBytes(ecSignature),
	}, nil
}
