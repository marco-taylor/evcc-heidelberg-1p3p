package heidelbergtest

import (
	"encoding/binary"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/evcc-io/evcc/util/modbus"
	"github.com/stretchr/testify/require"
)

// Controller is an isolated Modbus TCP firmware simulator for integration tests.
type Controller struct {
	mu                                 sync.Mutex
	phases, requested, current, status uint16
	fail                               bool
	phaseException                     byte
	dropPhaseAck                       bool
	connections                        int
	writes                             [][2]uint16
	listener                           net.Listener
}

func New(t *testing.T, phases uint16) *Controller {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	s := &Controller{phases: phases, status: 4, listener: l}
	t.Cleanup(func() { _ = l.Close() })
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			s.mu.Lock()
			s.connections++
			s.mu.Unlock()
			go s.serve(c)
		}
	}()
	return s
}

func (s *Controller) serve(c net.Conn) {
	defer c.Close()
	for {
		h := make([]byte, 7)
		if _, err := io.ReadFull(c, h); err != nil {
			return
		}
		n := int(binary.BigEndian.Uint16(h[4:])) - 1
		if n < 1 || n > 253 {
			return
		}
		p := make([]byte, n)
		if _, err := io.ReadFull(c, p); err != nil {
			return
		}
		s.mu.Lock()
		if s.fail {
			s.mu.Unlock()
			return
		}
		var r []byte
		drop := false
		if len(p) >= 5 {
			fc, addr, count := p[0], binary.BigEndian.Uint16(p[1:]), binary.BigEndian.Uint16(p[3:])
			switch fc {
			case 3, 4:
				r = []byte{fc, byte(2 * count)}
				for reg := addr; reg < addr+count; reg++ {
					var value uint16
					if fc == 3 {
						switch reg {
						case 4:
							value = s.phases
						case 261:
							value = s.current
						case 259:
							value = 1
						}
					} else {
						switch reg {
						case 4:
							value = 0x108
						case 5:
							value = s.status
						case 6:
							value = s.current
						case 7, 8:
							if s.phases == 3 {
								value = s.current
							}
						case 10:
							value = 230
						case 11, 12:
							if s.phases == 3 {
								value = 230
							}
						case 14:
							value = uint16(230 * int(s.current) * int(s.phases) / 10)
						}
					}
					r = binary.BigEndian.AppendUint16(r, value)
				}
			case 6:
				s.writes = append(s.writes, [2]uint16{addr, count})
				if s.phaseException != 0 {
					r = []byte{0x86, s.phaseException}
					break
				}
				if addr == 4 {
					s.requested = count
					s.phases = 0
					drop = s.dropPhaseAck
					s.dropPhaseAck = false
				}
				r = append([]byte(nil), p...)
			case 16:
				if len(p) != int(6+count*2) {
					break
				}
				for i := uint16(0); i < count; i++ {
					reg, value := addr+i, binary.BigEndian.Uint16(p[6+2*i:])
					s.writes = append(s.writes, [2]uint16{reg, value})
					if reg == 261 {
						s.current = value
					}
				}
				r = append([]byte(nil), p[:5]...)
			}
		}
		s.mu.Unlock()
		if drop {
			return
		}
		if len(r) == 0 {
			r = []byte{p[0] | 0x80, 2}
		}
		frame := binary.BigEndian.AppendUint16(append([]byte(nil), h[:4]...), uint16(len(r)+1))
		frame = append(append(frame, h[6]), r...)
		if _, err := c.Write(frame); err != nil {
			return
		}
	}
}

func (s *Controller) Settings() modbus.Settings {
	return modbus.Settings{URI: s.listener.Addr().String(), ID: 1, Timeout: 100 * time.Millisecond}
}
func (s *Controller) State(phases uint16, fail bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.phases = phases
	s.fail = fail
}
func (s *Controller) Snapshot() (uint16, [][2]uint16, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.current, append([][2]uint16(nil), s.writes...), s.connections
}

func (s *Controller) SetStatus(status uint16) { s.mu.Lock(); defer s.mu.Unlock(); s.status = status }
func (s *Controller) SetWriteFault(lostAck bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if lostAck {
		s.dropPhaseAck = true
	} else {
		s.phaseException = 6
	}
}
