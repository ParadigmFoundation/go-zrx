package zrx

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/0xProject/0x-mesh/ethereum"
	"github.com/0xProject/0x-mesh/ethereum/wrappers"
	"github.com/0xProject/0x-mesh/zeroex"
	"github.com/0xProject/0x-mesh/zeroex/ordervalidator"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

// NULL_ADDRESS is the Ethereum address with 20 null bytes
var NULL_ADDRESS = common.Address{}

// PROTOCOL_FEE_MULTIPLIER is the value that a fill transaction's gas price must be multipled by, and paid in ETH
// https://github.com/0xProject/0x-protocol-specification/blob/master/v3/v3-specification.md#protocol-fees
var PROTOCOL_FEE_MULTIPLIER = big.NewInt(150000)

// EXECUTE_FILL_TX_GAS_LIMIT is the maximum gas cost (with buffer) of executing a single fill transaction
// This value accounts for a scenario in which a 0x staking epoch has ended and must be settled
var EXECUTE_FILL_TX_GAS_LIMIT = uint64(330000)

type contracts struct {
	Exchange    *wrappers.Exchange
	ExchangeABI abi.ABI

	DevUtils *wrappers.DevUtilsCaller
}

type ZeroExHelper struct {
	ChainID           *big.Int
	ContractAddresses ethereum.ContractAddresses

	Client         *ethclient.Client
	Contracts      *contracts
	OrderValidator *ordervalidator.OrderValidator
}

func NewZeroExHelper(client *ethclient.Client, maxContentLength int) (*ZeroExHelper, error) {
	chainID, err := client.ChainID(context.Background())
	if err != nil {
		return nil, err
	}

	addresses, err := ethereum.GetContractAddressesForChainID(int(chainID.Int64()))
	if err != nil {
		return nil, err
	}

	exchangeABI, err := abi.JSON(strings.NewReader(wrappers.ExchangeABI))
	if err != nil {
		return nil, err
	}

	exchange, err := wrappers.NewExchange(addresses.Exchange, client)
	if err != nil {
		return nil, err
	}

	devUtils, err := wrappers.NewDevUtilsCaller(addresses.DevUtils, client)
	if err != nil {
		return nil, err
	}

	orderValidator, err := ordervalidator.New(client, int(chainID.Int64()), maxContentLength)
	if err != nil {
		return nil, err
	}

	return &ZeroExHelper{
		ChainID:           chainID,
		ContractAddresses: addresses,
		Client:            client,
		OrderValidator:    orderValidator,
		Contracts:         &contracts{Exchange: exchange, ExchangeABI: exchangeABI, DevUtils: devUtils},
	}, nil
}

// DevUtils returns an initialized 0x DevUtils contract caller
func (zh *ZeroExHelper) DevUtils() *wrappers.DevUtilsCaller { return zh.Contracts.DevUtils }

// GetFillOrderCallData generates the underlying 0x exchange call data for the fill (to be singed by the taker)
func (zh *ZeroExHelper) GetFillOrderCallData(order zeroex.Order, takerAssetAmount *big.Int, signature []byte) ([]byte, error) {
	return zh.Contracts.ExchangeABI.Pack("fillOrder", order, takerAssetAmount, signature)
}

// GetTransactionHash gets the 0x transaction hash for the current chain ID
func (zh *ZeroExHelper) GetTransactionHash(tx *Transaction) (common.Hash, error) {
	return tx.ComputeHashForChainID(int(zh.ChainID.Int64()))
}

// ValidateFill is a convenience wrapper for ordervalidator.BatchValidate with a single order
// In addition, it also verifies the taker balance/allowance if the taker address is present.
func (zh *ZeroExHelper) ValidateFill(ctx context.Context, order *zeroex.SignedOrder, takerAssetAmount *big.Int) error {
	orders := []*zeroex.SignedOrder{order}
	rawValidationResults := zh.OrderValidator.BatchValidate(ctx, orders, true, rpc.LatestBlockNumber)

	if len(rawValidationResults.Rejected) == 1 {
		return fmt.Errorf("%s", rawValidationResults.Rejected[0].Status.Message)
	}

	if len(rawValidationResults.Accepted) != 1 {
		return fmt.Errorf("unable to validate order")
	}

	// if taker is null address, skip taker checks as we cannot verify their balance/allowance
	if order.TakerAddress == NULL_ADDRESS {
		return nil
	}

	takerBalanceInfo, err := zh.Contracts.DevUtils.GetBalanceAndAssetProxyAllowance(nil, order.TakerAddress, order.TakerAssetData)
	if err != nil {
		return err
	}

	if order.TakerAssetAmount.Cmp(takerBalanceInfo.Allowance) > 0 {
		return fmt.Errorf("taker has insufficient allowance for trade: (has: %s), (want: %s)", takerBalanceInfo.Allowance, order.TakerAssetAmount)
	}
	if order.TakerAssetAmount.Cmp(takerBalanceInfo.Balance) > 0 {
		return fmt.Errorf("taker has insufficient balance for trade: (has: %s), (want: %s)", takerBalanceInfo.Balance, order.TakerAssetAmount)
	}

	return nil
}

// ExecuteTransaction prepares ZEIP-18 transaction ztx with signature sig and executes it against the Exchange contract
func (zh *ZeroExHelper) ExecuteTransaction(opts *bind.TransactOpts, ztx *Transaction, sig []byte) (*types.Transaction, error) {
	transaction := wrappers.Struct3{
		Salt:                  ztx.Salt,
		ExpirationTimeSeconds: ztx.ExpirationTimeSeconds,
		GasPrice:              ztx.GasPrice,
		SignerAddress:         ztx.SignerAddress,
		Data:                  ztx.Data,
	}

	return zh.Contracts.Exchange.ExecuteTransaction(opts, transaction, sig)
}

// CreateOrder creates an unsigned order with the specified values, and generates pseudo-random salt
func (zh *ZeroExHelper) CreateOrder(
	maker common.Address,
	taker common.Address,
	sender common.Address,
	feeRecipient common.Address,
	makerAsset common.Address,
	takerAsset common.Address,
	makerAmount *big.Int,
	takerAmount *big.Int,
	makerFee *big.Int,
	takerFee *big.Int,
	makerFeeAsset common.Address,
	takerFeeAsset common.Address,
	expirationTimeSeconds *big.Int,
) (*zeroex.Order, error) {
	salt, err := GeneratePseudoRandomSalt()
	if err != nil {
		return nil, err
	}

	var makerFeeAssetData []byte
	if makerFeeAsset != NULL_ADDRESS {
		makerFeeAssetData = EncodeERC20AssetData(makerFeeAsset)
	}

	var takerFeeAssetData []byte
	if takerFeeAsset != NULL_ADDRESS {
		takerFeeAssetData = EncodeERC20AssetData(takerFeeAsset)
	}

	return &zeroex.Order{
		ChainID:               zh.ChainID,
		ExchangeAddress:       zh.ContractAddresses.Exchange,
		MakerAddress:          maker,
		MakerAssetData:        EncodeERC20AssetData(makerAsset),
		MakerFeeAssetData:     makerFeeAssetData,
		MakerAssetAmount:      makerAmount,
		MakerFee:              makerFee,
		TakerAddress:          taker,
		TakerAssetData:        EncodeERC20AssetData(takerAsset),
		TakerFeeAssetData:     takerFeeAssetData,
		TakerAssetAmount:      takerAmount,
		TakerFee:              takerFee,
		SenderAddress:         sender,
		FeeRecipientAddress:   feeRecipient,
		ExpirationTimeSeconds: expirationTimeSeconds,
		Salt:                  salt,
	}, nil
}
