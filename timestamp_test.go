package logparser

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_containsTimestamp(t *testing.T) {
	assert.True(t, containsTimestamp("2005-08-09"))
	assert.True(t, containsTimestamp("2020/06/26"))
	assert.True(t, containsTimestamp("02/17/2009"))
	assert.True(t, containsTimestamp("25.02.2013"))
	assert.True(t, containsTimestamp("2013.25.02"))
	assert.True(t, containsTimestamp("18:31"))
	assert.True(t, containsTimestamp("18:31:42"))
	assert.True(t, containsTimestamp("18:31:42+03"))
	assert.True(t, containsTimestamp("18:31:42-03"))
	assert.True(t, containsTimestamp("18:31:42+03:30"))
	assert.True(t, containsTimestamp("18:31:42-03:30"))
	assert.True(t, containsTimestamp("2005-08-09T18:31:42"))
	assert.True(t, containsTimestamp("2005-08-09T18:31:42+03"))
	assert.True(t, containsTimestamp("2005-08-09T18:31:42-03"))
	assert.True(t, containsTimestamp("2005-08-09T18:31:42+03:30"))
	assert.True(t, containsTimestamp("2005-08-09T18:31:42-03:30"))
	assert.True(t, containsTimestamp("2005-08-09T18:31:42"))
	assert.True(t, containsTimestamp("2005-08-09T18:31:42.201"))
}
