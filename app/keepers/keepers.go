package keepers

import (
	log "cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	evidencekeeper "cosmossdk.io/x/evidence/keeper"
	evidencetypes "cosmossdk.io/x/evidence/types"
	"cosmossdk.io/x/feegrant"
	feegrantkeeper "cosmossdk.io/x/feegrant/keeper"
	upgradekeeper "cosmossdk.io/x/upgrade/keeper"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	appparams "github.com/cosmos-builders/chaos/app/params"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	consensusparamkeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	consensusparamtypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	crisiskeeper "github.com/cosmos/cosmos-sdk/x/crisis/keeper"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govtypesv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govtypesv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/cosmos/cosmos-sdk/x/params"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	paramproposal "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	icacontrollertypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/types"
	icahost "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host"
	icahosttypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/types"
	"github.com/cosmos/ibc-go/v8/modules/apps/transfer"
	ibctransferkeeper "github.com/cosmos/ibc-go/v8/modules/apps/transfer/keeper"
	ibctransfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	ibcclient "github.com/cosmos/ibc-go/v8/modules/core/02-client"
	ibcclienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
	ibchost "github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"
)

type AppKeepers struct {
	AccountKeeper         *authkeeper.AccountKeeper
	BankKeeper            bankkeeper.BaseKeeper
	CapabilityKeeper      *capabilitykeeper.Keeper
	StakingKeeper         *stakingkeeper.Keeper
	SlashingKeeper        *slashingkeeper.Keeper
	DistrKeeper           *distrkeeper.Keeper
	GovKeeper             *govkeeper.Keeper
	ConsensusParamsKeeper *consensusparamkeeper.Keeper
	CrisisKeeper          *crisiskeeper.Keeper
	UpgradeKeeper         *upgradekeeper.Keeper
	ParamsKeeper          *paramskeeper.Keeper
	IBCKeeper             *ibckeeper.Keeper // IBC Keeper must be a pointer in the app, so we can SetRouter on it correctly
	EvidenceKeeper        *evidencekeeper.Keeper
	TransferKeeper        *ibctransferkeeper.Keeper
	FeeGrantKeeper        *feegrantkeeper.Keeper

	// make scoped keepers public for test purposes
	ScopedIBCKeeper           capabilitykeeper.ScopedKeeper
	ScopedTransferKeeper      capabilitykeeper.ScopedKeeper
	ScopedICAControllerKeeper capabilitykeeper.ScopedKeeper
	ScopedICAHostKeeper       capabilitykeeper.ScopedKeeper
	ScopedLevtradeibcKeeper   capabilitykeeper.ScopedKeeper

	ScopedInterchainqueryKeeper capabilitykeeper.ScopedKeeper
	ScopedRecordsKeeper         capabilitykeeper.ScopedKeeper
	ScopedratelimitKeeper       capabilitykeeper.ScopedKeeper

	TransferAppModule *transfer.AppModule
	TransferIBCModule *transfer.IBCModule

	ICAHostIBCModule *icahost.IBCModule

	// keys to access the substores
	keys    map[string]*storetypes.KVStoreKey
	tkeys   map[string]*storetypes.TransientStoreKey
	memKeys map[string]*storetypes.MemoryStoreKey
}

func (appKeepers *AppKeepers) InitKeepers(
	appCodec codec.Codec,
	bApp *baseapp.BaseApp,
	cdc *codec.LegacyAmino,
	invCheckPeriod uint,
	skipUpgradeHeights map[int64]bool,
	homePath string,
	maccPerms map[string][]string,
	blockedAddress map[string]bool,
) {
	appKeepers.GenerateKeys()
	paramsKeeper := appKeepers.initParamsKeeper(
		appCodec,
		cdc,
		appKeepers.keys[paramstypes.StoreKey],
		appKeepers.tkeys[paramstypes.TStoreKey],
	)
	appKeepers.ParamsKeeper = &paramsKeeper

	consensusParamsKeeper := consensusparamkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(storetypes.NewKVStoreKey(consensusparamtypes.StoreKey)),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		runtime.EventService{},
	)
	appKeepers.ConsensusParamsKeeper = &consensusParamsKeeper

	bApp.SetParamStore(appKeepers.ConsensusParamsKeeper.ParamsStore)

	appKeepers.CapabilityKeeper = capabilitykeeper.NewKeeper(
		appCodec,
		appKeepers.keys[capabilitytypes.StoreKey],
		appKeepers.memKeys[capabilitytypes.MemStoreKey],
	)

	appKeepers.ScopedIBCKeeper = appKeepers.CapabilityKeeper.ScopeToModule(ibchost.ModuleName)
	appKeepers.ScopedTransferKeeper = appKeepers.CapabilityKeeper.ScopeToModule(ibctransfertypes.ModuleName)
	appKeepers.ScopedICAControllerKeeper = appKeepers.CapabilityKeeper.ScopeToModule(icacontrollertypes.SubModuleName)
	appKeepers.ScopedICAHostKeeper = appKeepers.CapabilityKeeper.ScopeToModule(icahosttypes.SubModuleName)
	appKeepers.CapabilityKeeper.Seal()


	accountKeeper := authkeeper.NewAccountKeeper(
		appCodec,
		runtime.NewKVStoreService(storetypes.NewKVStoreKey(authtypes.ModuleName)),
		authtypes.ProtoBaseAccount,
		maccPerms, 
		address.NewBech32Codec(appparams.Bech32Prefix),
		appparams.Bech32Prefix,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	appKeepers.AccountKeeper = &accountKeeper

	appKeepers.BankKeeper = bankkeeper.NewBaseKeeper(
		appCodec,
		runtime.NewKVStoreService(storetypes.NewKVStoreKey(banktypes.ModuleName)),
		appKeepers.AccountKeeper,
		blockedAddress, authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		log.NewNopLogger(),
	)

	appKeepers.StakingKeeper = stakingkeeper.NewKeeper(
		appCodec,
		// appKeepers.keys[stakingtypes.StoreKey],
		runtime.NewKVStoreService(storetypes.NewKVStoreKey(stakingtypes.ModuleName)),
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		address.NewBech32Codec(appparams.Bech32PrefixValAddr),
		address.NewBech32Codec(appparams.Bech32PrefixConsAddr),
	)


	distrKeeper := distrkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(storetypes.NewKVStoreKey(distrtypes.ModuleName)),
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		appKeepers.StakingKeeper,
		authtypes.FeeCollectorName,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	appKeepers.DistrKeeper = &distrKeeper

	slashingKeeper := slashingkeeper.NewKeeper(
		appCodec,
		cdc,
		runtime.NewKVStoreService(storetypes.NewKVStoreKey(slashingtypes.ModuleName)),
		appKeepers.StakingKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	appKeepers.SlashingKeeper = &slashingKeeper

	appKeepers.CrisisKeeper = crisiskeeper.NewKeeper(
		appCodec,
		// appKeepers.keys[crisistypes.StoreKey],
		runtime.NewKVStoreService(storetypes.NewKVStoreKey(crisistypes.ModuleName)),
		invCheckPeriod,
		appKeepers.BankKeeper,
		authtypes.FeeCollectorName,
		authtypes.NewModuleAddress(crisistypes.ModuleName).String(),
		address.NewBech32Codec(appparams.Bech32Prefix),
	)

	feeGrantKeeper := feegrantkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(storetypes.NewKVStoreKey(feegrant.ModuleName)),
		appKeepers.AccountKeeper,
	)
	appKeepers.FeeGrantKeeper = &feeGrantKeeper

	appKeepers.UpgradeKeeper = upgradekeeper.NewKeeper(
		skipUpgradeHeights,
		runtime.NewKVStoreService(storetypes.NewKVStoreKey(upgradetypes.ModuleName)),
		appCodec,
		homePath,
		bApp,
		authtypes.NewModuleAddress(govtypes.ModuleName).String())

	appKeepers.IBCKeeper = ibckeeper.NewKeeper(
		appCodec,
		appKeepers.keys[ibchost.StoreKey],
		appKeepers.GetSubspace(ibchost.ModuleName),
		appKeepers.StakingKeeper,
		appKeepers.UpgradeKeeper,
		appKeepers.ScopedIBCKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	// appKeepers.IBCKeeper = ibcKeeper

	transferKeeper := ibctransferkeeper.NewKeeper(
		appCodec,
		appKeepers.keys[ibctransfertypes.StoreKey],
		appKeepers.GetSubspace(ibctransfertypes.ModuleName),
		appKeepers.IBCKeeper.ChannelKeeper,
		appKeepers.IBCKeeper.ChannelKeeper,
		appKeepers.IBCKeeper.PortKeeper,
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		appKeepers.ScopedTransferKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	appKeepers.TransferKeeper = &transferKeeper

	transferAppModule := transfer.NewAppModule(*appKeepers.TransferKeeper)
	appKeepers.TransferAppModule = &transferAppModule

	transferIbcModule := transfer.NewIBCModule(*appKeepers.TransferKeeper)
	appKeepers.TransferIBCModule = &transferIbcModule

	appKeepers.EvidenceKeeper = evidencekeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(storetypes.NewKVStoreKey(evidencetypes.ModuleName)),
		appKeepers.StakingKeeper,
		appKeepers.SlashingKeeper,
		address.NewBech32Codec(appparams.Bech32Prefix),
		runtime.ProvideCometInfoService(),
	)

	// Register Gov
	govRouter := govtypesv1beta1.NewRouter()
	govRouter.AddRoute(govtypes.RouterKey, govtypesv1beta1.ProposalHandler).
		AddRoute(paramproposal.RouterKey, params.NewParamChangeProposalHandler(*appKeepers.ParamsKeeper)).
		// SoftwareUpgradeProposal is a gov Content type for initiating a software
		// upgrade.
		// Deprecated: This legacy proposal is deprecated in favor of Msg-based gov
		// proposals, see MsgSoftwareUpgrade.
		//
		// Deprecated: Do not use.
		// AddRoute(upgradetypes.RouterKey, upgrade.NewSoftwareUpgradeProposalHandler(appKeepers.UpgradeKeeper)).
		AddRoute(ibcclienttypes.RouterKey, ibcclient.NewClientProposalHandler(appKeepers.IBCKeeper.ClientKeeper))

	appKeepers.GovKeeper = govkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(storetypes.NewKVStoreKey(govtypes.ModuleName)),
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		appKeepers.StakingKeeper,
		appKeepers.DistrKeeper,
		bApp.MsgServiceRouter(),
		govtypes.DefaultConfig(),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)



	var transferStack porttypes.IBCModule = appKeepers.TransferIBCModule

	ibcRouter := porttypes.NewRouter()
	ibcRouter.AddRoute(ibctransfertypes.ModuleName, transferStack)

	appKeepers.IBCKeeper.SetRouter(ibcRouter)

}

// initParamsKeeper init params keeper and its subspaces
func (appKeepers *AppKeepers) initParamsKeeper(appCodec codec.BinaryCodec, legacyAmino *codec.LegacyAmino, key, tkey storetypes.StoreKey) paramskeeper.Keeper {
	paramsKeeper := paramskeeper.NewKeeper(appCodec, legacyAmino, key, tkey)

	paramsKeeper.Subspace(authtypes.ModuleName).WithKeyTable(authtypes.ParamKeyTable())
	paramsKeeper.Subspace(banktypes.ModuleName).WithKeyTable(banktypes.ParamKeyTable())
	paramsKeeper.Subspace(stakingtypes.ModuleName).WithKeyTable(stakingtypes.ParamKeyTable())
	paramsKeeper.Subspace(distrtypes.ModuleName).WithKeyTable(distrtypes.ParamKeyTable())       //nolint:staticcheck
	paramsKeeper.Subspace(slashingtypes.ModuleName).WithKeyTable(slashingtypes.ParamKeyTable()) //nolint:staticcheck
	paramsKeeper.Subspace(govtypes.ModuleName).WithKeyTable(govtypesv1.ParamKeyTable())
	paramsKeeper.Subspace(crisistypes.ModuleName).WithKeyTable(crisistypes.ParamKeyTable())
	paramsKeeper.Subspace(ibctransfertypes.ModuleName)
	paramsKeeper.Subspace(ibchost.ModuleName)
	paramsKeeper.Subspace(icacontrollertypes.SubModuleName)
	paramsKeeper.Subspace(icahosttypes.SubModuleName)

	return paramsKeeper
}

// SetupHooks sets up hooks for module.
func (appKeepers *AppKeepers) SetupHooks() {

	// register the staking hooks
	// NOTE: stakingKeeper above is passed by reference, so that it will contain these hooks
	appKeepers.StakingKeeper.SetHooks(
		stakingtypes.NewMultiStakingHooks(
			appKeepers.DistrKeeper.Hooks(),
			appKeepers.SlashingKeeper.Hooks(),
		),
	)
}

func KVStoreKeys() []string {
	return []string{
		authtypes.StoreKey,
		banktypes.StoreKey,
		stakingtypes.StoreKey,
		distrtypes.StoreKey,
		slashingtypes.StoreKey,
		govtypes.StoreKey,
		paramstypes.StoreKey,
		ibchost.StoreKey,
		upgradetypes.StoreKey,
		feegrant.StoreKey,
		evidencetypes.StoreKey,
		ibctransfertypes.StoreKey,
		capabilitytypes.StoreKey,
		icacontrollertypes.StoreKey,
		icahosttypes.StoreKey,
		crisistypes.StoreKey,
		consensusparamtypes.StoreKey,
	}
}
