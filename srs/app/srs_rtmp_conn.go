/*
The MIT License (MIT)

Copyright (c) 2013-2015 GOSRS(gosrs)

Permission is hereby granted, free of charge, to any person obtaining a copy of
this software and associated documentation files (the "Software"), to deal in
the Software without restriction, including without limitation the rights to
use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
the Software, and to permit persons to whom the Software is furnished to do so,
subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/
package app

import (
	"net"
	"strings"
	"net/url"
	"errors"
	"go_srs/srs/protocol/rtmp"
	"go_srs/srs/protocol/packet"
	"go_srs/srs/app/config"
	"go_srs/srs/utils"
)

type SrsRtmpConn struct {
	rtmp 					*rtmp.SrsRtmpServer
	req						*SrsRequest
	res 					*SrsResponse
	server					*SrsServer
	source					*SrsSource
	clientType 				rtmp.SrsRtmpConnType
}

func NewSrsRtmpConn(c net.Conn, s *SrsServer) *SrsRtmpConn {
	rtmpConn := &SrsRtmpConn{
		req:NewSrsRequest(),
		res:NewSrsResponse(1),
		server:s,
	}
	rtmpConn.rtmp = rtmp.NewSrsRtmpServer(c, rtmpConn)
	return rtmpConn
}

func (this *SrsRtmpConn) Start() error {
	return this.doCycle()
}

func (this *SrsRtmpConn) Stop() {
	this.rtmp.Close()
	if this.req.typ == rtmp.SrsRtmpConnFMLEPublish || this.req.typ == rtmp.SrsRtmpConnFlashPublish || this.req.typ == rtmp.SrsRtmpConnHaivisionPublish {
		this.source.RemoveConsumers()
		RemoveSrsSource(this.source)
	}
}

/*
* @fun：关闭连接，由底层consumer或者source调用，将导致内部socket读取接口返回错误，从而回溯
*/
func (this *SrsRtmpConn) Close() {
	this.rtmp.Close()
}

func (this *SrsRtmpConn) doCycle() error {
	if err := this.rtmp.HandShake(); err != nil {
		return err
	}

	pkt, err := this.rtmp.ConnectApp()
	if err != nil {
		return err
	}
	
	err = pkt.(*packet.SrsConnectAppPacket).CommandObj.Get("tcUrl", &this.req.tcUrl)
	if err != nil {
		return err
	}

	_ = pkt.(*packet.SrsConnectAppPacket).CommandObj.Get("pageUrl", &this.req.pageUrl)
	_ = pkt.(*packet.SrsConnectAppPacket).CommandObj.Get("swfUrl", &this.req.swfUrl)
	_ = pkt.(*packet.SrsConnectAppPacket).CommandObj.Get("objectEncoding", &this.req.objectEncoding)
	u, err := url.Parse(this.req.tcUrl)
	this.req.schema = u.Scheme
	this.req.host = u.Host
	p := strings.Split(u.Path, "/")
	if len(p) >= 2 {
		this.req.app = p[1]
	}

	if len(p) >= 3 {
		this.req.stream = p[2]
	}

	m, _ := url.ParseQuery(u.RawQuery)
	this.req.vhost = this.req.host
	vhost, ok := m["vhost"]
	if ok {
		this.req.vhost = vhost[0]
	}

	this.serviceCycle()
	return nil
}

func (this *SrsRtmpConn) serviceCycle() error {
	err := this.rtmp.SetWindowAckSize((int32)(1000000))
	if err != nil {
		return err
	}

	err = this.rtmp.SetPeerBandwidth(1000*1000, 2)
	if err != nil {
		return err
	}

	this.req.ip = this.rtmp.GetClientIP()

	err = this.rtmp.SetChunkSize(config.GetInstance().GetChunkSize(this.req.vhost))
	if err != nil {
		return err
	}

	err = this.rtmp.ResponseConnectApp(this.req.objectEncoding)
	if err != nil {
		return err
	}

	err = this.rtmp.OnBwDone()
	if err != nil {
		return err
	}

	return this.streamServiceCycle()
}

func (this *SrsRtmpConn) streamServiceCycle() error {
	var err error
	this.req.typ, this.req.stream, this.req.duration, err = this.rtmp.IdentifyClient(this.res.StreamId)
	if err != nil {
		return err
	}


	this.req.schema, this.req.host, this.req.vhost, this.req.app, _, this.req.port, this.req.param, err = utils.SrsDiscoveryTcUrl(this.req.tcUrl, this.req.stream)

	if strings.Contains(this.req.stream, "?") {

		i := strings.Index(this.req.stream, "?")
		param := this.req.stream[i+1:]
		m, _ := url.ParseQuery(param)
		vhost_params, ok := m["vhost"]
		if ok {
			this.req.vhost = vhost_params[0]
		}
		this.req.stream = this.req.stream[0:i]
	}

	if err != nil {
		return errors.New("srs_discovery_tc_url failed")
	}
	//todo check edge vhost
	//todo security check

	if this.req.stream == "" {
		return errors.New("RTMP: Empty stream name not allowed")
	}

	this.source, err = FetchOrCreate(this, this.req, this.server)
	if err != nil {
		return err
	}

	this.clientType = this.req.typ

	switch(this.req.typ) {
	case rtmp.SrsRtmpConnPlay:{
		if err := this.rtmp.StartPlay(this.res.StreamId); err != nil {
			return err
		}

		if err := this.httpHooksOnPlay(); err != nil {
			return err
		}
		
		if err := this.playing(this.source); err != nil {
			return err
		}

		if err := this.httpHooksOnStop(); err != nil {
			return err
		}

		return nil
	}
	case rtmp.SrsRtmpConnFMLEPublish:{
		if err := this.rtmp.StartFmlePublish(0); err != nil {
			return err
		}
		return this.publishing(this.source)
	}
	default:{
		return errors.New("invalid client type")
	}
	//todo SrsRtmpConnHaivisionPublish,SrsRtmpConnFlashPublish
	}
	return nil
}

func (this *SrsRtmpConn) httpHooksOnPlay() error {
	vhost := config.GetInstance().GetVHost(this.req.vhost)
	if vhost == nil {
		return nil
	}

	if vhost.HttpHooks != nil && vhost.HttpHooks.Enabled == "on" {
		if err := OnPlay(vhost.HttpHooks.OnPlay, this.req); err != nil {
			return err
		}
	}
	return nil
}

func (this *SrsRtmpConn) httpHooksOnStop() error {
	vhost := config.GetInstance().GetVHost(this.req.vhost)
	if vhost == nil {
		return nil
	}

	if vhost.HttpHooks != nil && vhost.HttpHooks.Enabled == "on" {
		if err := OnStop(vhost.HttpHooks.OnStop, this.req); err != nil {
			return err
		}
	}
	return nil
}

func (this *SrsRtmpConn) playing( source *SrsSource) error {
	consumer := source.CreateConsumer(this, true, true, true)
	return this.doPlaying(source, consumer)
}

func (this *SrsRtmpConn) RemoveSelf() {
	this.server.RemoveConn(this)
}

func (this *SrsRtmpConn) OnRecvError(err error) {
	//判断如果是publish，则删除源
	this.Stop()
	this.server.OnRecvError(err, this)
}

func (this *SrsRtmpConn) doPlaying(source *SrsSource, consumer Consumer) error {
	//todo refer check
	//todo srsprint
	// realtime := false
	if err := consumer.ConsumeCycle(); err != nil {
		return err
	}
	return nil
}

func (this *SrsRtmpConn) publishing(s *SrsSource) error {
	//TODO
	//refer.check
	if err := this.httpHooksOnPublish(); err != nil {
		return err
	}
	//judge edge host
	if err := this.acquirePublish(s, false); err != nil {
		return err
	}

	err := this.doPublishing(s)
	//todo release publish
	if err := this.httpHooksOnUnpublish(); err != nil {
		return err
	}
	return err
}

func (this *SrsRtmpConn) httpHooksOnPublish() error {
	vhost := config.GetInstance().GetVHost(this.req.vhost)
	if vhost == nil {
		return nil
	}

	if vhost.HttpHooks != nil && vhost.HttpHooks.Enabled == "on" {
		if err := OnPublish(vhost.HttpHooks.OnPublish, this.req); err != nil {
			return err
		}
	}
	return nil
}

func (this *SrsRtmpConn) httpHooksOnUnpublish() error {
	vhost := config.GetInstance().GetVHost(this.req.vhost)
	if vhost == nil {
		return nil
	}

	if vhost.HttpHooks != nil && vhost.HttpHooks.Enabled == "on" {
		if err := OnPublish(vhost.HttpHooks.OnUnpublish, this.req); err != nil {
			return err
		}
	}
	return nil
}



func (this *SrsRtmpConn) acquirePublish(source *SrsSource, isEdge bool) error {
	//TODO edge process
	
	err := this.source.onPublish()
	if err != nil {
		return err
	}
	return nil
}

func (this *SrsRtmpConn) doPublishing(source *SrsSource) error {
	return source.CyclePublish()
}

func (this *SrsRtmpConn) Playing(source *SrsSource) {
	//todo
}
