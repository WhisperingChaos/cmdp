package cmdp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Start_CommandsNotDefined(t *testing.T) {
	assrt := assert.New(t)
	var cmdef []Cdef
	shutdown, err := Start(cmdef)
	assrt.NotNil(err)
	assrt.Contains(err.Error(), "not defined")
	assrt.Nil(shutdown)
}

func Test_Start_ThenImmediatelyShutdown(t *testing.T) {
	assrt := assert.New(t)
	conCmds := []Cdef{
		{
			NmShort: "t",
			NmLong:  "test",
			Run:     runNone{},
			Parse:   ParseNone(),
			Help:    "Just a test function!",
		},
	}
	shutdown, err := Start(conCmds)
	assrt.Nil(err)
	assrt.NotNil(shutdown)
	shutdown <- true
	<-shutdown
}

type runNone struct{}

func (runNone) Run(args []string) error {
	return nil
}
