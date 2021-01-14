/*
SPDX-License-Identifier: BSD-3-Clause-Open-MPI
*/

package contract

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

var testContract = "../../contract/sample-contract.json"

func TestContract(t *testing.T) {
	fmt.Println("TestContract")
	spec, err := ReadContract(testContract)
	assert.NoError(t, err, "read sample contract should not throw error")
	assert.Equal(t, 1, len(spec.Contracts), "sample file should contain 1 contract")

	count := 0
	for _, tx := range spec.Contracts["demo-contract"].Transactions {
		if tx.Name == "transferMarble" {
			assert.Equal(t, 4, len(tx.Rules), "transferMarble should contain 4 rules")
			count++
		}
	}
	assert.Equal(t, 1, count, "transferMarble rules should have been tested")
}
