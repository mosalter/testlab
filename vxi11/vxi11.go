/*
Copyright © 2022 Mark Salter

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
*/
package vxi11

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/rpc"

	"github.com/prashanthpai/sunrpc"
)

const (
	CHANNEL_CORE  uint32 = 395183
	CHANNEL_ABORT uint32 = 395184
	CHANNEL_IRQ   uint32 = 395185
)

type DeviceLink int32
type DeviceFlags int32
type DeviceErrCode int32

type DeviceErr struct {
	err DeviceErrCode
}

const (
	ErrNo           DeviceErrCode = 0
	ErrSyntax       DeviceErrCode = 1
	ErrNotAccepted  DeviceErrCode = 3
	ErrBadLinkId    DeviceErrCode = 4
	ErrParam        DeviceErrCode = 5
	ErrNoChan       DeviceErrCode = 6
	ErrNotSupported DeviceErrCode = 8
	ErrNoResource   DeviceErrCode = 9
	ErrLocked       DeviceErrCode = 11
	ErrNotLocked    DeviceErrCode = 12
	ErrIoTimeout    DeviceErrCode = 15
	ErrIo           DeviceErrCode = 17
	ErrBadAddr      DeviceErrCode = 21
	ErrAbort        DeviceErrCode = 23
	ErrChanExist    DeviceErrCode = 29
)

type genericParms struct {
	Lid         DeviceLink
	Flags       DeviceFlags
	LockTimeout uint32
	IoTimeout   uint32
}

type CreateLinkParms struct {
	ClientId    int32
	LockDevice  bool
	LockTimeout uint32
	Device      string
}

type CreateLinkResp struct {
	Err         DeviceErrCode
	Lid         DeviceLink
	AbortPort   uint16
	MaxRecvSize uint32
}

type Client struct {
	client *rpc.Client
}

func (c *Client) Close() error {
	return c.client.Close()
}

func (c *Client) CreateLink(devName string, clientId int32, lock bool, timeout uint32) (*Link, error) {
	parm := &CreateLinkParms{
		ClientId:    clientId,
		LockDevice:  lock,
		LockTimeout: timeout,
		Device:      devName,
	}
	resp := &CreateLinkResp{}

	err := c.client.Call("Client.CreateLink", parm, resp)
	if err != nil {
		return nil, err
	}

	if resp.Err != ErrNo {
		return nil, fmt.Errorf("CreateLink error %d", resp.Err)
	}

	link := &Link{
		client:      c,
		Id:          resp.Lid,
		AbortPort:   resp.AbortPort,
		MaxRecvSize: resp.MaxRecvSize,
	}

	return link, nil
}

type Link struct {
	client      *Client
	Id          DeviceLink
	AbortPort   uint16
	MaxRecvSize uint32
}

func (l *Link) Destroy() error {
	var deverr DeviceErr

	err := l.client.client.Call("Link.Destroy", &l.Id, &deverr)
	if err != nil {
		return err
	}
	if deverr.err != ErrNo {
		return fmt.Errorf("DestroyLink error %d", deverr)
	}
	return nil
}

type procedure struct {
	name string
	id   uint32
}

var coreProcs = []procedure{
	{"Client.CreateLink", 10},
	{"Link.Destroy", 23},
}

func FindPorts(host string) (ports []uint32, err error) {
	host += ":111"
	ports = make([]uint32, 3)
	ports[0], err = sunrpc.PmapGetPort(host, CHANNEL_CORE, 1, sunrpc.IPProtoTCP)
	if err != nil {
		fmt.Println(err)
		return
	}
	ports[1], err = sunrpc.PmapGetPort(host, CHANNEL_ABORT, 1, sunrpc.IPProtoTCP)
	if err != nil {
		fmt.Println(err)
		return
	}
	ports[2], err = sunrpc.PmapGetPort(host, CHANNEL_IRQ, 1, sunrpc.IPProtoTCP)
	if err != nil {
		fmt.Println(err)
	}
	return
}

func openConn(address string, port uint32) (*Client, error) {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", address, port))
	if err != nil {
		return nil, err
	}

	// Get notified on server closes the connection
	notifyClose := make(chan io.ReadWriteCloser, 5)
	go func() {
		for rwc := range notifyClose {
			conn := rwc.(net.Conn)
			log.Printf("Server %s disconnected", conn.RemoteAddr().String())
		}
	}()

	// Create client using sunrpc codec
	client := rpc.NewClientWithCodec(sunrpc.NewClientCodec(conn, notifyClose))

	return &Client{client: client}, nil
}

func registerProcs(prog uint32, ver uint32, procs []procedure) (err error) {
	for _, p := range procs {
		pid := sunrpc.Procedure{
			Name: p.name,
			ID: sunrpc.ProcedureID{
				ProgramNumber:   prog,
				ProgramVersion:  ver,
				ProcedureNumber: p.id,
			},
		}
		if err := sunrpc.RegisterProcedure(pid, true); err != nil {
			return err
		}
	}
	return nil
}

func DoTest(args []string) (err error) {
	var (
		client *Client
		link   *Link
	)

	if len(args) < 1 {
		return fmt.Errorf("No server address!")
	}

	ports, err := FindPorts(args[0])
	if err != nil {
		return
	}

	client, err = openConn(args[0], ports[0])
	if err != nil {
		return err
	}

	log.Println("Client created...")

	registerProcs(CHANNEL_CORE, 1, coreProcs)

	link, err = client.CreateLink("inst0", 1887, false, 0)
	if err != nil {
		client.Close()
		return err
	}

	log.Printf("Link ID: %d, abortPort: %d, maxRecv: %d\n",
		link.Id, link.AbortPort, link.MaxRecvSize)

	err = link.Destroy()
	if err != nil {
		client.Close()
		return err
	}

	return client.Close()
}
