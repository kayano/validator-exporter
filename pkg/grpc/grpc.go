package grpc

import (
	"context"
	"fmt"
	"time"

	base "cosmossdk.io/api/cosmos/base/tendermint/v1beta1"

	"github.com/archway-network/validator-exporter/pkg/config"
	"github.com/archway-network/validator-exporter/pkg/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/types/query"
	slashing "github.com/cosmos/cosmos-sdk/x/slashing/types"
	staking "github.com/cosmos/cosmos-sdk/x/staking/types"
	"google.golang.org/grpc"

	log "github.com/archway-network/validator-exporter/pkg/logger"
)

const valConsStr = "valcons"

type Client struct {
	ctx       context.Context
	ctxCancel context.CancelFunc
	cfg       config.Config
	conn      *grpc.ClientConn
	connClose func()
}

func NewClient(cfg config.Config) (Client, error) {
	valsInfo := Client{
		cfg: cfg,
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(valsInfo.cfg.Timeout)*time.Second,
	)

	valsInfo.ctx = ctx
	valsInfo.ctxCancel = cancel

	conn, err := cfg.GRPCConn()
	if err != nil {
		return Client{}, err
	}

	valsInfo.conn = conn
	valsInfo.connClose = func() {
		if err := conn.Close(); err != nil {
			log.Error(fmt.Sprintf("failed to close connection :%s", err))
		}
	}

	return valsInfo, nil
}

func (c Client) SignigInfos() ([]slashing.ValidatorSigningInfo, error) {
	infos := []slashing.ValidatorSigningInfo{}
	key := []byte{}
	client := slashing.NewQueryClient(c.conn)

	for {
		request := &slashing.QuerySigningInfosRequest{Pagination: &query.PageRequest{Key: key}}

		slashRes, err := client.SigningInfos(c.ctx, request)
		if err != nil {
			return nil, err
		}

		if slashRes == nil {
			return nil, fmt.Errorf("got empty response from signing infos endpoint")
		}

		infos = append(infos, slashRes.GetInfo()...)

		page := slashRes.GetPagination()
		if page == nil {
			break
		}

		key = page.GetNextKey()
		if len(key) == 0 {
			break
		}
	}

	log.Debug(fmt.Sprintf("SigningInfos: %d", len(infos)))

	return infos, nil
}

func (c Client) Validators() ([]staking.Validator, error) {
	vals := []staking.Validator{}
	key := []byte{}
	encCfg := testutil.MakeTestEncodingConfig()
	interfaceRegistry := encCfg.InterfaceRegistry
	client := staking.NewQueryClient(c.conn)

	for {
		request := &staking.QueryValidatorsRequest{Pagination: &query.PageRequest{Key: key}}

		stakingRes, err := client.Validators(c.ctx, request)
		if err != nil {
			return nil, err
		}

		if stakingRes == nil {
			return nil, fmt.Errorf("got empty response from validators endpoint")
		}

		for _, v := range stakingRes.GetValidators() {
			err = v.UnpackInterfaces(interfaceRegistry)
			if err != nil {
				return nil, err
			}

			vals = append(vals, v)
		}

		page := stakingRes.GetPagination()
		if page == nil {
			break
		}

		key = page.GetNextKey()
		if len(key) == 0 {
			break
		}
	}

	log.Debug(fmt.Sprintf("Validators: %d", len(vals)))

	return vals, nil
}

func (c Client) valConsMap(vals []staking.Validator) (map[string]staking.Validator, error) {
	vMap := map[string]staking.Validator{}

	for _, v := range vals {
		addr, err := v.GetConsAddr()
		if err != nil {
			return nil, err
		}

		consAddr, err := bech32.ConvertAndEncode(c.cfg.Prefix+valConsStr, sdk.ConsAddress(addr))
		if err != nil {
			return nil, err
		}

		vMap[consAddr] = v
	}

	return vMap, nil
}

func SigningValidators(cfg config.Config) ([]types.Validator, error) {
	sVals := []types.Validator{}

	client, err := NewClient(cfg)
	if err != nil {
		log.Error(err.Error())

		return []types.Validator{}, err
	}

	defer client.connClose()
	defer client.ctxCancel()

	sInfos, err := client.SignigInfos()
	if err != nil {
		log.Error(err.Error())

		return []types.Validator{}, err
	}

	vals, err := client.Validators()
	if err != nil {
		log.Error(err.Error())

		return []types.Validator{}, err
	}

	valsMap, err := client.valConsMap(vals)
	if err != nil {
		log.Error(err.Error())

		return []types.Validator{}, err
	}

	for _, info := range sInfos {
		if _, ok := valsMap[info.Address]; !ok {
			log.Debug(fmt.Sprintf("Not in validators: %s", info.Address))
		}

		sVals = append(sVals, types.Validator{
			ConsAddress:     info.Address,
			OperatorAddress: valsMap[info.Address].OperatorAddress,
			Moniker:         valsMap[info.Address].Description.Moniker,
			MissedBlocks:    info.MissedBlocksCounter,
		})
	}

	return sVals, nil
}

func LatestBlockHeight(cfg config.Config) (int64, error) {
	client, err := NewClient(cfg)
	if err != nil {
		log.Error(err.Error())

		return 0, err
	}

	defer client.connClose()
	defer client.ctxCancel()

	request := &base.GetLatestBlockRequest{}
	baseClient := base.NewServiceClient(client.conn)

	blockResp, err := baseClient.GetLatestBlock(client.ctx, request)
	if err != nil {
		log.Error(err.Error())

		return 0, err
	}

	height := blockResp.GetBlock().Header.Height
	log.Debug(fmt.Sprintf("Latest height: %+v", height))

	return height, nil
}
