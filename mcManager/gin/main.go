package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/hpcloud/tail"
	log "github.com/sirupsen/logrus"
)

func shell(command string) (string, error) {
	log.Println(command)
	cmd := exec.Command("/bin/bash", "-c", command)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	return out.String(), err
}

func printPlayerMap() {
	bytes, err := json.Marshal(playerMap)
	if err != nil {
		log.Println(err)
	}
	log.Println(string(bytes))
}

type playerStatus struct {
	playerName string
	ifSleep    bool
	ifOnline   bool
}

var playerMap map[string]playerStatus
var ifSleepVote bool

func main() {
	voteChan := make(chan playerStatus, 20)
	playerMap = make(map[string]playerStatus)

	logFile, err := os.OpenFile("gogogo.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		log.Println(err)
	}
	log.SetOutput(logFile)
	defer logFile.Close()

	result, err := shell("cd /root/Minecraft/rcon-0.10.2-amd64_linux/ && ./rcon --config=rcon.yaml list")
	if err != nil {
		log.Fatal(err)
		return
	}
	lists := strings.Split(result, ":")[1]
	names := strings.Split(lists, ",")
	for _, n := range names {
		//初始化
		playerMap[strings.TrimSpace(n)] = playerStatus{
			playerName: strings.TrimSpace(n),
			ifOnline:   true,
		}
	}
	printPlayerMap()

	fileName := "/root/Minecraft/Book/logs/latest.log"
	config := tail.Config{
		ReOpen:    true,
		Follow:    true,
		Location:  &tail.SeekInfo{Offset: 0, Whence: 2},
		MustExist: false,
		Poll:      true,
	}
	tails, err := tail.TailFile(fileName, config)
	if err != nil {
		log.Println("tail file failed, err:", err)
		return
	}
	var (
		line *tail.Line
		ok   bool
	)
	for {
		line, ok = <-tails.Lines
		if !ok {
			log.Printf("tail file close reopen, filename:%s\n", tails.Filename)
			time.Sleep(time.Second)
			continue
		}
		contents := strings.Fields(line.Text)
		if len(contents) <= 5 {
			continue
		}

		//登入登出
		if strings.Contains(line.Text, "left the game") {
			playerMap[strings.TrimSpace(contents[3])] = playerStatus{
				playerName: strings.TrimSpace(contents[3]),
				ifOnline:   false,
			}
			log.Println(contents[3], "下线")
			printPlayerMap()
		} else if strings.Contains(line.Text, "joined the game") {
			playerMap[strings.TrimSpace(contents[3])] = playerStatus{
				playerName: strings.TrimSpace(contents[3]),
				ifOnline:   true,
			}
			log.Println(contents[3], "上线")
			printPlayerMap()
		}

		//聊天
		var playerName string
		if strings.HasPrefix(contents[3], "<") && strings.HasSuffix(contents[3], ">") {
			playerName = contents[3][1 : len(contents[3])-1]
			log.Println(playerName, contents[4])

			if strings.HasPrefix(contents[4], "一起睡觉") {
				result, err = shell("cd /root/Minecraft/rcon-0.10.2-amd64_linux/ && ./rcon --config=rcon.yaml \"time query daytime\"")
				log.Println("查询时间：", result)
				if err != nil {
					log.Fatal(err)
					return
				}
				timeStr := strings.Fields(result)[3]
				timeInt, err := strconv.Atoi(timeStr)
				if err != nil {
					log.Fatal("字符转换失败", timeStr)
					return
				}
				if timeInt < 14000 {
					result, err = shell("cd /root/Minecraft/rcon-0.10.2-amd64_linux/ && ./rcon --config=rcon.yaml \"say 还不能睡觉哦\"")
					if err != nil {
						log.Fatal(err)
						return
					}
					log.Println(result)
					continue
				}

				//开启投票
				if !ifSleepVote {
					result, err = shell("cd /root/Minecraft/rcon-0.10.2-amd64_linux/ && ./rcon --config=rcon.yaml \"say 一起睡觉！\"")
					if err != nil {
						log.Fatal(err)
						return
					}
					log.Println(result)

					ifSleepVote = true
					go nightThrough(voteChan)
				}
				playerTmp := playerStatus{
					playerName: playerName,
					ifSleep:    true,
					ifOnline:   true,
				}
				//进行投票
				voteChan <- playerTmp
			}
		}
	}
}

func nightThrough(votes chan playerStatus) {
	for vote := range votes {
		log.Println(vote)
		playerMap[vote.playerName] = vote
		printPlayerMap()

		//统计睡觉玩家数量
		onlineNum := 0
		sleepNum := 0
		for _, player := range playerMap {
			if player.ifOnline {
				onlineNum++
				if player.ifSleep {
					sleepNum++
				}
			}
		}
		if onlineNum == 1 {
			result, err := shell("cd /root/Minecraft/rcon-0.10.2-amd64_linux/ && ./rcon --config=rcon.yaml \"say 1个人自生自灭吧\"")
			if err != nil {
				log.Fatal(err)
				return
			}
			log.Println(result)
			return
		}
		if onlineNum == 2 {
			result, err := shell("cd /root/Minecraft/rcon-0.10.2-amd64_linux/ && ./rcon --config=rcon.yaml \"say 2个人一起睡有点少\"")
			if err != nil {
				log.Fatal(err)
				return
			}
			log.Println(result)
			return
		}
		log.Println("sleepNum:"+strconv.Itoa(sleepNum), "onlineNum:"+strconv.Itoa(onlineNum))

		//在线玩家睡觉超过一半
		if sleepNum > (onlineNum / 2) {
			ifSleepVote = false
			//执行白天
			result, err := shell("cd /root/Minecraft/rcon-0.10.2-amd64_linux/ && ./rcon --config=rcon.yaml \"time set 1000\"")
			log.Println("执行白天：", result)
			if err != nil {
				log.Fatal(err)
				return
			}

			//置醒
			for _, player := range playerMap {
				player.ifSleep = false
				playerMap[player.playerName] = player
			}
			printPlayerMap()
			return
		}
	}
}
