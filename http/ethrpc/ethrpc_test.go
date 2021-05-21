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
package ethrpc

import (
	"testing"
)

func TestEthRpc(t *testing.T) {
	//	calculator := new(EthereumAPI)
	//	server := rpc.NewServer()
	//	err := server.RegisterName("eth", calculator)
	//	assert.Nil(t, err)
	//
	//	testBaseString := "http://localhost:8545/"
	//
	//	putreq := httptest.NewRequest("GET", testBaseString, bytes.NewBuffer([]byte(`
	//{"id":"1620893182862","jsonrpc":"2.0","method":"eth_chainId","params":[]}
	//`)))
	//	putreq.Header.Add("content-type", "application/json")
	//	putrr := httptest.NewRecorder()
	//	server.ServeHTTP(putrr, putreq)
	//
	//	val, _ := io.ReadAll(putreq.Body)
	//
	//	fmt.Printf("%v \n", string(val))
}
