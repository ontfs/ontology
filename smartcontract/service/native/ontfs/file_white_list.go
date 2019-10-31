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
	"github.com/ontio/ontology/smartcontract/service/native"
	"github.com/ontio/ontology/smartcontract/service/native/utils"
)

type FileWhiteList struct {
	FileOwner     common.Address
	FileHash      []byte
	WhiteListInfo WhiteList
}

type WhiteList struct {
	UsersAddr []common.Address
}

func (this *WhiteList) Serialization(sink *common.ZeroCopySink) {
	ruleCount := uint64(len(this.UsersAddr))
	utils.EncodeVarUint(sink, ruleCount)
	for i := uint64(0); i < ruleCount; i++ {
		sink.WriteAddress(this.UsersAddr[i])
	}
}

func (this *WhiteList) Deserialization(source *common.ZeroCopySource) error {
	ruleCount, err := utils.DecodeVarUint(source)
	if err != nil {
		return err
	}
	for index := uint64(0); index < ruleCount; index++ {
		userAddr, err := utils.DecodeAddress(source)
		if err != nil {
			return err
		}
		this.UsersAddr = append(this.UsersAddr, userAddr)
	}
	return err
}

func (this *FileWhiteList) Serialization(sink *common.ZeroCopySink) {
	sinkTmp := common.NewZeroCopySink(nil)
	this.WhiteListInfo.Serialization(sinkTmp)

	sink.WriteAddress(this.FileOwner)
	sink.WriteVarBytes(this.FileHash)
	sink.WriteVarBytes(sinkTmp.Bytes())
}

func (this *FileWhiteList) Deserialization(source *common.ZeroCopySource) error {
	var err error
	if this.FileOwner, err = utils.DecodeAddress(source); err != nil {
		return err
	}
	if this.FileHash, err = DecodeVarBytes(source); err != nil {
		return err
	}
	var whiteList WhiteList
	if err = whiteList.Deserialization(source); err != nil {
		return err
	}
	this.WhiteListInfo.UsersAddr = whiteList.UsersAddr
	return err
}

func setWhiteList(native *native.NativeService, fileOwner common.Address, fileHash []byte, whiteList *WhiteList) {
	contract := native.ContextRef.CurrentContext().ContractAddress
	whiteListKey := GenFsWhiteListKey(contract, fileOwner, fileHash)

	sink := common.NewZeroCopySink(nil)
	whiteList.Serialization(sink)

	utils.PutBytes(native, whiteListKey, sink.Bytes())
}

func getRawWhiteList(native *native.NativeService, fileOwner common.Address, fileHash []byte) []byte {
	contract := native.ContextRef.CurrentContext().ContractAddress
	whiteListKey := GenFsWhiteListKey(contract, fileOwner, fileHash)

	item, err := utils.GetStorageItem(native, whiteListKey)
	if err != nil || item == nil || item.Value == nil {
		return nil
	}

	return item.Value
}

func getWhiteList(native *native.NativeService, fileOwner common.Address, fileHash []byte) *WhiteList {
	contract := native.ContextRef.CurrentContext().ContractAddress
	whiteListKey := GenFsWhiteListKey(contract, fileOwner, fileHash)

	item, err := utils.GetStorageItem(native, whiteListKey)
	if err != nil || item == nil || item.Value == nil {
		return nil
	}

	var whiteList WhiteList
	source := common.NewZeroCopySource(item.Value)
	if err := whiteList.Deserialization(source); err != nil {
		return nil
	}
	return &whiteList
}

func delWhiteList(native *native.NativeService, fileOwner common.Address, fileHash []byte) {
	contract := native.ContextRef.CurrentContext().ContractAddress
	fileInfoKey := GenFsFileInfoKey(contract, fileOwner, fileHash)
	native.CacheDB.Delete(fileInfoKey)
}
