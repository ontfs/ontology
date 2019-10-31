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
	"github.com/ontio/ontology/smartcontract/service/native/utils"
)

type PdpData struct {
	Version         uint64
	FileHash        []byte
	NodeAddr        common.Address
	MultiRes        []byte
	AddRes          []byte
	ChallengeHeight uint64
}

func (this *PdpData) Serialization(sink *common.ZeroCopySink) {
	utils.EncodeVarUint(sink, this.Version)
	sink.WriteVarBytes(this.FileHash)
	utils.EncodeAddress(sink, this.NodeAddr)
	sink.WriteVarBytes(this.MultiRes)
	sink.WriteVarBytes(this.AddRes)
	utils.EncodeVarUint(sink, this.ChallengeHeight)
}

func (this *PdpData) Deserialization(source *common.ZeroCopySource) error {
	var err error
	this.Version, err = utils.DecodeVarUint(source)
	if err != nil {
		return err
	}
	this.FileHash, err = DecodeVarBytes(source)
	if err != nil {
		return err
	}
	this.NodeAddr, err = utils.DecodeAddress(source)
	if err != nil {
		return err
	}
	this.MultiRes, err = DecodeVarBytes(source)
	if err != nil {
		return err
	}
	this.AddRes, err = DecodeVarBytes(source)
	if err != nil {
		return err
	}
	this.ChallengeHeight, err = utils.DecodeVarUint(source)
	if err != nil {
		return err
	}
	return nil
}
