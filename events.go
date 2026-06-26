package main

import (
	"encoding/json"
	"os"
)

func emitRawJSONEvent(opt Options, event any) {
	if !opt.ProgressJSON {
		return
	}
	line, err := json.Marshal(event)
	if err != nil {
		return
	}
	line = append(line, '\n')
	if writeRawEventLine(line) {
		return
	}
	_, _ = os.Stdout.Write(line)
}
