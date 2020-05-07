/*
Package cmdp implements rudimentdary,concurrent console Command Processor.
*/
package cmdp

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

/*
Cdef Command Definition presents the attributes employed to define a single command.
Create a static array of Cdef's to specify the command set recognized by a console.
*/
type Cdef struct {
	/*
		NmShort (optional) - A command name that offers a short form of NmLong.
		It must be unique, regardless of case, when compared to all NmShort and
		NmLong values. It must not contain spaces.
	*/
	NmShort string
	/*
		NmLong (required) - A command name.  It must be unique, regardless of case,
		when compared to all NmShort and NmLong values. It must not contain spaces,
	*/
	NmLong string
	/*
		ArgDesc (optional) - Describes, using a synatx like BNF, the format of arguments
		that follow a command.
	*/
	ArgDesc string
	/*
		Help (required) - Text describing a command that's displayed to the console when
		requested.
	*/
	Help string
	/*
		Parse (required) - A function that extracts a command's argument(s), if any, from the entered command line.
		This function is provided the entire, normalized command string.  Normalization includes
		trimming leading and trailing spaces, converting the command name to lower case characters,
		and eliminating all repeating spaces between a command and the first argument.
		If a command requires no arguments, then use the public convience function 'ParseNone'
		to specify a do nothing parse function.
	*/
	Parse Parser
	/*
		Run (required) - A function provided with all the parsed arguments that executes the command.
		Run binds the text command to its go counterpart using a closure.
	*/
	Run Runner
}

/*
Parser defines an interface which accepts the text of a newline delimited
command.  A parser verifies the format of a command and separates it into
tokens.  The returned tokens become arguments that are supplied to
a Runner. Errors returned from the parser are written to STDERR.
*/
type Parser interface {
	Parse(cmdln string) (args []string, err error)
}

/*
Runner accepts string forms of a command's arguments, returned by the Parser
and then executes the command.  Use a closure to bind the appropriate
state needed to submit commands to the "backend" processor.
Errors returned by the Run command are written to STDERR by the command
processor.
*/
type Runner interface {
	Run(args []string) error
}

/*
RunHelp is a convenience function that generates a help document for all
commands when it's executed. Associate it to your help command or
write your own.
*/
func RunHelp() Runner {
	return &helpRunner{}
}

/*
ParseNone is a convenience function for commands without arguments.
*/
func ParseNone() Parser {
	return pn{}
}

/*
Start ensures required command attributes have been specified.  Once validated
it begins a dialog between end user, console processor, and the backend that
executes commands.

Use the shutdown channel returned by starting the Command Processor to request
a cooperative shutdown by sending 'true'.  The Command Processor will close
the shutdown channel to signal its completion of a graceful shutdown.
*/
func Start(cds []Cdef) (shutdown chan bool, err error) {
	return start(cds, bufio.NewReader(os.Stdin))
}

//-----------------------------------------------------------------------------
//--                            private section                              --
//-----------------------------------------------------------------------------

// Created function to ease testing by allowing mock of STDIN
func start(cds []Cdef, rCmdLn *bufio.Reader) (shutdown chan bool, err error) {
	var cs *cmds
	cs, err = validate(cds)
	if err != nil {
		return nil, err
	}
	shutdown = make(chan bool)
	go processCmdLn(cs.cmmds, shutdown, rCmdLn)
	return shutdown, nil
}

type cdef struct {
	Cdef
	nmShortL string
	nmLongL  string
}

type cmds struct {
	cmmds []cdef
}

func validate(cds []Cdef) (cs *cmds, errs error) {
	if len(cds) < 1 {
		return nil, fmt.Errorf("commands not defined")
	}
	cs = &cmds{cmmds: make([]cdef, len(cds))}
	for i, cPub := range cds {
		if err := cdefVerify(cPub, strconv.Itoa(i)); err != nil {
			errs = errorsConcat(errs, err)
			continue
		}
		cs.cmmds[i].Cdef = cPub
		cs.cmmds[i].nmShortL = strings.ToLower(cPub.NmShort)
		cs.cmmds[i].nmLongL = strings.ToLower(cPub.NmLong)
		if hlp, ok := cs.cmmds[i].Run.(*helpRunner); ok {
			hlp.patch(cs)
		}
	}
	return cs, errs
}

type pn struct {
}

func (pn) Parse(string) ([]string, error) {
	return nil, nil
}

type helpRunner struct {
	cs *cmds
}

func (hr *helpRunner) Run(args []string) error {
	fmt.Println("Help :")
	for _, c := range hr.cs.cmmds {
		fmt.Printf("%s,%s %s\n", c.NmShort, c.NmLong, c.ArgDesc)
		fmt.Printf("    %s\n", c.Help)
	}
	return nil
}
func (hr *helpRunner) patch(cs *cmds) {
	hr.cs = cs
}

func cdefVerify(c Cdef, altNm string) (errs error) {
	cdefRef := c.NmLong
	if c.NmLong == "" {
		cdefRef = altNm
		errs = errorsConcat(errs, fmt.Errorf("Please specify long command name fo: %s", cdefRef))
	} else if len(c.NmShort) > len(c.NmLong) {
		errs = errorsConcat(errs, fmt.Errorf("Please specify a short name whose length doesn't exceed its corresponding long one for: %s", cdefRef))
	}
	if c.Parse == nil {
		errs = errorsConcat(errs, fmt.Errorf("Please specify a Parse function for command: %s", cdefRef))
	}
	if c.Run == nil {
		errs = errorsConcat(errs, fmt.Errorf("Please specify a Run function for command: %s", cdefRef))
	}
	if c.Help == "" {
		errs = errorsConcat(errs, fmt.Errorf("Please specify a Help text for command: %s", cdefRef))
	}
	return errs
}
func errorsConcat(errs error, err error) (errcat error) {
	return fmt.Errorf("%s%s\n", errs, err)
}
func processCmdLn(cmds []cdef, shutdown chan bool, rCmdLn *bufio.Reader) {
	defer close(shutdown)
	resp := responseConfig(rCmdLn)
	for {
		select {
		case cmdLn, ok := <-resp:
			if !ok {
				return
			}
			cmdParseRun(cmds, cmdLn)
		case sd := <-shutdown:
			if sd {
				return
			}
		}
	}
}
func responseConfig(rCmdLn *bufio.Reader) (response <-chan string) {
	resp := make(chan string)
	go responseFetch(resp, rCmdLn)
	return resp
}
func responseFetch(resp chan<- string, rCmdLn *bufio.Reader) {
	defer close(resp)
	for {
		// general issue with golang when integrating blocking i/o
		// especially without deadline support.  This routing will
		// always survive shutdown request until main function
		// (goroutine) terminates.  When it does terminate, it will
		// issue an io.EOF error.  The code below ignores this error.
		cmdLn, err := rCmdLn.ReadString('\n')
		if err != nil {
			fmt.Printf("Abort: unexpected %s\n", err)
			break
		}
		// due to sequential nature of this code, and this
		// goroutine's deferred closure of the response channel
		// this select, if it eventually does execute after a shutdown,
		// should not generate a panic.
		resp <- cmdLn
	}
}
func cmdParseRun(cmds []cdef, cmdLn string) {
	cmdNm, cmdArg := cmdNormalize(cmdLn)
	cmd, err := cmdSelect(cmds, cmdNm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return
	}
	var args []string
	args, err = cmd.Parse.Parse(cmdNm + " " + cmdArg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return
	}
	if err := cmd.Run.Run(args); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return
	}
}
func cmdNormalize(cmdln string) (cmdNm string, cmdArg string) {
	cmdln = strings.TrimSpace(cmdln)
	cmdls := strings.SplitAfterN(cmdln, " ", 2)
	cmdNm = strings.TrimSpace(cmdls[0])
	cmdNm = strings.ToLower(cmdNm)
	if len(cmdls) > 1 {
		cmdArg = strings.TrimSpace(cmdls[1])
	}
	return cmdNm, cmdArg
}
func cmdSelect(cmds []cdef, cmdNm string) (cdef, error) {
	for _, c := range cmds {
		if cmdNm == c.nmShortL || cmdNm == c.nmLongL {
			return c, nil
		}
	}
	return cdef{}, fmt.Errorf("Error: unknown command: '%s' - try 'h' for help\n", cmdNm)
}
