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

	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/errors"
	"github.com/ontio/ontology/smartcontract/service/native"
	"github.com/ontio/ontology/smartcontract/service/native/ont"
	"github.com/ontio/ontology/smartcontract/service/native/utils"
)

const (
	FS_SET_GLOBAL_PARAM      = "FsSetGlobalParam"
	FS_GET_GLOBAL_PARAM      = "FsGetGlobalParam"
	FS_NODE_REGISTER         = "FsNodeRegister"
	FS_NODE_QUERY            = "FsNodeQuery"
	FS_NODE_UPDATE           = "FsNodeUpdate"
	FS_NODE_CANCEL           = "FsNodeCancel"
	FS_FILE_PROVE            = "FsFileProve"
	FS_NODE_WITH_DRAW_PROFIT = "FsNodeWithDrawProfit"
	FS_GET_NODE_LIST         = "FsGetNodeList"
	FS_GET_PDP_INFO_LIST     = "FsGetPdpInfoList"
	FS_STORE_FILES           = "FsStoreFiles"
	FS_RENEW_FILES           = "FsRenewFiles"
	FS_DELETE_FILES          = "FsDeleteFiles"
	FS_TRANSFER_FILES        = "FsTransferFiles"
	FS_GET_FILE_INFO         = "FsGetFileInfo"
	FS_GET_FILE_LIST         = "FsGetFileList"
	FS_READ_FILE_PLEDGE      = "FsReadFilePledge"
	FS_READ_FILE_SETTLE      = "FsReadFileSettle"
	FS_GET_READ_PLEDGE       = "FsGetReadPledge"
	FS_CANCEL_FILE_READ      = "FsCancelFileRead"
	FS_SET_WHITE_LIST        = "FsSetWhiteList"
	FS_GET_WHITE_LIST        = "FsGetWhiteList"
	FS_CREATE_SPACE          = "FsCreateSpace"
	FS_DELETE_SPACE          = "FsDeleteSpace"
	FS_UPDATE_SPACE          = "FsUpdateSpace"
	FS_GET_SPACE_INFO        = "FsGetSpaceInfo"
)

const (
	ONTFS_GLOBAL_PARAM     = "ontFsGlobalParam"
	ONTFS_NODE_INFO        = "ontFsNodeInfo"
	ONTFS_FILE_INFO        = "ontFsFileInfo"
	ONTFS_FILE_PDP         = "ontFsFilePdp"
	ONTFS_FILE_OWNER       = "ontFsFileOwner"
	ONTFS_FILE_WHITE_LIST  = "ontFsFileWhiteList"
	ONTFS_FILE_READ_PLEDGE = "ontFsFileReadPledge"
	ONTFS_FILE_SPACE       = "ontFsFileSpace"
)

func GenGlobalParamKey(contract common.Address) []byte {
	return append(contract[:], ONTFS_GLOBAL_PARAM...)
}

func GenFsNodeInfoPrefix(contract common.Address) []byte {
	prefix := append(contract[:], ONTFS_NODE_INFO...)
	return prefix
}

func GenFsNodeInfoKey(contract common.Address, nodeAddr common.Address) []byte {
	prefix := GenFsNodeInfoPrefix(contract)
	return append(prefix, nodeAddr[:]...)
}

func GenFsFileInfoPrefix(contract common.Address, fileOwner common.Address) []byte {
	prefix := append(contract[:], ONTFS_FILE_INFO...)
	prefix = append(prefix, fileOwner[:]...)
	return prefix
}

func GenFsFileInfoKey(contract common.Address, fileOwner common.Address, fileHash []byte) []byte {
	prefix := GenFsFileInfoPrefix(contract, fileOwner)
	return append(prefix, fileHash...)
}

func GenFsPdpRecordPrefix(contract common.Address, fileHash []byte, fileOwner common.Address) []byte {
	prefix := append(contract[:], ONTFS_FILE_PDP...)
	prefix = append(prefix, fileHash...)
	prefix = append(prefix, fileOwner[:]...)
	return prefix
}

func GenFsPdpRecordKey(contract common.Address, fileHash []byte, fileOwner common.Address, nodeAddr common.Address) []byte {
	prefix := GenFsPdpRecordPrefix(contract, fileHash, fileOwner)
	return append(prefix, nodeAddr[:]...)
}

func GenFsWhiteListKey(contract common.Address, fileOwner common.Address, fileHash []byte) []byte {
	key := append(contract[:], ONTFS_FILE_WHITE_LIST...)
	key = append(key, fileOwner[:]...)
	return append(key, fileHash...)
}

func GenFsFileOwnerKey(contract common.Address, fileHash []byte) []byte {
	prefix := append(contract[:], ONTFS_FILE_OWNER...)
	return append(prefix, fileHash...)
}

func GenFsReadPledgeKey(contract common.Address, downloader common.Address, fileHash []byte) []byte {
	key := append(contract[:], ONTFS_FILE_READ_PLEDGE...)
	key = append(key[:], downloader[:]...)
	return append(key, fileHash[:]...)
}

func GenFsSpaceKey(contract common.Address, spaceOwner common.Address) []byte {
	key := append(contract[:], ONTFS_FILE_SPACE...)
	return append(key, spaceOwner[:]...)
}

func appCallTransfer(native *native.NativeService, contract common.Address, from common.Address, to common.Address, amount uint64) error {
	var sts []ont.State
	sts = append(sts, ont.State{
		From:  from,
		To:    to,
		Value: amount,
	})
	transfers := ont.Transfers{
		States: sts,
	}

	sink := common.NewZeroCopySink(nil)
	transfers.Serialization(sink)

	if _, err := native.NativeCall(contract, "transfer", sink.Bytes()); err != nil {
		return fmt.Errorf("appCallTransfer, appCall error: %v", err)
	}
	return nil
}

func DecodeVarBytes(source *common.ZeroCopySource) ([]byte, error) {
	var err error
	buf, _, irregular, eof := source.NextVarBytes()
	if eof {
		return utils.BYTE_FALSE, errors.NewDetailErr(err, errors.ErrNoCode, "serialization.ReadVarBytes, contract params deserialize error!")
	}
	if irregular {
		return utils.BYTE_FALSE, common.ErrIrregularData
	}
	return buf, err
}

func DecodeBool(source *common.ZeroCopySource) (bool, error) {
	var err error
	ret, irregular, eof := source.NextBool()
	if eof {
		return false, errors.NewDetailErr(err, errors.ErrNoCode, "serialization.ReadBool, contract params deserialize error!")
	}
	if irregular {
		return false, common.ErrIrregularData
	}
	return ret, err
}
