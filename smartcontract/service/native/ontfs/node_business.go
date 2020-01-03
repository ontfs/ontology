/*
 * Copyright (C) 2018 The ontology Authors
 * This file is part of The ontology library.
 *
 * The ontology is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The ontology is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public License
 * along with The ontology.  If not, see <http://www.gnu.org/licenses/>.
 */

package ontfs

import (
	"fmt"
	"github.com/ontio/ontology-crypto/keypair"
	"github.com/ontio/ontology-crypto/pdp"
	"github.com/ontio/ontology-crypto/signature"
	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/core/types"
	"github.com/ontio/ontology/errors"
	"github.com/ontio/ontology/smartcontract/service/native"
	"github.com/ontio/ontology/smartcontract/service/native/utils"
)

func FsFileProve(native *native.NativeService) ([]byte, error) {
	var pdpData PdpData
	source := common.NewZeroCopySource(native.Input)
	if err := pdpData.Deserialization(source); err != nil {
		return utils.BYTE_FALSE, errors.NewErr("[Node Business] FsFileProve Deserialization error!")
	}
	if !native.ContextRef.CheckWitness(pdpData.NodeAddr) {
		return utils.BYTE_FALSE, errors.NewErr("[Node Business] FsFileProve CheckWitness failed!")
	}

	globalParam, err := getGlobalParam(native)
	if err != nil {
		return utils.BYTE_FALSE, errors.NewErr("[Node Business] FsFileProve getGlobalParam error!")
	}

	fileInfo := getFileInfoByHash(native, pdpData.FileHash)
	if fileInfo == nil {
		return utils.BYTE_FALSE, errors.NewErr("[Node Business] FsFileProve getFileInfoByHash error!")
	}

	nodeInfo := getNodeInfo(native, pdpData.NodeAddr)
	if nodeInfo == nil {
		return utils.BYTE_FALSE, errors.NewErr("[Node Business] FsFileProve getNodeInfo error!")
	}

	currPdpEndPoint := calcPdpEndPoint(fileInfo.TimeStart, fileInfo.PdpInterval, uint64(native.Time))
	if currPdpEndPoint > fileInfo.TimeExpired {
		currPdpEndPoint = fileInfo.TimeExpired
	}
	pdpRecord := getPdpRecord(native, fileInfo.FileHash, fileInfo.FileOwner, pdpData.NodeAddr)
	if pdpRecord == nil {
		if err = checkPdpData(native, &pdpData, fileInfo); err != nil {
			return utils.BYTE_FALSE, fmt.Errorf("[Node Business] FsFileProve checkPdpData(file) error: %s",
				err.Error())
		}
		pdpRecord = &PdpRecord{NodeAddr: pdpData.NodeAddr, FileHash: pdpData.FileHash,
			FileOwner: fileInfo.FileOwner, PdpCount: 0, LastPdpTime: currPdpEndPoint,
			NextHeight: uint64(native.Height) + DefaultPdpHeightIV, SettleFlag: false}

		if nodeInfo.RestVol < fileInfo.FileBlockCount*DefaultPerBlockSize {
			return utils.BYTE_FALSE, errors.NewErr("[Node Business] FsFileProve space RestVol not enough error!")
		}

		nodeInfo.RestVol -= fileInfo.FileBlockCount * DefaultPerBlockSize
	} else {
		if pdpRecord.SettleFlag {
			return utils.BYTE_FALSE, errors.NewErr("[Node Business] FsFileProve pdp finished!")
		}
		if uint64(native.Time) <= pdpRecord.LastPdpTime {
			return utils.BYTE_FALSE, errors.NewErr("[Node Business] FsFileProve already FileProve!")
		}
		if pdpData.ChallengeHeight != pdpRecord.NextHeight {
			return utils.BYTE_FALSE, errors.NewErr("[Node Business] FsFileProve pdpData ChallengeHeight error!")
		}
		if err = checkPdpData(native, &pdpData, fileInfo); err != nil {
			return utils.BYTE_FALSE, fmt.Errorf("[Node Business] FsFileProve checkPdpData(space) error: %s",
				err.Error())
		}

		pdpRecord.PdpCount += 1
		pdpRecord.LastPdpTime = currPdpEndPoint
		pdpRecord.NextHeight = uint64(native.Height) + DefaultPdpHeightIV

		var oncePdpProfit uint64
		if fileInfo.StorageType == FileStorageTypeUseFile {
			oncePdpProfit = calcPerFileOncePdpProfitByFile(fileInfo)
			if fileInfo.RestAmount < oncePdpProfit {
				return utils.BYTE_FALSE, errors.NewErr("[Node Business] FsFileProve file RestAmount not enough error!")
			}
			fileInfo.RestAmount -= oncePdpProfit
			fileInfo.FileCost += oncePdpProfit
		} else if fileInfo.StorageType == FileStorageTypeUseSpace {
			space := getAndUpdateSpaceInfo(native, fileInfo.FileOwner)
			if space == nil {
				return utils.BYTE_FALSE, errors.NewErr("[Node Business] FsFileProve getAndUpdateSpaceInfo error!")
			}
			oncePdpProfit = calcPerFileOncePdpProfitBySpace(fileInfo, space, globalParam.GasPerKbForSaveWithSpace)
			if space.RestAmount < oncePdpProfit {
				return utils.BYTE_FALSE, errors.NewErr("[Node Business] FsFileProve space RestAmount not enough error!")
			}
			space.RestAmount -= oncePdpProfit
			fileInfo.FileCost += oncePdpProfit
			addSpaceInfo(native, space)
		} else {
			return utils.BYTE_FALSE, errors.NewErr("[Node Business] FsFileProve file storage type error!")
		}
		nodeInfo.Profit += oncePdpProfit
	}

	//file become due, start settlement
	if !fileInfo.ValidFlag {
		nodeInfo.RestVol += fileInfo.FileBlockCount * DefaultPerBlockSize
		pdpRecord.SettleFlag = true
	}

	addFileInfo(native, fileInfo)
	addNodeInfo(native, nodeInfo)
	addPdpRecord(native, pdpRecord)
	return utils.BYTE_TRUE, nil
}

func calcPdpEndPoint(fileTimeStart uint64, pdpInterval uint64, currTime uint64) uint64 {
	fileSaveTime := currTime - fileTimeStart
	return currTime + pdpInterval - fileSaveTime%pdpInterval
}

func checkPdpData(native *native.NativeService, pdpData *PdpData, fileInfo *FileInfo) error {
	blockHeader, err := native.Store.GetHeaderByHeight(uint32(pdpData.ChallengeHeight))
	if err != nil || blockHeader == nil {
		return errors.NewErr("[Node Business] checkPdpData GetHeaderByHeight error!")
	}
	blockHash := blockHeader.Hash()
	hexBlockHash := blockHash.ToArray()

	return CheckPdpProve(pdpData.NodeAddr, hexBlockHash, fileInfo.FileBlockCount, fileInfo.PdpParam, pdpData.ProveData)
}

//export this function for ontfs
func CheckPdpProve(nodeAddr common.Address, blockHash []byte, fileBlockCount uint64, pdpParamData []byte,
	proveData []byte) error {
	var err error

	var filePdpHashSt pdp.FilePdpHashSt
	if err = filePdpHashSt.Deserialize(pdpParamData); err != nil {
		return err
	}

	var pdpObj = pdp.NewPdp(filePdpHashSt.Version)
	blockIndexes := pdpObj.GenChallenge(nodeAddr, blockHash, fileBlockCount)

	for _, blockIndex := range blockIndexes {
		ret := pdpObj.VerifyProofWithPerBlock(vkData, proveData, blockHash, filePdpHashSt.BlockPdpHashes[blockIndex])
		if !ret {
			return errors.NewErr("[Node Business] checkPdpData ProveData Verify failed!")
		}
	}
	return nil
}

func FsReadFileSettle(native *native.NativeService) ([]byte, error) {
	var settleSlice FileReadSettleSlice
	source := common.NewZeroCopySource(native.Input)
	if err := settleSlice.Deserialization(source); err != nil {
		return utils.BYTE_FALSE, errors.NewErr("[Node Business] FsReadFileSettle Deserialization error!")
	}

	if !native.ContextRef.CheckWitness(settleSlice.PayTo) {
		return utils.BYTE_FALSE, errors.NewErr("[Node Business] FsReadFileSettle Check Slice owner failed!")
	}

	globalParam, err := getGlobalParam(native)
	if err != nil {
		return utils.BYTE_FALSE, errors.NewErr("[Node Business] FsReadFileSettle getGlobalParam error!")
	}

	fileInfo := getFileInfoByHash(native, settleSlice.FileHash)
	if fileInfo == nil {
		return utils.BYTE_FALSE, errors.NewErr("[Node Business] FsReadFileSettle getFileInfoByHash error!")
	}

	readPledge, err := getReadPledge(native, settleSlice.PayFrom, settleSlice.FileHash)
	if err != nil {
		return utils.BYTE_FALSE, errors.NewErr("[Node Business] FsReadFileSettle getReadPledge error!")
	}

	for i := 0; i < len(readPledge.ReadPlans); i++ {
		//search FsNode
		if readPledge.ReadPlans[i].NodeAddr != settleSlice.PayTo {
			continue
		}
		if readPledge.ReadPlans[i].HaveReadBlockNum >= settleSlice.SliceId ||
			readPledge.ReadPlans[i].MaxReadBlockNum < settleSlice.SliceId {
			return utils.BYTE_FALSE, errors.NewErr("[Node Business] FsReadFileSettle SliceId error!")
		}
		if readPledge.Downloader != settleSlice.PayFrom {
			return utils.BYTE_FALSE, errors.NewErr("[Node Business] FsReadFileSettle Downloader error!")
		}

		if settleSlice.PledgeHeight != readPledge.BlockHeight {
			return utils.BYTE_FALSE, errors.NewErr("[Node Business] FsReadFileSettle PledgeHeight failed!")
		}

		ret, err := checkSettleSig(settleSlice)
		if err != nil || !ret {
			return utils.BYTE_FALSE, errors.NewErr("[Node Business] FsReadFileSettle checkSettleSig failed!")
		}

		readFee := (settleSlice.SliceId - readPledge.ReadPlans[i].HaveReadBlockNum) * DefaultPerBlockSize *
			globalParam.GasPerKbForRead
		if readPledge.RestMoney < readFee {
			return utils.BYTE_FALSE, errors.NewErr("[Node Business] FsReadFileSettle RestMoney < readFee ")
		}
		//if settleSlice.SliceId == readPledge.ReadPlans[i].MaxReadBlockNum {
		//	var readPlans []ReadPlan
		//	for index, readPlanTmp := range readPledge.ReadPlans {
		//		if index != i {
		//			readPlans = append(readPlans, readPlanTmp)
		//		}
		//	}
		//	readPledge.ReadPlans = readPlans
		//} else {
		//	readPledge.ReadPlans[i].HaveReadBlockNum = settleSlice.SliceId
		//}
		readPledge.ReadPlans[i].HaveReadBlockNum = settleSlice.SliceId
		if readPledge.RestMoney < readFee {
			return utils.BYTE_FALSE, errors.NewErr("[Node Business] FsReadFileSettle RestMoney < readFee error!")
		}
		readPledge.RestMoney -= readFee

		nodeInfo := getNodeInfo(native, settleSlice.PayTo)
		if nodeInfo == nil {
			return utils.BYTE_FALSE, errors.NewErr("[Node Business] FsReadFileSettle getNodeInfo error!")
		}
		nodeInfo.Profit += readFee

		addNodeInfo(native, nodeInfo)
		addReadPledge(native, readPledge)

		return utils.BYTE_TRUE, nil
	}
	return utils.BYTE_FALSE, errors.NewErr("[Node Business] FsReadFileSettle settleSlice PayTo error!")
}

func checkSettleSig(settleSlice FileReadSettleSlice) (bool, error) {
	settleSliceTmp := FileReadSettleSlice{
		FileHash:     settleSlice.FileHash,
		PayFrom:      settleSlice.PayFrom,
		PayTo:        settleSlice.PayTo,
		SliceId:      settleSlice.SliceId,
		PledgeHeight: settleSlice.PledgeHeight,
	}

	sink := common.NewZeroCopySink(nil)
	settleSliceTmp.Serialization(sink)

	pubKey, err := keypair.DeserializePublicKey(settleSlice.PubKey)
	if err != nil {
		return false, fmt.Errorf("checkSettleSig DeserializePublicKey error: %s", err.Error())
	}
	addr := types.AddressFromPubKey(pubKey)
	if addr != settleSlice.PayFrom {
		return false, fmt.Errorf("checkSettleSig Pubkey not match walletAddr ")
	}
	signValue, err := signature.Deserialize(settleSlice.Sig)
	if err != nil {
		return false, fmt.Errorf("checkSettleSig signature Deserialize error: %s", err.Error())
	}

	result := signature.Verify(pubKey, sink.Bytes(), signValue)
	return result, nil
}

func calcPerFileOncePdpProfitByFile(fileInfo *FileInfo) uint64 {
	filePdpNeedCount := (fileInfo.TimeExpired-fileInfo.TimeStart)/fileInfo.PdpInterval + 1
	return (fileInfo.PayAmount / fileInfo.CopyNumber) / filePdpNeedCount
}

func calcTotalFilePayAmountByFile(fileInfo *FileInfo, gasPerKbForSaveWithFile uint64) uint64 {
	filePdpNeedCount := (fileInfo.TimeExpired-fileInfo.TimeStart)/fileInfo.PdpInterval + 1
	return filePdpNeedCount * fileInfo.CopyNumber * fileInfo.FileBlockCount *
		DefaultPerBlockSize * gasPerKbForSaveWithFile
}

func calcPerFileOncePdpProfitBySpace(fileInfo *FileInfo, space *SpaceInfo, gasPerKbForSaveWithSpace uint64) uint64 {
	return fileInfo.FileBlockCount * DefaultPerBlockSize * gasPerKbForSaveWithSpace
}
