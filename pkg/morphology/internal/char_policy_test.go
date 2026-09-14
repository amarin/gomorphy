package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRussianCharPolicySubstitutesE(t *testing.T) {
	pol := RussianCharPolicy()

	to, ok := pol.Substitute('е')
	assert.True(t, ok)
	assert.Equal(t, 'ё', to)
}

func TestCharPolicyNeutralByDefault(t *testing.T) {
	pol := NewCharPolicy()
	_, ok := pol.Substitute('е')
	assert.False(t, ok, "пустой CharPolicy не подменяет символы")
}

func TestCharPolicyCustomSubstitution(t *testing.T) {
	pol := NewCharPolicy(Substitution{From: 'i', To: 'j'})

	to, ok := pol.Substitute('i')
	assert.True(t, ok)
	assert.Equal(t, 'j', to)

	_, ok = pol.Substitute('a')
	assert.False(t, ok)
}
