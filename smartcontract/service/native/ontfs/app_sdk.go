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
	"math/rand"
	"time"

	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/common/log"
	"github.com/ontio/ontology/errors"
	"github.com/ontio/ontology/smartcontract/service/native"
	"github.com/ontio/ontology/smartcontract/service/native/utils"
)

func FsGetNodeInfoList(native *native.NativeService) ([]byte, error) {
	var nodesInfoList FsNodeInfoList

	source := common.NewZeroCopySource(native.Input)
	count, err := utils.DecodeVarUint(source)
	if err != nil {
		return utils.BYTE_FALSE, errors.NewErr("[APP SDK] FsGetNodeInfoList DecodeVarBytes error!")
	}

	nodeList := getNodeAddrList(native)
	if nodeList != nil {
		r := rand.New(rand.NewSource(time.Now().Unix()))
		r.Shuffle(len(nodeList), func(i, j int) {
			nodeList[i], nodeList[j] = nodeList[j], nodeList[i]
		})
	}

	for _, addr := range nodeList {
		nodeInfo := getNodeInfo(native, addr)
		if nodeInfo == nil {
			fmt.Errorf("[APP SDK] FsGetNodeInfoList getNodeInfo(%v) error", addr)
			continue
		}
		nodesInfoList.NodesInfo = append(nodesInfoList.NodesInfo, *nodeInfo)
		count--
		if count <= 0 {
			break
		}
	}

	sink := common.NewZeroCopySink(nil)
	nodesInfoList.Serialization(sink)

	return EncRet(true, sink.Bytes()), nil
}

func FsCreateSpace(native *native.NativeService) ([]byte, error) {
	contract := native.ContextRef.CurrentContext().ContractAddress

	var spaceInfo SpaceInfo
	spaceInfoSrc := common.NewZeroCopySource(native.Input)
	spaceInfoData, err := DecodeVarBytes(spaceInfoSrc)
	if err != nil {
		return utils.BYTE_FALSE, errors.NewErr("[APP SDK] FsCreateSpace DecodeVarBytes error!")
	}

	source := common.NewZeroCopySource(spaceInfoData)
	if err := spaceInfo.Deserialization(source); err != nil {
		return utils.BYTE_FALSE, errors.NewErr("[APP SDK] FsCreateSpace Deserialization error!")
	}

	if !native.ContextRef.CheckWitness(spaceInfo.SpaceOwner) {
		return utils.BYTE_FALSE, errors.NewErr("[APP SDK] FsCreateSpace CheckSpaceOwner failed!")
	}

	if spaceInfoExist(native, spaceInfo.SpaceOwner) {
		return utils.BYTE_FALSE, errors.NewErr("[APP SDK] FsCreateSpace Space has been created!")
	}

	if spaceInfo.PdpInterval == 0 {
		return utils.BYTE_FALSE, errors.NewErr("[APP SDK] FsCreateSpace PdpInterval equals zero!")
	}

	globalParam, err := getGlobalParam(native)
	if err != nil {
		return utils.BYTE_FALSE, errors.NewErr("[APP SDK] FsCreateSpace getGlobalParam error!")
	}

	spaceInfo.ValidFlag = true
	spaceInfo.TimeStart = uint64(native.Time)
	spaceInfo.RestVol = spaceInfo.Volume

	if spaceInfo.TimeExpired <= spaceInfo.TimeStart {
		return utils.BYTE_FALSE, errors.NewErr("[APP SDK] FsCreateSpace TimeExpired less than TimeStart!")
	}

	spacePdpNeedCount := (spaceInfo.TimeExpired-spaceInfo.TimeStart)/spaceInfo.PdpInterval + 1
	spaceInfo.PayAmount = spacePdpNeedCount * spaceInfo.Volume * spaceInfo.CopyNumber *
		globalParam.GasPerKbForSaveWithSpace
	spaceInfo.RestAmount = spaceInfo.PayAmount

	err = appCallTransfer(native, utils.OngContractAddress, spaceInfo.SpaceOwner, contract, spaceInfo.PayAmount)
	if err != nil {
		return utils.BYTE_FALSE, errors.NewErr("[APP SDK] FsCreateSpace AppCallTransfer, transfer error!")
	}
	addSpaceInfo(native, &spaceInfo)
	return utils.BYTE_TRUE, nil
}

func FsDeleteSpace(native *native.NativeService) ([]byte, error) {
	contract := native.ContextRef.CurrentContext().ContractAddress

	source := common.NewZeroCopySource(native.Input)
	spaceOwner, err := utils.DecodeAddress(source)
	if err != nil {
		return utils.BYTE_FALSE, errors.NewErr("[APP SDK] FsDeleteSpace DecodeAddress error!")
	}

	if !native.ContextRef.CheckWitness(spaceOwner) {
		return utils.BYTE_FALSE, errors.NewErr("[APP SDK] FsDeleteSpace CheckSpaceOwner failed!")
	}

	space := getAndUpdateSpaceInfo(native, spaceOwner)
	if space == nil {
		return utils.BYTE_FALSE, errors.NewErr("[APP SDK] FsDeleteSpace getAndUpdateSpaceInfo error!")
	}
	if space.Volume != space.RestVol {
		return utils.BYTE_FALSE, errors.NewErr("[APP SDK] FsDeleteSpace not allow, check files!")
	}

	err = appCallTransfer(native, utils.OngContractAddress, contract, space.SpaceOwner, space.RestAmount)
	if err != nil {
		return utils.BYTE_FALSE, errors.NewErr("[APP SDK] FsDeleteSpace AppCallTransfer, transfer error!")
	}

	delSpaceInfo(native, spaceOwner)
	return utils.BYTE_TRUE, nil
}

func FsUpdateSpace(native *native.NativeService) ([]byte, error) {
	contract := native.ContextRef.CurrentContext().ContractAddress

	var spaceUpdate SpaceUpdate
	spaceUpdateSrc := common.NewZeroCopySource(native.Input)
	spaceInfoData, err := DecodeVarBytes(spaceUpdateSrc)
	if err != nil {
		return utils.BYTE_FALSE, errors.NewErr("[APP SDK] FsUpdateSpace DecodeVarBytes error!")
	}

	source := common.NewZeroCopySource(spaceInfoData)
	if err := spaceUpdate.Deserialization(source); err != nil {
		return utils.BYTE_FALSE, errors.NewErr("[APP SDK] FsUpdateSpace Deserialization error!")
	}

	if spaceUpdate.NewTimeExpired == 0 && spaceUpdate.NewVolume == 0 {
		return utils.BYTE_FALSE, errors.NewErr("[APP SDK] FsUpdateSpace Param error!")
	}

	if !native.ContextRef.CheckWitness(spaceUpdate.Payer) {
		return utils.BYTE_FALSE, errors.NewErr("[APP SDK] FsUpdateSpace CheckPayer failed!")
	}

	globalParam, err := getGlobalParam(native)
	if err != nil {
		return utils.BYTE_FALSE, errors.NewErr("[APP SDK] FsUpdateSpace getGlobalParam error!")
	}

	space := getAndUpdateSpaceInfo(native, spaceUpdate.SpaceOwner)
	if space == nil {
		return utils.BYTE_FALSE, errors.NewErr("[APP SDK] FsUpdateSpace getSpaceRawInfo error!")
	}

	if !space.ValidFlag {
		return utils.BYTE_FALSE, errors.NewErr("[APP SDK] FsUpdateSpace space timeExpired! please create space again")
	}

	if spaceUpdate.NewTimeExpired != 0 && uint64(native.Time) >= spaceUpdate.NewTimeExpired {
		return utils.BYTE_FALSE, errors.NewErr("[APP SDK] FsUpdateSpace NewTimeExpired error!")
	}

	if spaceUpdate.NewTimeExpired == 0 {
		spaceUpdate.NewTimeExpired = space.TimeExpired
	}

	if spaceUpdate.NewVolume == 0 {
		spaceUpdate.NewVolume = space.Volume
	}

	if space.Volume-space.RestVol >= spaceUpdate.NewVolume {
		return utils.BYTE_FALSE, errors.NewErr("[APP SDK] FsUpdateSpace NewVolume is not enough!")
	}

	newSpacePdpNeedCount := (spaceUpdate.NewTimeExpired-space.TimeStart)/space.PdpInterval + 1
	newPayAmount := newSpacePdpNeedCount * spaceUpdate.NewVolume * space.CopyNumber * globalParam.GasPerKbForSaveWithSpace

	var newFee uint64
	var payer, payee common.Address
	if newPayAmount > space.PayAmount {
		newFee = newPayAmount - space.PayAmount
		payer = spaceUpdate.Payer
		payee = contract
		space.RestAmount += newFee
	} else if newPayAmount < space.PayAmount {
		newFee = space.PayAmount - newPayAmount
		payee = spaceUpdate.Payer
		payer = contract
		if space.RestAmount < newFee {
			return utils.BYTE_FALSE, errors.NewErr("[APP SDK] FsUpdateSpace space RestAmount < newFee error!")
		}
		space.RestAmount -= newFee
	} else {
		newFee = 0
	}
	space.PayAmount = newPayAmount
	space.RestVol = spaceUpdate.NewVolume - (space.Volume - space.RestVol)
	space.Volume = spaceUpdate.NewVolume
	space.TimeExpired = spaceUpdate.NewTimeExpired

	if newFee != 0 {
		err = appCallTransfer(native, utils.OngContractAddress, payer, payee, newFee)
		if err != nil {
			return utils.BYTE_FALSE, errors.NewErr("[APP SDK] FsUpdateSpace AppCallTransfer, transfer error!")
		}
	}

	addSpaceInfo(native, space)
	return utils.BYTE_TRUE, nil
}

func FsGetSpaceInfo(native *native.NativeService) ([]byte, error) {
	source := common.NewZeroCopySource(native.Input)
	spaceOwner, err := utils.DecodeAddress(source)
	if err != nil {
		return EncRet(false, []byte("[APP SDK] FsGetSpaceInfo DecodeAddress error!")), nil
	}

	spaceInfo := getSpaceRawRealInfo(native, spaceOwner)
	if spaceInfo == nil {
		return EncRet(false, []byte("[APP SDK] FsGetSpaceInfo getSpaceRawInfo error!")), nil
	}

	return EncRet(true, spaceInfo), nil
}

func FsStoreFiles(native *native.NativeService) ([]byte, error) {
	contract := native.ContextRef.CurrentContext().ContractAddress

	var errInfos Errors
	var fileInfoList FileInfoList
	source := common.NewZeroCopySource(native.Input)
	fileInfoListData, err := DecodeVarBytes(source)
	if err != nil {
		return utils.BYTE_FALSE, errors.NewErr("[APP SDK] FsStoreFiles DecodeVarBytes error!")
	}

	fileInfoListDataSrc := common.NewZeroCopySource(fileInfoListData)
	if err := fileInfoList.Deserialization(fileInfoListDataSrc); err != nil {
		return utils.BYTE_FALSE, errors.NewErr("[APP SDK] FsStoreFiles Deserialization error!")
	}

	globalParam, err := getGlobalParam(native)
	if err != nil {
		return utils.BYTE_FALSE, errors.NewErr("[APP SDK] FsStoreFiles getGlobalParam error!")
	}

	for _, fileInfo := range fileInfoList.FilesI {
		if !native.ContextRef.CheckWitness(fileInfo.FileOwner) {
			errInfos.AddObjectError(string(fileInfo.FileHash), "[APP SDK] FsStoreFiles CheckFileOwner failed!")
			log.Error("[APP SDK] FsStoreFiles CheckFileOwner failed!")
			continue
		}

		if fileInfo.PdpInterval == 0 {
			errInfos.AddObjectError(string(fileInfo.FileHash), "[APP SDK] FsStoreFiles PdpInterval equals zero!")
			log.Error("[APP SDK] FsStoreFiles PdpInterval equals zero!")
			continue
		}

		if fileInfo.TimeExpired < uint64(native.Time) {
			errInfos.AddObjectError(string(fileInfo.FileHash), "[APP SDK] FsStoreFiles fileInfo TimeExpired error!")
			log.Error("[APP SDK] FsStoreFiles fileInfo TimeExpired error!")
			continue
		}

		if fileExist := getAndUpdateFileInfo(native, fileInfo.FileOwner, fileInfo.FileHash); fileExist != nil {
			if !fileExist.ValidFlag {
				log.Debug("[APP SDK] FsStoreFiles Delete old fileInfo")
				if !deleteFile(native, fileExist, &errInfos) {
					continue
				}
			} else {
				errInfos.AddObjectError(string(fileInfo.FileHash), "[APP SDK] FsStoreFiles File has stored!")
				log.Debug("[APP SDK] FsStoreFiles File has stored!")
				continue
			}
		}

		fileInfo.FileCost = 0
		fileInfo.ValidFlag = true
		fileInfo.TimeStart = uint64(native.Time)

		log.Debugf("[APP SDK] FsStoreFiles BlockCount:%d, PayAmount :%d\n", fileInfo.FileBlockCount, fileInfo.PayAmount)

		if fileInfo.StorageType == FileStorageTypeUseSpace {
			space := getAndUpdateSpaceInfo(native, fileInfo.FileOwner)
			if space == nil {
				errInfos.AddObjectError(string(fileInfo.FileHash), "[APP SDK] FsStoreFiles getAndUpdateSpaceInfo error!")
				continue
			}
			if !space.ValidFlag {
				errInfos.AddObjectError(string(fileInfo.FileHash), "[APP SDK] FsStoreFiles space timeExpired!")
				continue
			}
			if space.RestVol <= fileInfo.FileBlockCount*DefaultPerBlockSize {
				errInfos.AddObjectError(string(fileInfo.FileHash), "[APP SDK] FsStoreFiles RestVol is not enough error!")
				continue
			}
			space.RestVol -= fileInfo.FileBlockCount * DefaultPerBlockSize
			fileInfo.PdpInterval = space.PdpInterval
			addSpaceInfo(native, space)
		} else if fileInfo.StorageType == FileStorageTypeUseFile {
			fileInfo.PayAmount = calcTotalFilePayAmountByFile(&fileInfo, globalParam.GasPerKbForSaveWithFile)
			fileInfo.RestAmount = fileInfo.PayAmount

			err = appCallTransfer(native, utils.OngContractAddress, fileInfo.FileOwner, contract, fileInfo.PayAmount)
			if err != nil {
				errInfos.AddObjectError(string(fileInfo.FileHash), "[APP SDK] FsStoreFiles AppCallTransfer, transfer error!")
				continue
			}
		} else {
			errInfos.AddObjectError(string(fileInfo.FileHash), "[APP SDK] FsStoreFiles unknown StorageType!")
			continue
		}
		addFileInfo(native, &fileInfo)
		log.Infof("setFileOwner %s %s", fileInfo.FileHash, fileInfo.FileOwner.ToBase58())
		setFileOwner(native, fileInfo.FileHash, fileInfo.FileOwner)
	}

	errInfos.AddErrorsEvent(native)
	return utils.BYTE_TRUE, nil
}

func FsRenewFiles(native *native.NativeService) ([]byte, error) {
	contract := native.ContextRef.CurrentContext().ContractAddress

	var errInfos Errors
	var filesReNew FileReNewList
	filesReNewSrc := common.NewZeroCopySource(native.Input)
	filesReNewData, err := DecodeVarBytes(filesReNewSrc)
	if err != nil {
		return utils.BYTE_FALSE, errors.NewErr("[APP SDK] FsRenewFiles DecodeVarBytes error!")
	}

	source := common.NewZeroCopySource(filesReNewData)
	if err := filesReNew.Deserialization(source); err != nil {
		return utils.BYTE_FALSE, errors.NewErr("[APP SDK] FsRenewFiles Deserialization error!")
	}

	globalParam, err := getGlobalParam(native)
	if err != nil {
		return utils.BYTE_FALSE, errors.NewErr("[APP SDK] FsRenewFiles getGlobalParam error!")
	}

	for _, fileReNew := range filesReNew.FilesReNew {
		if !native.ContextRef.CheckWitness(fileReNew.Payer) {
			errInfos.AddObjectError(string(fileReNew.FileHash), "[APP SDK] FsRenewFiles CheckPayer failed!")
			continue
		}

		fileInfo := getAndUpdateFileInfo(native, fileReNew.FileOwner, fileReNew.FileHash)
		if fileInfo == nil {
			errInfos.AddObjectError(string(fileReNew.FileHash), "[APP SDK] FsRenewFiles getAndUpdateFileInfo error!")
			continue
		}

		if fileInfo.StorageType == FileStorageTypeUseFile {
			if !fileInfo.ValidFlag {
				errInfos.AddObjectError(string(fileReNew.FileHash), "[APP SDK] FsRenewFiles File is expired! need to upload again")
				continue
			}

			fileInfo.TimeExpired = fileReNew.NewTimeExpired
			newFee := calcTotalFilePayAmountByFile(fileInfo, globalParam.GasPerKbForSaveWithFile)
			if newFee < fileInfo.PayAmount {
				errInfos.AddObjectError(string(fileReNew.FileHash), "[APP SDK] FsRenewFiles newFee < fileInfo.PayAmount")
				continue
			}

			renewFee := newFee - fileInfo.PayAmount
			err = appCallTransfer(native, utils.OngContractAddress, fileReNew.Payer, contract, renewFee)
			if err != nil {
				errInfos.AddObjectError(string(fileReNew.FileHash), "[APP SDK] FsRenewFiles AppCallTransfer, transfer error!")
				continue
			}

			fileInfo.PayAmount = newFee
			fileInfo.RestAmount = fileInfo.RestAmount + renewFee
			addFileInfo(native, fileInfo)
		} else {
			errInfos.AddObjectError(string(fileReNew.FileHash), "[APP SDK] FsRenewFiles StorageType is not FileStorageTypeUseFile!")
		}
	}

	errInfos.AddErrorsEvent(native)
	return utils.BYTE_TRUE, nil
}

func FsDeleteFiles(native *native.NativeService) ([]byte, error) {
	var errInfos Errors
	var fileDelList FileDelList
	fileDelListSrc := common.NewZeroCopySource(native.Input)
	fileDelListData, err := DecodeVarBytes(fileDelListSrc)
	if err != nil {
		return utils.BYTE_FALSE, errors.NewErr("[APP SDK] FsDeleteFiles DecodeVarBytes error!")
	}
	source := common.NewZeroCopySource(fileDelListData)
	if err := fileDelList.Deserialization(source); err != nil {
		return utils.BYTE_FALSE, errors.NewErr("[APP SDK] FsDeleteFiles Deserialization error!")
	}

	for _, fileDel := range fileDelList.FilesDel {
		fileInfo := getFileInfoByHash(native, fileDel.FileHash)
		if fileInfo == nil {
			errInfos.AddObjectError(string(fileDel.FileHash), "[APP SDK] FsDeleteFiles fileInfo is nil")
			continue
		}

		if !native.ContextRef.CheckWitness(fileInfo.FileOwner) {
			errInfos.AddObjectError(string(fileDel.FileHash), "[APP SDK] FsDeleteFiles CheckFileOwner failed!")
			continue
		}
		deleteFile(native, fileInfo, &errInfos)

		//pdpRecordList := getPdpRecordList(native, fileInfo.FileHash, fileInfo.FileOwner)
		//
		//for _, pdpRecord := range pdpRecordList.PdpRecords {
		//	nodeInfo := getNodeInfo(native, pdpRecord.NodeAddr)
		//	if nodeInfo == nil {
		//		log.Error("[APP SDK] FsDeleteFiles getNodeInfo error")
		//		continue
		//	}
		//
		//	if !pdpRecord.SettleFlag {
		//		nodeInfo.RestVol += fileInfo.FileBlockCount * DefaultPerBlockSize
		//		addNodeInfo(native, nodeInfo)
		//		pdpRecord.SettleFlag = true
		//	}
		//}
		//
		//if fileInfo.StorageType == FileStorageTypeUseFile {
		//	err := appCallTransfer(native, utils.OngContractAddress, contract, fileInfo.FileOwner, fileInfo.RestAmount)
		//	if err != nil {
		//		log.Error("[APP SDK] FsDeleteFiles AppCallTransfer, transfer error!")
		//		continue
		//	}
		//} else if fileInfo.StorageType == FileStorageTypeUseSpace {
		//	space := getAndUpdateSpaceInfo(native, fileInfo.FileOwner)
		//	if space == nil {
		//		return utils.BYTE_FALSE, errors.NewErr("[APP SDK] FsDeleteFiles getAndUpdateSpaceInfo error!")
		//	}
		//	space.RestVol += fileInfo.FileBlockCount * DefaultPerBlockSize
		//	addSpaceInfo(native, space)
		//} else {
		//	log.Error("[APP SDK] FsDeleteFiles file StorageType error")
		//	continue
		//}
		//
		//delFileInfo(native, fileInfo.FileOwner, fileDel.FileHash)
		//delFileOwner(native, fileInfo.FileHash)
		//delPdpRecordList(native, fileDel.FileHash, fileInfo.FileOwner)
	}

	errInfos.AddErrorsEvent(native)
	return utils.BYTE_TRUE, nil
}

func deleteFile(native *native.NativeService, fileInfo *FileInfo, errInfos *Errors) bool {
	contract := native.ContextRef.CurrentContext().ContractAddress
	pdpRecordList := getPdpRecordList(native, fileInfo.FileHash, fileInfo.FileOwner)

	for _, pdpRecord := range pdpRecordList.PdpRecords {
		nodeInfo := getNodeInfo(native, pdpRecord.NodeAddr)
		if nodeInfo == nil {
			errInfos.AddObjectError(string(fileInfo.FileHash), "[APP SDK] DeleteFile getNodeInfo error")
			return false
		}

		if !pdpRecord.SettleFlag {
			nodeInfo.RestVol += fileInfo.FileBlockCount * DefaultPerBlockSize
			addNodeInfo(native, nodeInfo)
			pdpRecord.SettleFlag = true
		}
	}

	if fileInfo.StorageType == FileStorageTypeUseFile {
		err := appCallTransfer(native, utils.OngContractAddress, contract, fileInfo.FileOwner, fileInfo.RestAmount)
		if err != nil {
			errInfos.AddObjectError(string(fileInfo.FileHash), "[APP SDK] DeleteFile AppCallTransfer, transfer error!")
			return false
		}
	} else if fileInfo.StorageType == FileStorageTypeUseSpace {
		space := getAndUpdateSpaceInfo(native, fileInfo.FileOwner)
		if space == nil {
			errInfos.AddObjectError(string(fileInfo.FileHash), "[APP SDK] DeleteFile getAndUpdateSpaceInfo error!")
			return false
		}
		space.RestVol += fileInfo.FileBlockCount * DefaultPerBlockSize
		addSpaceInfo(native, space)
	} else {
		errInfos.AddObjectError(string(fileInfo.FileHash), "[APP SDK] DeleteFile file StorageType error")
		return false
	}

	delFileInfo(native, fileInfo.FileOwner, fileInfo.FileHash)
	delFileOwner(native, fileInfo.FileHash)
	delPdpRecordList(native, fileInfo.FileHash, fileInfo.FileOwner)
	return true
}

func FsTransferFiles(native *native.NativeService) ([]byte, error) {
	//Note: May cause storage node not to find PdpInfo, so when an error occurs,
	//the storage node needs to try to commit more than once

	var errInfos Errors
	var fileTransferList FileTransferList
	fileTransferListSrc := common.NewZeroCopySource(native.Input)
	fileTransferListData, err := DecodeVarBytes(fileTransferListSrc)
	if err != nil {
		return utils.BYTE_FALSE, errors.NewErr("[APP SDK] FsTransferFiles DecodeVarBytes error!")
	}
	source := common.NewZeroCopySource(fileTransferListData)
	if err := fileTransferList.Deserialization(source); err != nil {
		return utils.BYTE_FALSE, errors.NewErr("[APP SDK] FsTransferFiles OwnerChange Deserialization error!")
	}

	for _, fileTransfer := range fileTransferList.FilesTransfer {
		if native.ContextRef.CheckWitness(fileTransfer.OriOwner) == false {
			errInfos.AddObjectError(string(fileTransfer.FileHash), "[APP SDK] FsTransferFiles CheckFileOwner failed!")
			continue
		}

		fileInfo := getAndUpdateFileInfo(native, fileTransfer.OriOwner, fileTransfer.FileHash)
		if fileInfo == nil {
			errInfos.AddObjectError(string(fileTransfer.FileHash), "[APP SDK] FsTransferFiles GetFsFileInfo error!")
			continue
		}

		if !fileInfo.ValidFlag {
			errInfos.AddObjectError(string(fileTransfer.FileHash), "[APP SDK] FsTransferFiles File is expired!")
			continue
		}

		if fileInfo.StorageType != FileStorageTypeUseFile {
			errInfos.AddObjectError(string(fileTransfer.FileHash), "[APP SDK] FsTransferFiles file StorageType is not FileStorageTypeUseFile error!")
			continue
		}

		if fileInfo.FileOwner != fileTransfer.OriOwner {
			errInfos.AddObjectError(string(fileTransfer.FileHash), "[APP SDK] FsTransferFiles Caller is not file's owner!")
			continue
		}

		fileInfo.FileOwner = fileTransfer.NewOwner
		delFileInfo(native, fileTransfer.OriOwner, fileTransfer.FileHash)
		addFileInfo(native, fileInfo)

		pdpRecordList := getPdpRecordList(native, fileTransfer.FileHash, fileTransfer.OriOwner)
		for _, pdpInfo := range pdpRecordList.PdpRecords {
			delPdpRecord(native, pdpInfo.FileHash, pdpInfo.FileOwner, pdpInfo.NodeAddr)
			pdpInfo.FileOwner = fileTransfer.NewOwner
			addPdpRecord(native, &pdpInfo)
		}
		delFileOwner(native, fileInfo.FileHash)
		setFileOwner(native, fileInfo.FileHash, fileInfo.FileOwner)
	}

	errInfos.AddErrorsEvent(native)
	return utils.BYTE_TRUE, nil
}

func FsGetFileHashList(native *native.NativeService) ([]byte, error) {
	source := common.NewZeroCopySource(native.Input)
	passportData, err := DecodeVarBytes(source)
	if err != nil {
		return EncRet(false, []byte("[APP SDK] FsGetFileHashList DecodeVarBytes error!")), nil
	}

	walletAddr, err := CheckPassport(uint64(native.Height), passportData)
	if err != nil {
		errInfo := fmt.Sprintf("[APP SDK] FsGetFileHashList CheckFileListOwner error: %s", err.Error())
		return EncRet(false, []byte(errInfo)), nil
	}

	fileHashList := getFileHashList(native, walletAddr)
	sink := common.NewZeroCopySink(nil)
	fileHashList.Serialization(sink)

	return EncRet(true, sink.Bytes()), nil
}

func FsGetFileInfo(native *native.NativeService) ([]byte, error) {
	source := common.NewZeroCopySource(native.Input)
	fileHash, err := DecodeVarBytes(source)
	if err != nil {
		return EncRet(false, []byte("[APP SDK] FsGetFileInfo DecodeBytes error!")), nil
	}

	owner, err := getFileOwner(native, fileHash)
	if err != nil {
		return EncRet(false, []byte("[APP SDK] FsGetFileInfo getFileOwner error!")), nil
	}

	fileRawInfo := getFileRawRealInfo(native, owner, fileHash)
	return EncRet(true, fileRawInfo), nil
}

func FsGetPdpInfoList(native *native.NativeService) ([]byte, error) {
	source := common.NewZeroCopySource(native.Input)
	fileHash, err := DecodeVarBytes(source)
	if err != nil {
		return EncRet(false, []byte("[APP SDK] FsGetPdpInfoList DecodeBytes error!")), nil
	}

	owner, err := getFileOwner(native, fileHash)
	if err != nil {
		return EncRet(false, []byte("[APP SDK] FsGetPdpInfoList getFileOwner error!")), nil
	}

	pdpInfoList := getPdpRecordList(native, fileHash, owner)
	sink := common.NewZeroCopySink(nil)
	pdpInfoList.Serialization(sink)

	return EncRet(true, sink.Bytes()), nil
}

func FsSetWhiteList(native *native.NativeService) ([]byte, error) {
	var fileWhiteList FileWhiteList
	source := common.NewZeroCopySource(native.Input)
	if err := fileWhiteList.Deserialization(source); err != nil {
		return utils.BYTE_FALSE, errors.NewErr("[APP SDK] FsSetWhiteList Deserialization error!")
	}
	setWhiteList(native, fileWhiteList.FileOwner, fileWhiteList.FileHash, &fileWhiteList.WhiteListInfo)

	return utils.BYTE_TRUE, nil
}

func FsGetWhiteList(native *native.NativeService) ([]byte, error) {
	var fileWhiteList FileWhiteList
	source := common.NewZeroCopySource(native.Input)
	if err := fileWhiteList.Deserialization(source); err != nil {
		return EncRet(false, []byte("[APP SDK] FsGetWhiteList Deserialization error!")), nil
	}
	rawWhiteList := getRawWhiteList(native, fileWhiteList.FileOwner, fileWhiteList.FileHash)
	if rawWhiteList == nil {
		return EncRet(false, []byte("[APP SDK] FsGetWhiteList getRawWhiteList error")), nil
	}

	return EncRet(true, rawWhiteList), nil
}

func FsReadFilePledge(native *native.NativeService) ([]byte, error) {
	contract := native.ContextRef.CurrentContext().ContractAddress

	var err error
	var readPledge ReadPledge

	readPledgeSrc := common.NewZeroCopySource(native.Input)
	readPledgeSrcData, err := DecodeVarBytes(readPledgeSrc)
	if err != nil {
		return utils.BYTE_FALSE, errors.NewErr("[APP SDK] FsReadFilePledge DecodeVarBytes error!")
	}

	source := common.NewZeroCopySource(readPledgeSrcData)
	if err := readPledge.Deserialization(source); err != nil {
		return utils.BYTE_FALSE, errors.NewErr("[APP SDK] FsReadFilePledge deserialization error!")
	}

	globalParam, err := getGlobalParam(native)
	if err != nil {
		return utils.BYTE_FALSE, errors.NewErr("[APP SDK] FsReadFilePledge getGlobalParam error!")
	}

	fileInfo := getFileInfoByHash(native, readPledge.FileHash)
	if fileInfo == nil {
		return utils.BYTE_FALSE, errors.NewErr("[APP SDK] FsReadFilePledge getFsFileInfo error!")
	}

	if !fileInfo.ValidFlag {
		return utils.BYTE_FALSE, errors.NewErr("[APP SDK] FsReadFilePledge file out of date!")
	}

	//validation authority
	if !native.ContextRef.CheckWitness(readPledge.Downloader) {
		return utils.BYTE_FALSE, errors.NewErr("[APP SDK] FsReadFilePledge CheckDownloader failed!")
	}

	//oriPlan ==> newPlan
	var totalAddMaxBlockNumToRead uint64
	for index, readPlan := range readPledge.ReadPlans {
		totalAddMaxBlockNumToRead += readPlan.MaxReadBlockNum
		readPledge.ReadPlans[index].HaveReadBlockNum = 0
	}

	oriPledge, err := getReadPledge(native, readPledge.Downloader, readPledge.FileHash)
	if err == nil && oriPledge != nil {
		for _, oriReadPlan := range oriPledge.ReadPlans {
			foundSamePlan := false
			for index, readPlan := range readPledge.ReadPlans {
				if readPlan.NodeAddr == oriReadPlan.NodeAddr {
					foundSamePlan = true

					readPledge.ReadPlans[index].MaxReadBlockNum += oriReadPlan.MaxReadBlockNum
					readPledge.ReadPlans[index].HaveReadBlockNum = oriReadPlan.HaveReadBlockNum
				}
			}
			if !foundSamePlan {
				readPledge.ReadPlans = append(readPledge.ReadPlans, oriReadPlan)
			}
		}
		readPledge.RestMoney = oriPledge.RestMoney
		if uint64(native.Height) >= oriPledge.ExpireHeight {
			readPledge.BlockHeight = uint64(native.Height)
		} else {
			readPledge.BlockHeight = oriPledge.BlockHeight
		}
	} else {
		readPledge.RestMoney = 0
		readPledge.BlockHeight = uint64(native.Height)
	}
	readPledge.ExpireHeight = uint64(native.Height) + fileInfo.FileBlockCount + 30

	newPledgeFee := totalAddMaxBlockNumToRead * DefaultPerBlockSize * globalParam.GasPerKbForRead
	readPledge.RestMoney += newPledgeFee

	err = appCallTransfer(native, utils.OngContractAddress, readPledge.Downloader, contract, newPledgeFee)
	if err != nil {
		return utils.BYTE_FALSE, errors.NewErr("[APP SDK] FsReadFilePledge AppCallTransfer, transfer error!")
	}

	addReadPledge(native, &readPledge)
	return utils.BYTE_TRUE, nil
}

func FsGetReadPledge(native *native.NativeService) ([]byte, error) {
	var getPledge GetReadPledge
	source := common.NewZeroCopySource(native.Input)
	if err := getPledge.Deserialization(source); err != nil {
		return EncRet(false, []byte("[APP SDK] FsGetReadPledge Deserialization error!")), nil
	}

	rawPledge, err := getRawReadPledge(native, getPledge.Downloader, getPledge.FileHash)
	if err != nil {
		return EncRet(false, []byte("[APP SDK] FsGetReadPledge getRawReadPledge error!")), nil
	}
	return EncRet(true, rawPledge), nil
}

func FsCancelFileRead(native *native.NativeService) ([]byte, error) {
	contract := native.ContextRef.CurrentContext().ContractAddress

	var getPledge GetReadPledge
	source := common.NewZeroCopySource(native.Input)
	if err := getPledge.Deserialization(source); err != nil {
		return utils.BYTE_FALSE, errors.NewErr("[APP SDK] FsCancelFileRead GetReadPledge Deserialization error!")
	}

	readPledge, err := getReadPledge(native, getPledge.Downloader, getPledge.FileHash)
	if err != nil || readPledge == nil {
		return utils.BYTE_FALSE, errors.NewErr("[APP SDK] FsCancelFileRead getReadFilePledge error!")
	}

	if !native.ContextRef.CheckWitness(readPledge.Downloader) {
		return utils.BYTE_FALSE, errors.NewErr("[APP SDK] FsCancelFileRead CheckDownloader failed!")
	}

	if uint64(native.Height) < readPledge.ExpireHeight {
		return utils.BYTE_FALSE, errors.NewErr("[APP SDK] FsCancelFileRead FileReadPledge locked!")
	}

	if readPledge.RestMoney > 0 {
		err = appCallTransfer(native, utils.OngContractAddress, contract, readPledge.Downloader, readPledge.RestMoney)
		if err != nil {
			return utils.BYTE_FALSE, errors.NewErr("[APP SDK] FsCancelFileRead AppCallTransfer, transfer error!")
		}
	}

	delReadPledge(native, getPledge.Downloader, getPledge.FileHash)
	return utils.BYTE_TRUE, nil
}
