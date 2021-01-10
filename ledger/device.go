package ledger

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/karalabe/hid"
)

const okResponse = 0x9000

func getDevice() (d *hid.Device, err error) {
	const vendorID = 0x2c97

	ds := hid.Enumerate(vendorID, 0)

	if len(ds) == 0 {
		return nil, errors.New("device not found")
	}

	return ds[0].Open()
}

// nolint:funlen,gocognit,gocyclo // TODO - Shorten the function length
func send(d io.ReadWriter, payload []byte) ([]byte, error) {
	const (
		Channel = 0x0101
		Tag     = 0x05
	)

	var (
		i      uint16
		p, r   []byte
		packet [64]byte
		resp   []byte
	)

	for i = 0; len(payload) > 0; i++ {
		p = packet[:]

		binary.BigEndian.PutUint16(p, Channel)

		p[2] = Tag

		binary.BigEndian.PutUint16(p[3:], i)

		p = p[5:]

		if i == 0 {
			binary.BigEndian.PutUint16(p, uint16(len(payload)))
			p = p[2:]
		}

		payload = payload[copy(p, payload):]

		if _, err := d.Write(packet[:]); err != nil {
			return nil, err
		}
	}

	for i = 0; i == 0 || len(r) > 0; i++ {
		n, err := d.Read(packet[:])
		if err != nil {
			return nil, err
		}

		err = errors.New("device read error")

		if n < len(packet) {
			return nil, err
		}

		p = packet[:]

		if binary.BigEndian.Uint16(p) != Channel {
			return nil, err
		}

		if p[2] != Tag {
			return nil, err
		}

		if binary.BigEndian.Uint16(p[3:]) != i {
			return nil, err
		}

		p = p[5:]

		if i == 0 {
			resp = make([]byte, binary.BigEndian.Uint16(p))
			p, r = p[2:], resp
		}

		r = r[copy(r, p):]
	}

	if sw := binary.BigEndian.Uint16(resp[len(resp)-2:]); sw != okResponse {
		return nil, fmt.Errorf("invalid status %x", sw)
	}

	return resp[:len(resp)-2], nil
}
