/*
 * @Author: EagleXiang
 * @Github: https://github.com/eaglexiang
 * @Date: 2018-12-27 08:24:42
 * @LastEditors: EagleXiang
 * @LastEditTime: 2019-08-28 19:54:28
 */

package cmd

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/eaglexiang/eagle.tunnel.go/server/protocols/et/comm"
	"github.com/eaglexiang/go/logger"
	"github.com/eaglexiang/go/tunnel"
	"github.com/eaglexiang/go/version"
)

// Et-CHECK请求的类型
const (
	EtCheckUNKNOWN = iota
	EtCheckAUTH
	EtCheckPING
	EtCheckVERSION
	EtCheckUSERS
)

// ET-CHECK请求类型的文本
const (
	EtCheckUnknownTEXT = "UNKNOWN"
	EtCheckAuthTEXT    = "AUTH"
	EtCheckPingTEXT    = "PING"
	EtCheckVersionTEXT = "VERSION"
	EtCheckUsersTEXT   = "USERS"
)

// ETCheckTypes ET-CHECK的类型
var ETCheckTypes map[string]int

// ETCheckTypeTexts ET-CHECK类型的文本
var ETCheckTypeTexts map[int]string

func init() {
	ETCheckTypes = make(map[string]int)
	ETCheckTypes[EtCheckAuthTEXT] = EtCheckAUTH
	ETCheckTypes[EtCheckPingTEXT] = EtCheckPING
	ETCheckTypes[EtCheckVersionTEXT] = EtCheckVERSION
	ETCheckTypes[EtCheckUsersTEXT] = EtCheckUSERS

	ETCheckTypeTexts = make(map[int]string)
	ETCheckTypeTexts[EtCheckAUTH] = EtCheckAuthTEXT
	ETCheckTypeTexts[EtCheckPING] = EtCheckPingTEXT
	ETCheckTypeTexts[EtCheckVERSION] = EtCheckVersionTEXT
	ETCheckTypeTexts[EtCheckUSERS] = EtCheckUsersTEXT
}

// Check Check子协议
// 必须使用NewCheck进行初始化
type Check struct {
	handlers map[int]func(reqs []string, t *tunnel.Tunnel)
}

// NewCheck 初始化Check
func NewCheck() Check {
	handlers := make(map[int]func([]string, *tunnel.Tunnel))
	handlers[EtCheckPING] = handleEtCheckPingReq
	handlers[EtCheckVERSION] = handleEtCheckVersionReq
	handlers[EtCheckUSERS] = handleEtCheckUsersReq
	return Check{handlers: handlers}
}

// ParseEtCheckType 将字符串转换为EtCHECK请求的类型
func ParseEtCheckType(src string) int {
	src = strings.ToUpper(src)
	if v, ok := ETCheckTypes[src]; ok {
		return v
	}
	return EtCheckUNKNOWN
}

// formatEtCheckType 得到EtCHECK请求类型对应的字符串
func formatEtCheckType(src int) string {
	if t, ok := ETCheckTypeTexts[src]; ok {
		return t
	}
	return EtCheckUnknownTEXT
}

// Handle 处理ET-Check请求
func (c Check) Handle(req string, t *tunnel.Tunnel) error {
	reqs := strings.Split(req, " ")
	if len(reqs) < 2 {
		return errors.New("no value for et-check req")
	}
	theType := ParseEtCheckType(reqs[1])
	if h, ok := c.handlers[theType]; ok {
		h(reqs, t)
	} else {
		logger.Warning("et check type not found:", reqs[1])
		return errors.New("et check type not found")
	}
	return nil
}

// Type ET子协议的类型
func (c Check) Type() comm.CMDType {
	return comm.CHECK
}

// Name ET子协议的名字
func (c Check) Name() string {
	return comm.CHECKTxt
}

// SendEtCheckAuthReq 发射 ET-CHECK-AUTH 请求
func SendEtCheckAuthReq() string {
	// null代表未启用本地用户
	if comm.DefaultArg.LocalUser.ID == "null" {
		return "no local user"
	}

	// 当connect2Remote成功，则说明鉴权成功
	t := tunnel.GetTunnel()
	defer tunnel.PutTunnel(t)
	err := comm.Connect2Remote(t)
	if err != nil {
		return err.Error()
	}

	return "AUTH OK with local user: " + comm.DefaultArg.LocalUser.ID
}

// SendEtCheckVersionReq 发射 ET-CHECK-VERSION 请求
func SendEtCheckVersionReq() (reply string, err error) {
	req := comm.FormatEtType(comm.CHECK) + " " +
		formatEtCheckType(EtCheckVERSION) + " " +
		comm.ProtocolVersion.Raw
	return comm.SendQueryReq(req)
}

// SendEtCheckPingReq 发射ET-CHECK-PING请求
func SendEtCheckPingReq(sig chan string) {

	start := time.Now() // 开始计时

	req := comm.FormatEtType(comm.CHECK) + " " + formatEtCheckType(EtCheckPING)
	reply, err := comm.SendQueryReq(req)
	if err != nil {
		logger.Warning(err)
		return
	}
	if reply != "ok" {
		sig <- "invalid PING reply: " + reply
		return
	}

	duration := time.Since(start)
	ns := duration.Nanoseconds()
	ms := ns / 1000 / 1000
	sig <- strconv.FormatInt(ms, 10)
	return
}

func handleEtCheckPingReq(reqs []string, t *tunnel.Tunnel) {
	reply := "ok"

	t.WriteLeftStr(reply)
}

func handleEtCheckVersionReq(reqs []string, t *tunnel.Tunnel) {
	var reply string
	defer t.WriteLeftStr(reply)

	if len(reqs) < 3 {
		reply = "no protocol version value"
		return
	}
	versionOfReq, err := version.CreateVersion(reqs[2])
	if err != nil {
		reply = err.Error()
		return
	}
	if versionOfReq.IsLessThan(comm.ProtocolCompatibleVersion) {
		reply = "the version of protocol may be incompatible"
		return
	}
	reply = "Protocol Version OK"
}

// SendEtCheckUsersReq 发射 ET-CHECK-USERS 请求
func SendEtCheckUsersReq() (string, error) {
	req := comm.FormatEtType(comm.CHECK) + " " +
		formatEtCheckType(EtCheckUSERS)
	return comm.SendQueryReq(req)
}

func handleEtCheckUsersReq(reqs []string, t *tunnel.Tunnel) {
	var reply string
	for _, user := range comm.DefaultArg.ValidUsers {
		line := user.ID + ": " + user.Count()
		reply += line + "\n"
	}
	t.WriteLeftStr(reply)
}
