package benchmark

import (
	"io"
	"testing"

	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func TestClient(t *testing.T) { TestingT(t) }

type MySuite struct {
	status int32
}

var _ = Suite(&MySuite{})

func (s *MySuite) SetUpSuite(c *C) {
}

func (s *MySuite) TearDownSuite(c *C) {
}

func (s *MySuite) SetUpTest(c *C) {
	s.status = Status_None
}

func (s *MySuite) TearDownTest(c *C) {
	s.status = Status_None
}

func (s *MySuite) TestHelloWorld(c *C) {
	c.Assert(42, Equals, 42)
	c.Assert(io.ErrClosedPipe, ErrorMatches, "io: .*on closed pipe")
	c.Check(42, Equals, 42)
}

func (s *MySuite) BenchmarkLogic(c *C) {
	for i := 0; i < c.N; i++ {
		a := 1
		a += 1
		c.Assert(42, Equals, 42)
	}
}
