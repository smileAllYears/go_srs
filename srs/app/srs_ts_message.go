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
	"go_srs/srs/utils"
)

/**
* the pid of ts packet,
* Table 2-3 - PID table, hls-mpeg-ts-iso13818-1.pdf, page 37
* NOTE - The transport packets with PID values 0x0000, 0x0001, and 0x0010-0x1FFE are allowed to carry a PCR.
 */
type SrsTsPid int

const (
	_ SrsTsPid = iota
	// Program Association Table(see Table 2-25).
	SrsTsPidPAT = 0x00
	// Conditional Access Table (see Table 2-27).
	SrsTsPidCAT = 0x01
	// Transport Stream Description Table
	SrsTsPidTSDT = 0x02
	// Reserved
	SrsTsPidReservedStart = 0x03
	SrsTsPidReservedEnd   = 0x0f
	// May be assigned as network_PID, Program_map_PID, elementary_PID, or for other purposes
	SrsTsPidAppStart = 0x10
	SrsTsPidAppEnd   = 0x1ffe
	// null packets (see Table 2-3)
	SrsTsPidNULL = 0x01FFF
)

/**
* the transport_scrambling_control of ts packet,
* Table 2-4 - Scrambling control values, hls-mpeg-ts-iso13818-1.pdf, page 38
 */
type SrsTsScrambled int

const (
	_ SrsTsScrambled = iota
	// Not scrambled
	SrsTsScrambledDisabled = 0x00
	// User-defined
	SrsTsScrambledUserDefined1 = 0x01
	// User-defined
	SrsTsScrambledUserDefined2 = 0x02
	// User-defined
	SrsTsScrambledUserDefined3 = 0x03
)

const SRS_TS_SYNC_BYTE = 0x47

const TS_PMT_NUMBER = 1
const TS_PMT_PID = 0x1001
const TS_VIDEO_AVC_PID = 0x100
const TS_AUDIO_AAC_PID = 0x101
const TS_AUDIO_MP3_PID = 0x102

type SrsTsPayload interface {
	Encode(stream *utils.SrsStream)
	Decode(stream *utils.SrsStream) error
	Size() uint32
}

type SrsTsMessage struct {
	channel        *SrsTsChannel
	packet         *SrsTsPacket
	writePcr       bool
	isDiscontinuty bool
	startPts       int64

	dts int64
	pts int64

	sid                SrsTsPESStreamId
	PES_packet_length  uint16
	continuity_counter uint8
	payload            []byte
}

func NewSrsTsMessage() *SrsTsMessage {
	return &SrsTsMessage{}
}

func (this *SrsTsMessage) IsAudio() bool {
	return ((this.sid >> 5) & 0x07) == SrsTsPESStreamIdAudioChecker
}

func (this *SrsTsMessage) IsVideo() bool {
	return ((this.sid >> 4) & 0x0f) == SrsTsPESStreamIdVideoChecker
}

func (this *SrsTsMessage) Fresh() bool {
	return len(this.payload) == 0
}

func (this *SrsTsMessage) Completed(payload_unit_start_indicator int8) bool {
	if this.PES_packet_length == 0 {
		if payload_unit_start_indicator == 1 {
			return true
		}
		return false
	}

	return len(this.payload) >= int(this.PES_packet_length)
}

func (this *SrsTsMessage) StartNumber() int {
	if this.IsAudio() {
		return int(this.sid) & 0x1f
	} else if this.IsVideo() {
		return int(this.sid) & 0x0f
	}
	return -1
}

func (this *SrsTsMessage) Dump(stream *utils.SrsStream, pnb_bytes *int) error {
	if stream.Empty() {
		return nil
	}
	return nil
	// // xB
	// int nb_bytes = stream->size() - stream->pos();
	// if (this.PES_packet_length > 0) {
	//     nb_bytes = srs_min(nb_bytes, PES_packet_length - payload->length());
	// }

	// if (nb_bytes > 0) {
	//     if (!stream->require(nb_bytes)) {
	//         ret = ERROR_STREAM_CASTER_TS_PSE;
	//         srs_error("ts: dump PSE bytes failed, requires=%dB. ret=%d", nb_bytes, ret);
	//         return ret;
	//     }

	//     payload->append(stream->data() + stream->pos(), nb_bytes);
	//     stream->skip(nb_bytes);
	// }

	// *pnb_bytes = nb_bytes;

	// return ret;
}
