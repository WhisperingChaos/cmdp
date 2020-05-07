package cmdp

import (
	"bufio"
	"fmt"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/whisperingchaos/mckio"
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
func Test_ExecuteRunCmd(t *testing.T) {
	assrt := assert.New(t)
	var ra runAway
	ra.quit = make(chan bool)
	conCmds := []Cdef{
		{
			NmShort: "r",
			NmLong:  "run",
			ArgDesc: "<WhereTo>",
			Run:     ra,
			Parse:   ra,
			Help:    "Run a test!",
		},
	}
	raCmd := []string{
		"run home",
		"run Away",
	}
	rdr := mckio.NewRstrings(raCmd, newLineDelimBlock{})
	shutdown, err := start(conCmds, bufio.NewReader(&rdr))
	assrt.Nil(err)
	assrt.NotNil(shutdown)
	<-ra.quit
	shutdown <- true
	<-shutdown
}

type newLineDelimBlock struct{}

func (newLineDelimBlock) BehaviorDelim() []byte {
	return []byte{'\n'}
}
func (newLineDelimBlock) BehaviorBlockAtEnd() {
	select {}
}

type runAway struct {
	quit chan bool
}

func (runAway) Parse(cmdLn string) (args []string, err error) {
	rgx := regexp.MustCompile("(run|r)[[:blank:]](Away)$")
	args = rgx.FindStringSubmatch(cmdLn)
	if len(args) != 3 {
		return nil, fmt.Errorf("command syntax invalid: '%s'", cmdLn)
	}
	return args[2:3], nil
}
func (ra runAway) Run(args []string) error {
	defer close(ra.quit)
	if len(args) != 1 {
		return fmt.Errorf("missing <WhereTo> argument")
	}
	if args[0] != "Away" {
		return fmt.Errorf("<WhereTo> argument must be 'Away'")
	}
	return nil
}
