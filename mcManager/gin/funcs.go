package main

import (
	"bytes"
	log "github.com/sirupsen/logrus"
	"os/exec"
)

func shell(command string) (error, string) {
	log.Println(command)
	cmd := exec.Command("/bin/bash", "-c", command)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	return err, out.String()
}

func printPlayerMap() {
	out := "状态:"
	for _, player := range playerMap {
		out += "<" + player.playerName + ">:{online:"
		if player.ifOnline {
			out += "1"
		} else {
			out += "0"
		}
		out += ",sleep:"
		if player.ifSleep {
			out += "1"
		} else {
			out += "0"
		}
		out += "}"

	}
	log.Println(out)
}
