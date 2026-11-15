package charger

import (
	"encoding/binary"
	"io"
	"net"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func heidelbergTestBridge(t *testing.T, phases uint16, absent, failWrite bool) (modbus.Settings, <-chan []byte) {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = l.Close() })
	requests := make(chan []byte, 16)
	ctx := t.Context()

	go func() {
		c, err := l.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		go func() {
			<-ctx.Done()
			_ = c.Close()
		}()
		for {
			header := make([]byte, 7)
			if _, err := io.ReadFull(c, header); err != nil {
				return
			}
			n := binary.BigEndian.Uint16(header[4:])
			if n < 2 || n > 254 {
				return
			}
			req := make([]byte, n-1)
			if _, err := io.ReadFull(c, req); err != nil {
				return
			}
			requests <- req
			var res []byte
			switch {
			case len(req) == 8 && req[0] == 16:
				res = req[:5]
			case len(req) == 5 && req[0] == 3 && binary.BigEndian.Uint16(req[1:]) == hecRegTimeoutConfig:
				res = []byte{3, 2, 0, 0}
			case len(req) == 5 && req[0] == 3 && binary.BigEndian.Uint16(req[1:]) == hecRegFirmware:
				res = []byte{3, 2, byte(phases >> 8), byte(phases)}
				if absent {
					res = []byte{0x83, 2}
				}
			case len(req) == 5 && req[0] == 6 && binary.BigEndian.Uint16(req[1:]) == hecRegFirmware:
				res = req
				if failWrite {
					res = []byte{0x86, 4}
				}
			default:
				res = []byte{req[0] | 0x80, 1}
			}
			frame := binary.BigEndian.AppendUint16(append([]byte{}, header[:4]...), uint16(len(res)+1))
			frame = append(append(frame, header[6]), res...)
			if _, err := c.Write(frame); err != nil {
				return
			}
		}
	}()

	return modbus.Settings{URI: l.Addr().String(), ID: 1, Timeout: time.Second}, requests
}

func heidelbergRequest(t *testing.T, requests <-chan []byte, expected []byte) {
	t.Helper()
	select {
	case req := <-requests:
		assert.Equal(t, expected, req)
	case <-time.After(3 * time.Second):
		t.Fatal("missing Modbus request")
	}
}

func TestHeidelbergPhaseSwitchDetection(t *testing.T) {
	for _, tc := range []struct {
		name   string
		phases uint16
		absent bool
		want   bool
	}{
		{"one phase", 1, false, true},
		{"three phases", 3, false, true},
		{"zero", 0, false, false},
		{"two phases", 2, false, false},
		{"unexpected register", 0xffff, false, false},
		{"unmodified charger", 0, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sponsor.SetSubjectForTest(t, "test")
			settings, requests := heidelbergTestBridge(t, tc.phases, tc.absent, false)
			wb, err := NewHeidelbergEC(t.Context(), settings)
			require.NoError(t, err)
			_, ok := api.Cap[api.PhaseSwitcher](wb)
			assert.Equal(t, tc.want, ok)
			heidelbergRequest(t, requests, []byte{16, 1, 2, 0, 1, 2, 0, 4})
			heidelbergRequest(t, requests, []byte{3, 1, 1, 0, 1})
			heidelbergRequest(t, requests, []byte{3, 0, 4, 0, 1})
			assert.Empty(t, requests, "detection must not send phase-switch writes")
		})
	}
}

func TestHeidelbergPhaseSwitchWrite(t *testing.T) {
	for _, tc := range []struct {
		name      string
		phases    int
		failWrite bool
	}{
		{"one phase", 1, false},
		{"three phases", 3, false},
		{"bridge error", 3, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sponsor.SetSubjectForTest(t, "test")
			settings, requests := heidelbergTestBridge(t, 1, false, tc.failWrite)
			wb, err := NewHeidelbergEC(t.Context(), settings)
			require.NoError(t, err)
			for range 3 {
				<-requests
			}
			switcher, ok := api.Cap[api.PhaseSwitcher](wb)
			require.True(t, ok)
			err = switcher.Phases1p3p(tc.phases)
			if tc.failWrite {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			heidelbergRequest(t, requests, []byte{6, 0, 4, 0, byte(tc.phases)})
			assert.Empty(t, requests, "writes must not be retried or bypass the bridge")
		})
	}
}
