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
package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseIPAndPort(t *testing.T) {
	assert := assert.New(t)
	ip, port, err := ParseHostAndPort("1.0.0.1:1234")
	assert.Nil(err)
	assert.Equal(ip, "1.0.0.1")
	assert.Equal(port, uint16(1234))

	_, _, err = ParseHostAndPort("1.0.0.1:100234")
	assert.NotNil(err)
}
