package main

import (
	"bytes"
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
	log.Println("名字 睡觉 在线")
	for k, v := range playerMap {
		log.Println(k, v.Sleep, v.Online)
	}
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

	logFile, err := os.OpenFile("mcm.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
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
	log.Println("START")
	for {
		line, ok = <-tails.Lines
		if !ok {
			log.Printf("tail file close reopen, filename:%s\n", tails.Filename)
			time.Sleep(time.Second)
			continue
		}
		contents := strings.Fields(line.Text)
		if len(contents) < 5 {
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
					remainTimeToSleep := strconv.Itoa(int(float32(14000-timeInt) * 0.05))
					_, err = shell("cd /root/Minecraft/rcon-0.10.2-amd64_linux/ && ./rcon --config=rcon.yaml \"say 还要" + remainTimeToSleep + "秒才能睡哦！\"")
					if err != nil {
						log.Fatal(err)
						return
					}
					continue
				}
				//人数不够
				if onlineNum < 3 {
					_, err = shell("cd /root/Minecraft/rcon-0.10.2-amd64_linux/ && ./rcon --config=rcon.yaml \"say 3个人以上才开启！\"")
					if err != nil {
						log.Fatal(err)
						return
					}
					continue
				}

				//开启投票
				if !ifSleepVote {
					_, err = shell("cd /root/Minecraft/rcon-0.10.2-amd64_linux/ && ./rcon --config=rcon.yaml \"say 开始睡觉投票！(60s)\"")
					if err != nil {
						log.Fatal(err)
						return
					}

					ifSleepVote = true
					destroy := make(chan bool)
					go func() {
						time.Sleep(60 * time.Second)
						destroy <- true
					}()
					go nightThrough(voteChan, destroy)
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

func nightThrough(votes chan playerStatus, destroy chan bool) {
	for {
		select {
		case vote := <-votes:
			//投票
			playerMap[vote.PlayerName] = vote
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
				//执行白天
				result, err := shell("cd /root/Minecraft/rcon-0.10.2-amd64_linux/ && ./rcon --config=rcon.yaml \"time set 1000\"")
				log.Println("执行白天：", result)
				if err != nil {
					log.Fatal(err)
					return
				}
				_, err = shell("cd /root/Minecraft/rcon-0.10.2-amd64_linux/ && ./rcon --config=rcon.yaml \"say 天亮了！\"")
				if err != nil {
					log.Fatal(err)
					return
				}

				//置醒
				for _, player := range playerMap {
					player.Sleep = false
					playerMap[player.PlayerName] = player
				}
				ifSleepVote = false
				return
			}

		case <-destroy:
			//置醒
			for _, player := range playerMap {
				player.Sleep = false
				playerMap[player.PlayerName] = player
			}
			ifSleepVote = false
			_, err := shell("cd /root/Minecraft/rcon-0.10.2-amd64_linux/ && ./rcon --config=rcon.yaml \"say 投票过期！\"")
			if err != nil {
				log.Fatal(err)
			}
			return
		}
	}

}
