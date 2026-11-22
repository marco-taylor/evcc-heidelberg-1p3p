package charger

import (
	"fmt"
	"sync"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/internal/heidelbergtest"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type heidelbergSafetySim struct{ *heidelbergtest.Controller }

func newHeidelbergSafetySim(t *testing.T, phases uint16) *heidelbergSafetySim {
	return &heidelbergSafetySim{heidelbergtest.New(t, phases)}
}
func (s *heidelbergSafetySim) charger(t *testing.T) api.Charger {
	t.Helper()
	wb, err := newHeidelbergEC(t.Context(), s.Settings(), true)
	require.NoError(t, err)
	return wb
}

func TestHeidelbergSafetyColdStart(t *testing.T) {
	sponsor.SetSubjectForTest(t, "test")
	s := newHeidelbergSafetySim(t, 0)
	wb := s.charger(t)
	getter, ok := api.Cap[api.PhaseGetter](wb)
	require.True(t, ok)
	switcher, ok := api.Cap[api.PhaseSwitcher](wb)
	require.True(t, ok)
	for _, op := range []func() error{func() error { _, e := wb.Status(); return e }, func() error { return wb.Enable(true) }, func() error { return wb.MaxCurrent(16) }, func() error { return switcher.Phases1p3p(3) }} {
		require.ErrorIs(t, op(), api.ErrPhaseStateUnknown)
	}
	n, e := getter.GetPhases()
	require.ErrorIs(t, e, api.ErrPhaseStateUnknown)
	assert.Zero(t, n)
	_, writes, _ := s.Snapshot()
	assert.Empty(t, writes)
	require.NoError(t, wb.Enable(false))
	_, writes, _ = s.Snapshot()
	assert.Equal(t, [][2]uint16{{261, 0}}, writes)
	s.State(1, false)
	n, e = getter.GetPhases()
	require.NoError(t, e)
	assert.Equal(t, 1, n)
	require.NoError(t, wb.MaxCurrent(16))
	require.NoError(t, wb.Enable(true))
	current, _, _ := s.Snapshot()
	assert.EqualValues(t, 160, current)
}

func TestHeidelbergSafetyTwoControllers(t *testing.T) {
	sponsor.SetSubjectForTest(t, "test")
	a, b := newHeidelbergSafetySim(t, 1), newHeidelbergSafetySim(t, 3)
	wa, wb := a.charger(t), b.charger(t)
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for _, w := range []api.Charger{wa, wb} {
		wg.Go(func() {
			for range 10 {
				if e := w.MaxCurrent(16); e != nil {
					errs <- e
					return
				}
				if _, e := w.Enabled(); e != nil {
					errs <- e
					return
				}
			}
		})
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		require.NoError(t, e)
	}
	ca, _, _ := a.Snapshot()
	cb, _, _ := b.Snapshot()
	assert.EqualValues(t, 160, ca)
	assert.EqualValues(t, 160, cb)
	sa, _ := api.Cap[api.PhaseSwitcher](wa)
	require.ErrorIs(t, sa.Phases1p3p(3), api.ErrPhaseSwitchPending)
	require.ErrorIs(t, wa.Enable(true), api.ErrPhaseStateUnknown)
	require.NoError(t, wb.MaxCurrent(8))
	a.State(3, false)
	ga, _ := api.Cap[api.PhaseGetter](wa)
	n, e := ga.GetPhases()
	require.NoError(t, e)
	assert.Equal(t, 3, n)
	require.NoError(t, wa.MaxCurrent(10))
	ca, _, _ = a.Snapshot()
	cb, _, _ = b.Snapshot()
	assert.EqualValues(t, 100, ca)
	assert.EqualValues(t, 80, cb)
	a.State(0, true)
	_, e = ga.GetPhases()
	require.ErrorIs(t, e, api.ErrPhaseStateUnknown)
	require.ErrorIs(t, wa.Enable(true), api.ErrPhaseStateUnknown)
	require.NoError(t, wb.MaxCurrent(6))
	a.State(1, false)
	n, e = ga.GetPhases()
	require.NoError(t, e)
	assert.Equal(t, 1, n)
	require.NoError(t, wa.MaxCurrent(7))
	_, writes, connections := a.Snapshot()
	assert.Greater(t, connections, 1)
	for _, write := range writes {
		assert.NotEqualValues(t, 259, write[0], "no automatic unlock")
	}
}

func TestHeidelbergSafetyRestartAndInvalidFeedback(t *testing.T) {
	sponsor.SetSubjectForTest(t, "test")
	for _, invalid := range []uint16{0, 2, 0xffff} {
		t.Run(fmt.Sprint(invalid), func(t *testing.T) {
			s := newHeidelbergSafetySim(t, 3)
			wb := s.charger(t)
			require.NoError(t, wb.MaxCurrent(16))
			s.State(invalid, false)
			require.ErrorIs(t, wb.MaxCurrent(16), api.ErrPhaseStateUnknown)
			require.ErrorIs(t, wb.Enable(true), api.ErrPhaseStateUnknown)
			g, _ := api.Cap[api.PhaseGetter](wb)
			n, e := g.GetPhases()
			assert.Zero(t, n)
			require.ErrorIs(t, e, api.ErrPhaseStateUnknown)
			s.State(1, false)
			restarted := s.charger(t)
			g, _ = api.Cap[api.PhaseGetter](restarted)
			n, e = g.GetPhases()
			require.NoError(t, e)
			assert.Equal(t, 1, n)
		})
	}
}

func TestHeidelbergSafetyNoRemoteUnlock(t *testing.T) {
	sponsor.SetSubjectForTest(t, "test")
	s := newHeidelbergSafetySim(t, 1)
	s.SetStatus(10)
	wb := s.charger(t)
	_, e := wb.Status()
	require.ErrorIs(t, e, api.ErrPhaseStateUnknown)
	w, _ := api.Cap[api.Resurrector](wb)
	require.Error(t, w.WakeUp())
	_, writes, _ := s.Snapshot()
	for _, write := range writes {
		assert.NotEqualValues(t, 259, write[0])
	}
}

func TestHeidelbergSafetyInvalidPhaseWrite(t *testing.T) {
	sponsor.SetSubjectForTest(t, "test")
	s := newHeidelbergSafetySim(t, 1)
	wb := s.charger(t)
	sw, _ := api.Cap[api.PhaseSwitcher](wb)
	for _, n := range []int{-1, 0, 2, 4, 65537} {
		require.Error(t, sw.Phases1p3p(n))
	}
	_, writes, _ := s.Snapshot()
	assert.Empty(t, writes)
}

func TestHeidelbergSafetyConfig(t *testing.T) {
	sponsor.SetSubjectForTest(t, "test")
	s := newHeidelbergSafetySim(t, 0)
	wb, err := NewHeidelbergECFromConfig(t.Context(), map[string]any{"uri": s.Settings().URI, "phasefeedback": true})
	require.NoError(t, err)
	assert.True(t, api.HasCap[api.PhaseGetter](wb))
	_, writes, connections := s.Snapshot()
	assert.Empty(t, writes)
	assert.Zero(t, connections, "startup must not require an armed controller")
}

func TestHeidelbergSafetyConcurrentSwitches(t *testing.T) {
	sponsor.SetSubjectForTest(t, "test")
	a, b := newHeidelbergSafetySim(t, 1), newHeidelbergSafetySim(t, 3)
	wa, wb := a.charger(t), b.charger(t)
	sa, _ := api.Cap[api.PhaseSwitcher](wa)
	sb, _ := api.Cap[api.PhaseSwitcher](wb)
	errors := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Go(func() { errors <- sa.Phases1p3p(3) })
	wg.Go(func() { errors <- sb.Phases1p3p(1) })
	wg.Wait()
	close(errors)
	for e := range errors {
		require.ErrorIs(t, e, api.ErrPhaseSwitchPending)
	}
	a.State(3, false)
	ga, _ := api.Cap[api.PhaseGetter](wa)
	gb, _ := api.Cap[api.PhaseGetter](wb)
	n, e := ga.GetPhases()
	require.NoError(t, e)
	assert.Equal(t, 3, n)
	n, e = gb.GetPhases()
	require.ErrorIs(t, e, api.ErrPhaseStateUnknown)
	assert.Zero(t, n)
	b.State(1, false)
	n, e = gb.GetPhases()
	require.NoError(t, e)
	assert.Equal(t, 1, n)
	require.NoError(t, wa.MaxCurrent(16))
	require.NoError(t, wb.MaxCurrent(16))
	require.NoError(t, wa.Enable(false))
	ca, _, _ := a.Snapshot()
	cb, _, _ := b.Snapshot()
	assert.Zero(t, ca)
	assert.EqualValues(t, 160, cb)
}

func TestHeidelbergSafetyPhaseWriteOutcome(t *testing.T) {
	sponsor.SetSubjectForTest(t, "test")
	for _, lostAck := range []bool{false, true} {
		t.Run(fmt.Sprint(lostAck), func(t *testing.T) {
			s := newHeidelbergSafetySim(t, 1)
			s.SetWriteFault(lostAck)
			wb := s.charger(t)
			sw, _ := api.Cap[api.PhaseSwitcher](wb)
			g, _ := api.Cap[api.PhaseGetter](wb)
			require.ErrorIs(t, sw.Phases1p3p(3), api.ErrPhaseStateUnknown)
			n, e := g.GetPhases()
			if lostAck {
				assert.Zero(t, n)
				require.ErrorIs(t, e, api.ErrPhaseStateUnknown)
				s.State(3, false)
			} else {
				require.NoError(t, e)
				assert.Equal(t, 1, n)
			}
			_, writes, _ := s.Snapshot()
			phaseWrites := 0
			for _, w := range writes {
				if w[0] == 4 {
					phaseWrites++
				}
			}
			assert.Equal(t, 1, phaseWrites, "uncertain writes must never be replayed")
			n, e = g.GetPhases()
			require.NoError(t, e)
			assert.Equal(t, map[bool]int{false: 1, true: 3}[lostAck], n)
		})
	}
}

func TestHeidelbergSafetyConcurrentAccess(t *testing.T) {
	sponsor.SetSubjectForTest(t, "test")
	s := newHeidelbergSafetySim(t, 1)
	wb := s.charger(t)
	var wg sync.WaitGroup
	errs := make(chan error, 3)
	wg.Go(func() {
		for range 20 {
			if err := wb.MaxCurrent(16); err != nil {
				errs <- err
				return
			}
		}
	})
	wg.Go(func() {
		for range 20 {
			if _, err := wb.Enabled(); err != nil {
				errs <- err
				return
			}
		}
	})
	wg.Go(func() {
		for range 20 {
			if _, err := wb.Status(); err != nil {
				errs <- err
				return
			}
		}
	})
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
}
