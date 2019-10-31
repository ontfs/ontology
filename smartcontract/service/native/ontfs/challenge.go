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
	"bytes"
	"crypto/sha256"
	"encoding/binary"

	"github.com/ontio/ontology-crypto/pdp"
	"github.com/ontio/ontology/common"
)

func GenChallenge(nodeAddr common.Address, blockHash []byte, fileBlockNum, pdpBlockNum uint32) []pdp.Challenge {
	plant := append(nodeAddr[:], blockHash...)
	hash := sha256.Sum256(plant)

	tmpHash := make([]byte, common.UINT256_SIZE+4)
	copy(tmpHash, hash[:])
	copy(tmpHash[common.UINT256_SIZE:], hash[:4])
	var blockNumPerPart, blockNumLastPart, blockNumOfPart uint32

	if fileBlockNum <= 3 {
		blockNumPerPart = 1
		blockNumLastPart = 1
		blockNumOfPart = 1
		pdpBlockNum = fileBlockNum
	} else {
		if fileBlockNum > 3 && fileBlockNum < pdpBlockNum {
			pdpBlockNum = 3
		}
		blockNumPerPart = fileBlockNum / pdpBlockNum
		blockNumLastPart = blockNumPerPart + fileBlockNum%pdpBlockNum
		blockNumOfPart = blockNumPerPart
	}

	challenge := make([]pdp.Challenge, pdpBlockNum)

	var hashIndex = 0
	for i := uint32(1); i <= pdpBlockNum; i++ {
		if i == pdpBlockNum {
			blockNumOfPart = blockNumLastPart
		}

		rd := BytesToInt(tmpHash[hashIndex : hashIndex+4])
		challenge[i-1].Index = (rd+1)%blockNumOfPart + (i-1)*blockNumPerPart + 1
		challenge[i-1].Rand = uint32(blockHash[hashIndex]) + 1

		hashIndex++
		hashIndex = hashIndex % common.UINT256_SIZE
	}
	return challenge
}

func BytesToInt(b []byte) uint32 {
	var tmp uint32
	bytesBuffer := bytes.NewBuffer(b)
	binary.Read(bytesBuffer, binary.LittleEndian, &tmp)
	return tmp
}
