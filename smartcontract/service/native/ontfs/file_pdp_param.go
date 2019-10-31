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
	"github.com/ontio/ontology/smartcontract/service/native/utils"
)

type PdpParam struct {
	Version uint64
	G       []byte
	G0      []byte
	PubKey  []byte
	FileId  []byte
}

func (this *PdpParam) Serialization(sink *common.ZeroCopySink) {
	utils.EncodeVarUint(sink, this.Version)
	sink.WriteVarBytes(this.G)
	sink.WriteVarBytes(this.G0)
	sink.WriteVarBytes(this.PubKey)
	sink.WriteVarBytes(this.FileId)
}

func (this *PdpParam) Deserialization(source *common.ZeroCopySource) error {
	var err error
	this.Version, err = utils.DecodeVarUint(source)
	if err != nil {
		return err
	}
	fmt.Printf("%d", this.Version)
	this.G, err = DecodeVarBytes(source)
	if err != nil {
		return err
	}
	this.G0, err = DecodeVarBytes(source)
	if err != nil {
		return err
	}
	this.PubKey, err = DecodeVarBytes(source)
	if err != nil {
		return err
	}
	this.FileId, err = DecodeVarBytes(source)
	if err != nil {
		return err
	}
	return nil
}
