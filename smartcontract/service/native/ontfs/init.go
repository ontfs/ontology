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
	"github.com/ontio/ontology/common"
	"github.com/ontio/ontology/errors"
	"github.com/ontio/ontology/smartcontract/service/native"
	"github.com/ontio/ontology/smartcontract/service/native/utils"
)

func InitFs() {
	native.Contracts[utils.OntFSContractAddress] = RegisterFsContract
}

func RegisterFsContract(native *native.NativeService) {
	//native.Register(FS_SET_GLOBAL_PARAM, FsSetGlobalParam)
	native.Register(FS_GET_GLOBAL_PARAM, FsGetGlobalParam)

	native.Register(FS_NODE_REGISTER, FsNodeRegister)
	native.Register(FS_NODE_QUERY, FsNodeQuery)
	native.Register(FS_NODE_UPDATE, FsNodeUpdate)
	native.Register(FS_NODE_CANCEL, FsNodeCancel)
	native.Register(FS_FILE_PROVE, FsFileProve)
	native.Register(FS_NODE_WITH_DRAW_PROFIT, FsNodeWithDrawProfit)

	native.Register(FS_GET_NODE_LIST, FsGetNodeInfoList)
	native.Register(FS_GET_PDP_INFO_LIST, FsGetPdpInfoList)

	native.Register(FS_STORE_FILES, FsStoreFiles)
	native.Register(FS_RENEW_FILES, FsRenewFiles)
	native.Register(FS_DELETE_FILES, FsDeleteFiles)
	native.Register(FS_TRANSFER_FILES, FsTransferFiles)

	native.Register(FS_GET_FILE_INFO, FsGetFileInfo)
	native.Register(FS_GET_FILE_LIST, FsGetFileHashList)

	native.Register(FS_READ_FILE_PLEDGE, FsReadFilePledge)
	native.Register(FS_READ_FILE_SETTLE, FsReadFileSettle)
	native.Register(FS_GET_READ_PLEDGE, FsGetReadPledge)
	native.Register(FS_CANCEL_FILE_READ, FsCancelFileRead)

	native.Register(FS_SET_WHITE_LIST, FsSetWhiteList)
	native.Register(FS_GET_WHITE_LIST, FsGetWhiteList)

	native.Register(FS_CREATE_SPACE, FsCreateSpace)
	native.Register(FS_DELETE_SPACE, FsDeleteSpace)
	native.Register(FS_UPDATE_SPACE, FsUpdateSpace)
	native.Register(FS_GET_SPACE_INFO, FsGetSpaceInfo)
}

func FsSetGlobalParam(native *native.NativeService) ([]byte, error) {
	var globalParam FsGlobalParam
	infoSource := common.NewZeroCopySource(native.Input)
	if err := globalParam.Deserialization(infoSource); err != nil {
		return utils.BYTE_FALSE, errors.NewDetailErr(err, errors.ErrNoCode, "[FS Init] FsSetGlobalParam Deserialization error!")
	}
	setGlobalParam(native, &globalParam)
	return utils.BYTE_TRUE, nil
}

func FsGetGlobalParam(native *native.NativeService) ([]byte, error) {
	globalParam, err := getGlobalParam(native)
	if err != nil || globalParam == nil {
		return utils.BYTE_FALSE, errors.NewDetailErr(err, errors.ErrNoCode, "[FS Init] FsGetGlobalParam error!")
	}
	sink := common.NewZeroCopySink(nil)
	globalParam.Serialization(sink)

	return EncRet(true, sink.Bytes()), nil
}
