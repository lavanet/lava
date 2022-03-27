package keeper

import (
	"encoding/binary"
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/servicer/types"
	usertypes "github.com/lavanet/lava/x/user/types"
	tendermintcrypto "github.com/tendermint/tendermint/crypto"
)

func (k Keeper) verifyPairingData(ctx sdk.Context, specID uint64, clientAddress sdk.AccAddress, isNew bool) (bool, error) {

	spec, found := k.specKeeper.GetSpec(ctx, specID)
	if !found {
		return false, fmt.Errorf("spec not found for id given: %d", specID)
	}

	verifiedUser := false

	//we get the latest user specStorage, we dont need the list of all users at all sessions, so we dont load the right one at the right storage
	//instead, to overcome unstaking users problems, we add support for unstaking users, and make sure users can't unstake sooner than savedSessions*blocksPerSession
	userSpecStakeStorageForSpec, found := k.userKeeper.GetSpecStakeStorage(ctx, spec.Name)
	if !found {
		return false, fmt.Errorf("no user spec stake storage for spec %s", spec.Name)
	}
	allStakedUsersForSpec := userSpecStakeStorageForSpec.StakeStorage.StakedUsers
	for _, stakedUser := range allStakedUsersForSpec {
		userAddr, err := sdk.AccAddressFromBech32(stakedUser.Index)
		if err != nil {
			panic(fmt.Sprintf("invalid user address saved in keeper %s, err: %s", stakedUser.Index, err))
		}
		if userAddr.Equals(clientAddress) {
			verifiedUser = true
			break
		}
	}
	if !isNew {
		//verification for unstaking users too, for the proof of work to work on unstaking users that used relays before unstaking
		unstakingUsersForSpec, _ := k.userKeeper.GetUnstakingUsersForSpec(ctx, usertypes.SpecName{Name: spec.Name})
		for _, userStake := range unstakingUsersForSpec {
			userAddr, err := sdk.AccAddressFromBech32(userStake.Index)
			if err != nil {
				panic(fmt.Sprintf("invalid user address saved on unstaking users storage in keeper %s, err: %s", userStake.Index, err))
			}
			if userAddr.Equals(clientAddress) {
				verifiedUser = true
				break
			}
		}

	}
	if !verifiedUser {
		return false, fmt.Errorf("client: %s isn't staked for spec %s", clientAddress, spec.Name)
	}
	return true, nil
}

//function used to get a new pairing from relayer and client
//first argument has all metadata, second argument is only the addresses
func (k Keeper) GetPairingForClient(ctx sdk.Context, specID uint64, clientAddress sdk.AccAddress) (servicerOptions []types.StakeMap, errorRet error) {
	k.verifyPairingData(ctx, specID, clientAddress, true)
	spec, _ := k.specKeeper.GetSpec(ctx, specID)
	sessionStart, found := k.GetCurrentSessionStart(ctx)
	if !found {
		return nil, fmt.Errorf("did not find currentSessionStart in keeper")
	}
	stakeStorage, _, err := k.GetSpecStakeStorageInSessionStorageForSpec(ctx, sessionStart.Block, spec.Name)
	if err != nil {
		return nil, fmt.Errorf("invalid GetSpecStakeStorageInSessionStorageForSpec when pairing: block: %d, error: %s", sessionStart.Block, err)
	}

	servicerOptions, _, errorRet = k.calculatePairingForClient(ctx, stakeStorage, clientAddress, sessionStart.Block)
	return
}

func (k Keeper) ValidatePairingForClient(ctx sdk.Context, specID uint64, clientAddress sdk.AccAddress, servicerAddress sdk.AccAddress, block types.BlockNum) (isValidPairing bool, isOverlap bool, errorRet error) {
	//TODO: this is by spec ID but spec might change, and we validate a past spec, and all our stuff are by specName, this can be a problem
	k.verifyPairingData(ctx, specID, clientAddress, false)
	spec, _ := k.specKeeper.GetSpec(ctx, specID)

	stakeStorage, previousOverlappingStakeStorage, err := k.GetSpecStakeStorageInSessionStorageForSpec(ctx, block, spec.Name)
	if err != nil {
		return false, false, err
	}
	sessionStart, overlappingBlock, err := k.GetSessionStartForBlock(ctx, block)
	if err != nil {
		return false, false, err
	}
	_, validAddresses, errorRet := k.calculatePairingForClient(ctx, stakeStorage, clientAddress, *sessionStart)
	if errorRet != nil {
		return false, false, errorRet
	}
	for _, possibleAddr := range validAddresses {
		if possibleAddr.Equals(servicerAddress) {
			return true, false, nil
		}
	}
	if previousOverlappingStakeStorage != nil {
		if overlappingBlock == nil {
			panic("no overlapping block but has overlapping stakeStorage")
		}
		_, validAddressesOverlap, errorRet := k.calculatePairingForClient(ctx, previousOverlappingStakeStorage, clientAddress, *overlappingBlock)
		if errorRet != nil {
			return false, false, errorRet
		}
		//check overlap addresses from previous session
		for _, possibleAddr := range validAddressesOverlap {
			if possibleAddr.Equals(servicerAddress) {
				return true, true, nil
			}
		}
	}
	return false, false, nil
}

func (k Keeper) calculatePairingForClient(ctx sdk.Context, stakedStorage *types.StakeStorage, clientAddress sdk.AccAddress, sessionStartBlock types.BlockNum) (validServicers []types.StakeMap, addrList []sdk.AccAddress, err error) {
	if sessionStartBlock.Num > uint64(ctx.BlockHeight()) {
		k.Logger(ctx).Error("\ninvalid session start\n")
		panic(fmt.Sprintf("invalid session start saved in keeper %d, current block was %d", sessionStartBlock.Num, uint64(ctx.BlockHeight())))
	}
	stakedServicers := stakedStorage.Staked
	//create a list of valid servicers (deadline reached)
	for _, stakeMap := range stakedServicers {
		if stakeMap.Deadline.Num > uint64(ctx.BlockHeight()) {
			//servicer deadline wasn't reached yet
			continue
		}
		validServicers = append(validServicers, stakeMap)
	}

	//calculates a hash and randomly chooses the servicers
	validServicers = k.returnSubsetOfServicersByStake(ctx, validServicers, k.ServicersToPairCount(ctx), sessionStartBlock.Num)

	for _, stakeMap := range validServicers {
		servicerAddress := stakeMap.Index
		servicerAccAddr, err := sdk.AccAddressFromBech32(servicerAddress)
		if err != nil {
			panic(fmt.Sprintf("invalid servicer address saved in keeper %s, err: %s", servicerAddress, err))
		}
		addrList = append(addrList, servicerAccAddr)
	}
	return validServicers, addrList, nil
}

//this function randomly chooses count servicers by weight
func (k Keeper) returnSubsetOfServicersByStake(ctx sdk.Context, servicersMaps []types.StakeMap, count uint64, block uint64) (returnedServicers []types.StakeMap) {
	var stakeSum uint64 = 0
	hashData := make([]byte, 0)
	for _, stakedServicer := range servicersMaps {
		stakeSum += stakedServicer.Stake.Amount.Uint64()
		hashData = append(hashData, []byte(stakedServicer.Index)...)
	}
	if stakeSum == 0 {
		//list is empty
		return
	}
	extradata := make([]byte, 8)
	binary.LittleEndian.PutUint64(extradata, uint64(block))
	hashData = append(hashData, extradata...)

	indexToSkip := make(map[int]bool) // a trick to create a unique set in golang
	for it := 0; it < int(count); it++ {
		hash := tendermintcrypto.Sha256(hashData) // TODO: we use cheaper algo for speed
		bigIntNum := new(big.Int).SetBytes(hash)
		hashAsNumber := sdk.NewIntFromBigInt(bigIntNum)
		modRes := hashAsNumber.ModRaw(int64(stakeSum)).Uint64()
		var newStakeSum uint64 = 0
		for idx, stakedServicer := range servicersMaps {
			if indexToSkip[idx] {
				//this is an index we added
				continue
			}
			newStakeSum += stakedServicer.Stake.Amount.Uint64()
			if modRes < newStakeSum {
				//we hit our chosen servicer
				returnedServicers = append(returnedServicers, stakedServicer)
				stakeSum -= stakedServicer.Stake.Amount.Uint64() //we remove this servicer from the random pool, so the sum is lower now
				indexToSkip[idx] = true
				break
			}
		}
		if uint64(len(returnedServicers)) >= count {
			return returnedServicers
		}
		hashData = append(hashData, []byte{uint8(it)}...)
	}
	return returnedServicers
}
