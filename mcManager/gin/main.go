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
	PlayerName string
	Sleep      bool
	Online     bool
}

var playerMap map[string]playerStatus
var ifSleepVote bool
var onlineNum int

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
			PlayerName: strings.TrimSpace(n),
			Online:     true,
		}
		onlineNum++
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
				PlayerName: strings.TrimSpace(contents[3]),
				Online:     false,
			}
			onlineNum--
			log.Println(contents[3], "下线")
			printPlayerMap()
		} else if strings.Contains(line.Text, "joined the game") {
			playerMap[strings.TrimSpace(contents[3])] = playerStatus{
				PlayerName: strings.TrimSpace(contents[3]),
				Online:     true,
			}
			onlineNum++
			log.Println(contents[3], "上线")
			printPlayerMap()
		}

		//聊天
		var playerName string
		if strings.HasPrefix(contents[3], "<") && strings.HasSuffix(contents[3], ">") {
			playerName = contents[3][1 : len(contents[3])-1]
			log.Println(playerName, "说", contents[4])

			if strings.HasPrefix(contents[4], "一起睡觉") {
				//查看时间
				result, err = shell("cd /root/Minecraft/rcon-0.10.2-amd64_linux/ && ./rcon --config=rcon.yaml \"time query daytime\"")
				if err != nil {
					log.Fatal(err)
					return
				}
				log.Println(result)
				timeStr := strings.Fields(result)[3]
				timeInt, err := strconv.Atoi(timeStr)
				if err != nil {
					log.Fatal("字符转换失败", timeStr)
					return
				}
				//没到点
				if timeInt < 14000 {
					result, err = shell("cd /root/Minecraft/rcon-0.10.2-amd64_linux/ && ./rcon --config=rcon.yaml \"say 还不能睡觉哦\"")
					if err != nil {
						log.Fatal(err)
						return
					}
					log.Println(result)
					continue
				}
				//人数不够
				if onlineNum <= 3 {
					result, err = shell("cd /root/Minecraft/rcon-0.10.2-amd64_linux/ && ./rcon --config=rcon.yaml \"say 3个人以上才开启\"")
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

				//进行投票
				playerTmp := playerStatus{
					PlayerName: playerName,
					Sleep:      true,
					Online:     true,
				}
				voteChan <- playerTmp
			}
		}
	}
}

func nightThrough(votes chan playerStatus) {
	for vote := range votes {
		//投票
		log.Println(vote)
		playerMap[vote.PlayerName] = vote
		printPlayerMap()

		//统计睡觉玩家数量
		online := 0
		sleepy := 0
		for _, player := range playerMap {
			if player.Online {
				online++
				if player.Sleep {
					sleepy++
				}
			}
		}
		log.Println("sleey:"+strconv.Itoa(sleepy), "online:"+strconv.Itoa(online))

		//在线玩家睡觉超过一半
		if sleepy > (online / 2) {
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
				player.Sleep = false
				playerMap[player.PlayerName] = player
			}
			printPlayerMap()
			return
		}
	}
}
