package client

import (
	"encoding/json"
	"fmt"
	"github.com/youpipe/go-youPipe/network"
	"github.com/youpipe/go-youPipe/service"
	"net"
)

func (c *Client) Proxying() {
	conn, err := c.proxyServer.Accept()
	if err != nil {
		fmt.Printf("finish to accept :%s", err)
		return
	}

	conn.(*net.TCPConn).SetKeepAlive(true)
	go c.consume(conn)
}

func (c *Client) consume(conn net.Conn) {
	defer conn.Close()

	obj, err := ProxyHandShake(conn)
	if err != nil {
		fmt.Println("sock5 handshake err:->", err)
		return
	}

	jsonConn, err := c.connectSockServer()
	if err != nil {
		return
	}

	if err := c.pipeHandshake(jsonConn, obj.target); err != nil {
		return
	}

	consumeConn := service.NewConsumerConn(jsonConn.Conn, c.aesKey)
	if consumeConn == nil {
		return
	}

	pipe := NewPipe(conn, consumeConn, c.payCh)

	go pipe.collectRequest()

	pipe.pullDataFromServer()
}

func (c *Client) connectSockServer() (*service.JsonConn, error) {

	port := c.curService.ID.ToServerPort()
	addr := network.JoinHostPort(c.curService.IP, port)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to (%s) access point server (%s):->", addr, err)

	}
	conn.(*net.TCPConn).SetKeepAlive(true)
	return &service.JsonConn{conn}, nil
}

func (c *Client) pipeHandshake(conn *service.JsonConn, target string) error {

	reqData := &service.PipeReqData{
		Addr:   c.Address.ToString(),
		Target: target,
	}

	data, err := json.Marshal(reqData)
	if err != nil {
		return fmt.Errorf("marshal hand shake data err:%v", err)
	}

	sig := c.Sign(data)

	hs := &service.YPHandShake{
		CmdType:     service.CmdPipe,
		Sig:         sig,
		PipeReqData: reqData,
	}

	if err := conn.WriteJsonMsg(hs); err != nil {
		return fmt.Errorf("write hand shake data err:%v", err)

	}
	ack := &service.YouPipeACK{}
	if err := conn.ReadJsonMsg(ack); err != nil {
		return fmt.Errorf("failed to read miner's response :->%v", err)
	}

	if !ack.Success {
		return fmt.Errorf("hand shake to miner err:%s", ack.Message)
	}

	return nil
}
