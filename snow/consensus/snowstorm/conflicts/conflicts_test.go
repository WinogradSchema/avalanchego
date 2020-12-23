// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package conflicts

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow/choices"
)

func TestNoConflicts(t *testing.T) {
	c := New()

	tx := &TestTx{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Processing,
		},
		TransitionV: &TestTransition{
			IDV: ids.GenerateTestID(),
		},
	}

	virtuous, err := c.IsVirtuous(tx)
	assert.NoError(t, err)
	assert.True(t, virtuous)

	conflicts, err := c.Conflicts(tx)
	assert.NoError(t, err)
	assert.Empty(t, conflicts)
}

func TestInputConflicts(t *testing.T) {
	c := New()

	inputIDs := []ids.ID{ids.GenerateTestID()}
	tx0 := &TestTx{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Processing,
		},
		TransitionV: &TestTransition{
			IDV:       ids.GenerateTestID(),
			InputIDsV: inputIDs,
		},
	}
	tx1 := &TestTx{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Processing,
		},
		TransitionV: &TestTransition{
			IDV:       ids.GenerateTestID(),
			InputIDsV: inputIDs,
		},
	}

	err := c.Add(tx0)
	assert.NoError(t, err)

	virtuous, err := c.IsVirtuous(tx1)
	assert.NoError(t, err)
	assert.False(t, virtuous)

	conflicts, err := c.Conflicts(tx1)
	assert.NoError(t, err)
	assert.NotEmpty(t, conflicts)
}

func TestOuterRestrictionConflicts(t *testing.T) {
	c := New()

	transitionID := ids.GenerateTestID()
	tx0 := &TestTx{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Processing,
		},
		TransitionV: &TestTransition{
			IDV: transitionID,
		},
	}
	tx1 := &TestTx{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Processing,
		},
		TransitionV: &TestTransition{
			IDV: ids.GenerateTestID(),
		},
		EpochV:        1,
		RestrictionsV: []ids.ID{transitionID},
	}

	err := c.Add(tx0)
	assert.NoError(t, err)

	virtuous, err := c.IsVirtuous(tx1)
	assert.NoError(t, err)
	assert.False(t, virtuous)

	conflicts, err := c.Conflicts(tx1)
	assert.NoError(t, err)
	assert.Len(t, conflicts, 1)
}

func TestInnerRestrictionConflicts(t *testing.T) {
	c := New()

	transitionID := ids.GenerateTestID()
	tx0 := &TestTx{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Processing,
		},
		TransitionV: &TestTransition{
			IDV: transitionID,
		},
	}
	tx1 := &TestTx{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Processing,
		},
		TransitionV: &TestTransition{
			IDV: ids.GenerateTestID(),
		},
		EpochV:        1,
		RestrictionsV: []ids.ID{transitionID},
	}

	err := c.Add(tx1)
	assert.NoError(t, err)

	virtuous, err := c.IsVirtuous(tx0)
	assert.NoError(t, err)
	assert.False(t, virtuous)

	conflicts, err := c.Conflicts(tx0)
	assert.NoError(t, err)
	assert.Len(t, conflicts, 1)
}

func TestAcceptNoConflicts(t *testing.T) {
	c := New()

	tx := &TestTx{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Processing,
		},
		TransitionV: &TestTransition{
			IDV: ids.GenerateTestID(),
		},
	}

	err := c.Add(tx)
	assert.NoError(t, err)

	toAccepts, toRejects := c.Updateable()
	assert.Empty(t, toAccepts)
	assert.Empty(t, toRejects)

	c.Accept(tx.ID())

	toAccepts, toRejects = c.Updateable()
	assert.Len(t, toAccepts, 1)
	assert.Empty(t, toRejects)
	assert.Empty(t, c.txs)
	assert.Empty(t, c.transitions)
	assert.Empty(t, c.utxos)
	assert.Empty(t, c.restrictions)
	assert.Empty(t, c.dependencies)
	assert.Empty(t, c.missingDependencies)

	toAccept := toAccepts[0]
	assert.Equal(t, tx.ID(), toAccept.ID())
}

func TestAcceptNoConflictsWithDependency(t *testing.T) {
	c := New()

	transitionID := ids.GenerateTestID()
	tx0 := &TestTx{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Processing,
		},
		TransitionV: &TestTransition{
			IDV: transitionID,
		},
	}
	tx1 := &TestTx{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Processing,
		},
		TransitionV: &TestTransition{
			IDV:           ids.GenerateTestID(),
			DependenciesV: []ids.ID{transitionID},
		},
	}

	err := c.Add(tx0)
	assert.NoError(t, err)

	err = c.Add(tx1)
	assert.NoError(t, err)

	toAccepts, toRejects := c.Updateable()
	assert.Empty(t, toAccepts)
	assert.Empty(t, toRejects)

	c.Accept(tx1.ID())

	toAccepts, toRejects = c.Updateable()
	assert.Empty(t, toAccepts)
	assert.Empty(t, toRejects)

	c.Accept(tx0.ID())

	toAccepts, toRejects = c.Updateable()
	assert.Len(t, toAccepts, 1)
	assert.Empty(t, toRejects)

	toAccept := toAccepts[0]
	assert.Equal(t, tx0.ID(), toAccept.ID())

	toAccepts, toRejects = c.Updateable()
	assert.Len(t, toAccepts, 1)
	assert.Empty(t, toRejects)
	assert.Empty(t, c.txs)
	assert.Empty(t, c.transitions)
	assert.Empty(t, c.utxos)
	assert.Empty(t, c.restrictions)
	assert.Empty(t, c.dependencies)
	assert.Empty(t, c.missingDependencies)

	toAccept = toAccepts[0]
	assert.Equal(t, tx1.ID(), toAccept.ID())
}

func TestNoConflictsNoEarlyAcceptDependency(t *testing.T) {
	c := New()

	transitionID := ids.GenerateTestID()
	tx0 := &TestTx{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Processing,
		},
		TransitionV: &TestTransition{
			IDV: transitionID,
		},
	}
	tx1 := &TestTx{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Processing,
		},
		TransitionV: &TestTransition{
			IDV:           ids.GenerateTestID(),
			DependenciesV: []ids.ID{transitionID},
		},
	}

	err := c.Add(tx0)
	assert.NoError(t, err)

	err = c.Add(tx1)
	assert.NoError(t, err)

	toAccepts, toRejects := c.Updateable()
	assert.Empty(t, toAccepts)
	assert.Empty(t, toRejects)

	c.Accept(tx0.ID())

	toAccepts, toRejects = c.Updateable()
	assert.Len(t, toAccepts, 1)
	assert.Empty(t, toRejects)

	toAccept := toAccepts[0]
	assert.Equal(t, tx0.ID(), toAccept.ID())

	toAccepts, toRejects = c.Updateable()
	assert.Empty(t, toAccepts)
	assert.Empty(t, toRejects)

	c.Accept(tx1.ID())

	toAccepts, toRejects = c.Updateable()
	assert.Len(t, toAccepts, 1)
	assert.Empty(t, toRejects)
	assert.Empty(t, c.txs)
	assert.Empty(t, c.transitions)
	assert.Empty(t, c.utxos)
	assert.Empty(t, c.restrictions)
	assert.Empty(t, c.dependencies)
	assert.Empty(t, c.missingDependencies)

	toAccept = toAccepts[0]
	assert.Equal(t, tx1.ID(), toAccept.ID())
}

func TestAcceptNoConflictsWithDependenciesAcrossMultipleRounds(t *testing.T) {
	c := New()

	transitionID0 := ids.GenerateTestID()
	tx0 := &TestTx{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Processing,
		},
		TransitionV: &TestTransition{
			IDV: transitionID0,
		},
	}
	transitionID1 := ids.GenerateTestID()
	tx1 := &TestTx{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Processing,
		},
		TransitionV: &TestTransition{
			IDV: transitionID1,
		},
	}
	tx2 := &TestTx{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Processing,
		},
		TransitionV: &TestTransition{
			IDV:           ids.GenerateTestID(),
			DependenciesV: []ids.ID{transitionID0, transitionID1},
		},
	}

	err := c.Add(tx0)
	assert.NoError(t, err)

	err = c.Add(tx1)
	assert.NoError(t, err)

	err = c.Add(tx2)
	assert.NoError(t, err)

	// Check that no transactions are mistakenly marked
	// as accepted/rejected
	toAccepts, toRejects := c.Updateable()
	assert.Empty(t, toAccepts)
	assert.Empty(t, toRejects)

	// Accept tx2 and ensure that it is marked
	// as conditionally accepted pending its
	// dependencies.
	c.Accept(tx2.ID())

	assert.Equal(t, c.conditionallyAccepted.Len(), 1)

	toAccepts, toRejects = c.Updateable()
	assert.Empty(t, toAccepts)
	assert.Empty(t, toRejects)
	assert.Equal(t, c.conditionallyAccepted.Len(), 1)

	// Accept tx1 and ensure that it is the only
	// transaction marked as accepted. Note: tx2
	// still requires tx0 to be accepted.
	c.Accept(tx1.ID())

	toAccepts, toRejects = c.Updateable()
	assert.Len(t, toAccepts, 1)
	assert.Empty(t, toRejects)

	toAccept := toAccepts[0]
	assert.Equal(t, tx1.ID(), toAccept.ID())

	// Ensure that additional call to updateable
	// does not return any new accepted/rejected txs.
	toAccepts, toRejects = c.Updateable()
	assert.Len(t, toAccepts, 0)
	assert.Len(t, toRejects, 0)

	// Accept tx0 and ensure that it is
	// returned from Updateable
	c.Accept(tx0.ID())

	toAccepts, toRejects = c.Updateable()
	assert.Len(t, toAccepts, 1)
	assert.Empty(t, toRejects)

	toAccept = toAccepts[0]
	assert.Equal(t, tx0.ID(), toAccept.ID())

	// tx2 should be returned by the subseqeuent call
	toAccepts, toRejects = c.Updateable()
	assert.Len(t, toAccepts, 1)
	assert.Empty(t, toRejects)

	toAccept = toAccepts[0]
	assert.Equal(t, tx2.ID(), toAccept.ID())

	assert.Empty(t, c.txs)
	assert.Empty(t, c.transitions)
	assert.Empty(t, c.utxos)
	assert.Empty(t, c.restrictions)
	assert.Empty(t, c.dependencies)
	assert.Empty(t, c.missingDependencies)
}
func TestAcceptRejectedDependency(t *testing.T) {
	c := New()

	transitionID := ids.GenerateTestID()
	inputIDs := []ids.ID{ids.GenerateTestID()}
	tx0 := &TestTx{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Processing,
		},
		TransitionV: &TestTransition{
			IDV:       transitionID,
			InputIDsV: inputIDs,
		},
	}
	tx1 := &TestTx{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Processing,
		},
		TransitionV: &TestTransition{
			IDV:       ids.GenerateTestID(),
			InputIDsV: inputIDs,
		},
	}
	tx2 := &TestTx{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Processing,
		},
		TransitionV: &TestTransition{
			IDV:           ids.GenerateTestID(),
			DependenciesV: []ids.ID{transitionID},
		},
	}

	err := c.Add(tx0)
	assert.NoError(t, err)

	err = c.Add(tx1)
	assert.NoError(t, err)

	err = c.Add(tx2)
	assert.NoError(t, err)

	toAccepts, toRejects := c.Updateable()
	assert.Empty(t, toAccepts)
	assert.Empty(t, toRejects)

	c.Accept(tx1.ID())

	toAccepts, toRejects = c.Updateable()
	assert.Len(t, toAccepts, 1)
	assert.Len(t, toRejects, 1)

	toAccept := toAccepts[0]
	assert.Equal(t, tx1.ID(), toAccept.ID())

	toReject := toRejects[0]
	assert.Equal(t, tx0.ID(), toReject.ID())

	toAccepts, toRejects = c.Updateable()
	assert.Empty(t, toAccepts)
	assert.Len(t, toRejects, 1)
	assert.Empty(t, c.txs)
	assert.Empty(t, c.transitions)
	assert.Empty(t, c.utxos)
	assert.Empty(t, c.restrictions)
	assert.Empty(t, c.dependencies)

	toReject = toRejects[0]
	assert.Equal(t, tx2.ID(), toReject.ID())
}

func TestAcceptRejectedEpochDependency(t *testing.T) {
	c := New()

	transitionID := ids.GenerateTestID()
	inputIDs := []ids.ID{ids.GenerateTestID()}
	tx0 := &TestTx{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Processing,
		},
		TransitionV: &TestTransition{
			IDV:       transitionID,
			InputIDsV: inputIDs,
		},
	}
	tx1 := &TestTx{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Processing,
		},
		TransitionV: &TestTransition{
			IDV:       transitionID,
			InputIDsV: inputIDs,
		},
	}
	tx2 := &TestTx{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Processing,
		},
		TransitionV: &TestTransition{
			IDV:       transitionID,
			InputIDsV: inputIDs,
		},
		EpochV: 1,
	}
	tx3 := &TestTx{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Processing,
		},
		TransitionV: &TestTransition{
			IDV:           ids.GenerateTestID(),
			DependenciesV: []ids.ID{transitionID},
		},
	}

	err := c.Add(tx0)
	assert.NoError(t, err)

	err = c.Add(tx1)
	assert.NoError(t, err)

	err = c.Add(tx2)
	assert.NoError(t, err)

	err = c.Add(tx3)
	assert.NoError(t, err)

	toAccepts, toRejects := c.Updateable()
	assert.Empty(t, toAccepts)
	assert.Empty(t, toRejects)

	c.Accept(tx2.ID())

	toAccepts, toRejects = c.Updateable()
	assert.Len(t, toAccepts, 1)
	assert.Len(t, toRejects, 3)
	assert.Empty(t, c.txs)
	assert.Empty(t, c.transitions)
	assert.Empty(t, c.utxos)
	assert.Empty(t, c.restrictions)
	assert.Empty(t, c.dependencies)
	assert.Empty(t, c.missingDependencies)
}

func TestRejectedRejectedDependency(t *testing.T) {
	c := New()

	transitionIDAX := ids.GenerateTestID()
	transitionIDAY := ids.GenerateTestID()
	transitionIDBX := ids.GenerateTestID()
	transitionIDBY := ids.GenerateTestID()

	inputIDA := ids.GenerateTestID()
	inputIDB := ids.GenerateTestID()

	//   A.X - A.Y
	//          |
	//   B.X - B.Y
	txAX := &TestTx{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Processing,
		},
		TransitionV: &TestTransition{
			IDV:       transitionIDAX,
			InputIDsV: []ids.ID{inputIDA, ids.GenerateTestID()},
		},
	}
	txAY := &TestTx{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Processing,
		},
		TransitionV: &TestTransition{
			IDV:       transitionIDAY,
			InputIDsV: []ids.ID{inputIDA},
		},
	}
	txBX := &TestTx{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Processing,
		},
		TransitionV: &TestTransition{
			IDV:       transitionIDBX,
			InputIDsV: []ids.ID{inputIDB},
		},
	}
	txBY := &TestTx{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Processing,
		},
		TransitionV: &TestTransition{
			IDV:           transitionIDBY,
			DependenciesV: []ids.ID{transitionIDAY},
			InputIDsV:     []ids.ID{inputIDB},
		},
	}

	err := c.Add(txAY)
	assert.NoError(t, err)

	err = c.Add(txAX)
	assert.NoError(t, err)

	err = c.Add(txBY)
	assert.NoError(t, err)

	err = c.Add(txBX)
	assert.NoError(t, err)

	toAccepts, toRejects := c.Updateable()
	assert.Empty(t, toAccepts)
	assert.Empty(t, toRejects)

	c.Accept(txBX.ID())

	toAccepts, toRejects = c.Updateable()
	assert.Len(t, toAccepts, 1)
	assert.Len(t, toRejects, 1)

	c.Accept(txAY.ID())

	toAccepts, toRejects = c.Updateable()
	assert.Len(t, toAccepts, 1)
	assert.Len(t, toRejects, 1)
	assert.Empty(t, c.txs)
	assert.Empty(t, c.transitions)
	assert.Empty(t, c.utxos)
	assert.Empty(t, c.restrictions)
	assert.Empty(t, c.dependencies)
	assert.Empty(t, c.missingDependencies)
}

func TestAcceptVirtuousRejectedDependency(t *testing.T) {
	c := New()

	transitionIDAX := ids.GenerateTestID()
	transitionIDAY := ids.GenerateTestID()
	transitionIDBX := ids.GenerateTestID()
	transitionIDBY := ids.GenerateTestID()
	inputIDsA := []ids.ID{ids.GenerateTestID()}
	inputIDsB := []ids.ID{ids.GenerateTestID()}

	//   A.X - A.Y
	//          |
	//   B.X - B.Y
	txAX := &TestTx{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Processing,
		},
		TransitionV: &TestTransition{
			IDV:       transitionIDAX,
			InputIDsV: inputIDsA,
		},
	}
	txAY := &TestTx{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Processing,
		},
		TransitionV: &TestTransition{
			IDV:       transitionIDAY,
			InputIDsV: inputIDsA,
		},
	}
	txBX := &TestTx{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Processing,
		},
		TransitionV: &TestTransition{
			IDV:       transitionIDBX,
			InputIDsV: inputIDsB,
		},
	}
	txBY := &TestTx{
		TestDecidable: choices.TestDecidable{
			IDV:     ids.GenerateTestID(),
			StatusV: choices.Processing,
		},
		TransitionV: &TestTransition{
			IDV:           transitionIDBY,
			DependenciesV: []ids.ID{transitionIDAY},
			InputIDsV:     inputIDsB,
		},
	}

	err := c.Add(txAX)
	assert.NoError(t, err)

	err = c.Add(txAY)
	assert.NoError(t, err)

	err = c.Add(txBX)
	assert.NoError(t, err)

	err = c.Add(txBY)
	assert.NoError(t, err)

	toAccepts, toRejects := c.Updateable()
	assert.Empty(t, toAccepts)
	assert.Empty(t, toRejects)

	c.Accept(txAX.ID())

	toAccepts, toRejects = c.Updateable()
	assert.Len(t, toAccepts, 1)
	assert.Len(t, toRejects, 1)

	toAccepts, toRejects = c.Updateable()
	assert.Len(t, toAccepts, 0)
	assert.Len(t, toRejects, 1)

	c.Accept(txBX.ID())

	toAccepts, toRejects = c.Updateable()
	assert.Len(t, toAccepts, 1)
	assert.Len(t, toRejects, 0)
	assert.Empty(t, c.txs)
	assert.Empty(t, c.transitions)
	assert.Empty(t, c.utxos)
	assert.Empty(t, c.restrictions)
	assert.Empty(t, c.dependencies)
	assert.Empty(t, c.missingDependencies)
}
