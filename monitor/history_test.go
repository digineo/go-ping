package monitor

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const ms = time.Millisecond

var errTest = errors.New("i/o timeout")

func BenchmarkAddResult(b *testing.B) {
	h := NewHistory(8)
	for i := 0; i < b.N; i++ {
		h.AddResult(time.Duration(i), nil) // 1 allocc
	}
}

func BenchmarkCompute(b *testing.B) {
	h := NewHistory(8)
	for i := 0; i < b.N; i++ {
		h.AddResult(time.Duration(i), nil) // 1 alloc
		h.Compute()                        // 2 allocs
	}
}

func TestComputeEmpty(t *testing.T) {
	h := NewHistory(4)
	assert.Nil(t, h.Compute())
}

func TestComputeFailed(t *testing.T) {
	assert := assert.New(t)

	h := NewHistory(4)
	h.AddResult(2, errTest)

	metrics := h.Compute()
	assert.EqualValues(1, metrics.PacketsSent)
	assert.EqualValues(1, metrics.PacketsLost)
	assert.EqualValues(0, metrics.Best)
	assert.EqualValues(0, metrics.Worst)
	assert.EqualValues(0, metrics.Median)
	assert.EqualValues(0, metrics.Mean)
	assert.EqualValues(0, metrics.StdDev)
}

func TestComputeMedian(t *testing.T) {
	assert := assert.New(t)

	h := NewHistory(5)
	h.AddResult(300*ms, nil)
	h.AddResult(200*ms, nil)
	h.AddResult(100*ms, nil)
	h.AddResult(0, nil)
	assert.EqualValues(150*ms, h.Compute().Median)

	h.AddResult(400*ms, nil)
	assert.EqualValues(200*ms, h.Compute().Median)
}

func TestCompute(t *testing.T) {
	assert := assert.New(t)

	{ // populate with 5 entries
		h := NewHistory(8)
		h.AddResult(0, nil)
		h.AddResult(100*ms, nil)
		h.AddResult(100*ms, nil)
		h.AddResult(0, errTest)
		h.AddResult(100*ms, nil)

		assert.Equal(h.count, 5)
		assert.EqualValues(1, h.Compute().PacketsLost)
	}

	{
		// test zero variance
		h := NewHistory(8)
		h.AddResult(100*ms, nil)
		h.AddResult(100*ms, nil)
		h.AddResult(0, errTest)

		metrics := h.Compute()
		assert.EqualValues(100*ms, metrics.Best)
		assert.EqualValues(100*ms, metrics.Worst)
		assert.EqualValues(100*ms, metrics.Mean)
		assert.EqualValues(100*ms, metrics.Median)
		assert.EqualValues(0, metrics.StdDev)
		assert.EqualValues(3, metrics.PacketsSent)
		assert.EqualValues(1, metrics.PacketsLost)

		// results getting worse
		h.AddResult(200*ms, nil)
		h.AddResult(100*ms, nil)
		h.AddResult(0, errTest)

		metrics = h.Compute()
		assert.EqualValues(100*ms, metrics.Best)
		assert.EqualValues(200*ms, metrics.Worst)
		assert.EqualValues(125*ms, metrics.Mean)
		assert.EqualValues(100*ms, metrics.Median)
		assert.EqualValues(43301270, metrics.StdDev)
		assert.EqualValues(6, metrics.PacketsSent)
		assert.EqualValues(2, metrics.PacketsLost)

		// finally something better
		h.AddResult(0, nil)
		metrics = h.Compute()
		assert.EqualValues(0*ms, metrics.Best)
		assert.EqualValues(200*ms, metrics.Worst)
		assert.EqualValues(100*ms, metrics.Mean)
		assert.EqualValues(100*ms, metrics.Median)
		assert.EqualValues(63245553, metrics.StdDev)
		assert.EqualValues(7, metrics.PacketsSent)
		assert.EqualValues(2, metrics.PacketsLost)
	}
}

func TestHistoryCapacity(t *testing.T) {
	assert := assert.New(t)

	h := NewHistory(3)
	assert.Equal(h.count, 0)
	h.AddResult(1, nil)
	h.AddResult(2, errTest)
	assert.Equal(h.count, 2)
	assert.Equal(h.position, 2)
	h.AddResult(1, nil)
	assert.Equal(h.count, 3)
	assert.Equal(h.position, 0)

	h.AddResult(0, nil)
	assert.Equal(h.count, 3)
	assert.Equal(h.position, 1)
	assert.EqualValues(1, h.Compute().PacketsLost)

	// overwrite lost packet result
	h.AddResult(0, nil)
	assert.EqualValues(0, h.Compute().PacketsLost)

	// clear
	h.ComputeAndClear()
	assert.Equal(h.count, 0)
	assert.Equal(h.position, 0)
}
