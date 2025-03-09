//
// Copyright 2017 Rackspace
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS-IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package cli

import (
	"errors"
	"fmt"
	"github.com/james4k/rcon"
	"github.com/peterh/liner"
	"io"
	"log"
	"os"
	"runtime"
	"strings"
)

const SectionSign = "§"
const Reset = "\u001B[0m"

var colors = map[string]string{
	"0": "\u001B[30m", // black
	"1": "\u001B[34m", // dark blue
	"2": "\u001B[32m", // dark green
	"3": "\u001B[36m", // dark aqua
	"4": "\u001B[31m", // dark red
	"5": "\u001B[35m", // dark purple
	"6": "\u001B[33m", // gold
	"7": "\u001B[37m", // gray
	"8": "\u001B[30m", // dark gray
	"9": "\u001B[34m", // blue
	"a": "\u001B[32m", // green
	"b": "\u001B[32m", // aqua
	"c": "\u001B[31m", // red
	"d": "\u001B[35m", // light purple
	"e": "\u001B[33m", // yellow
	"f": "\u001B[37m", // white
	"k": "",           // random
	"m": "\u001B[9m",  // strikethrough
	"o": "\u001B[3m",  // italic
	"l": "\u001B[1m",  // bold
	"n": "\u001B[4m",  // underline
	"r": Reset,        // reset
}

func Start(hostPort string, password string, out io.Writer) {
	remoteConsole, err := rcon.Dial(hostPort, password)
	if err != nil {
		log.Fatal("Failed to connect to RCON server", err)
	}
	defer remoteConsole.Close()

	lineEditor := liner.NewLiner()
	defer lineEditor.Close()

	for {
		cmd, err := lineEditor.Prompt("> ")

		if err != nil {
			if errors.Is(err, liner.ErrPromptAborted) {
				return
			}

			if errors.Is(err, io.EOF) {
				return
			}

			_, _ = fmt.Fprintln(os.Stderr, "Error reading input:", err)
		}

		if cmd == "exit" {
			return
		}

		reqId, err := remoteConsole.Write(cmd)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "Failed to send command:", err.Error())
			continue
		}

		resp, respReqId, err := remoteConsole.Read()
		if err != nil {
			if err == io.EOF {
				return
			}
			_, _ = fmt.Fprintln(os.Stderr, "Failed to read command:", err.Error())
			continue
		}

		if reqId != respReqId {
			_, _ = fmt.Fprintln(out, "Weird. This response is for another request.")
		}

		resp = colorize(resp)
		_, _ = fmt.Fprintln(out, resp)

		lineEditor.AppendHistory(cmd)
	}
}

func Execute(hostPort string, password string, out io.Writer, command ...string) {
	remoteConsole, err := rcon.Dial(hostPort, password)
	if err != nil {
		log.Fatal("Failed to connect to RCON server", err)
	}
	defer remoteConsole.Close()

	preparedCmd := strings.Join(command, " ")
	reqId, err := remoteConsole.Write(preparedCmd)

	resp, respReqId, err := remoteConsole.Read()
	if err != nil {
		if err == io.EOF {
			return
		}
		_, _ = fmt.Fprintln(os.Stderr, "Failed to read command:", err.Error())
		return
	}

	if reqId != respReqId {
		_, _ = fmt.Fprintln(out, "Weird. This response is for another request.")
	}

	resp = colorize(resp)
	_, _ = fmt.Fprintln(out, resp)
}

func colorize(str string) string {
	if runtime.GOOS == "windows" {
		return str
	}

	for code := range colors {
		str = strings.ReplaceAll(str, SectionSign+code, colors[code])
	}

	str = strings.ReplaceAll(str, "\n", "\n"+Reset)

	return str
}
