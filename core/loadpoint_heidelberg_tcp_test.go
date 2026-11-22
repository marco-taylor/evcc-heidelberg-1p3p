package core

import (
	"testing"

	evbus "github.com/asaskevich/EventBus"
	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger"
	"github.com/evcc-io/evcc/core/circuit"
	"github.com/evcc-io/evcc/internal/heidelbergtest"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeidelbergTwoLoadpointsOverTCP(t *testing.T) {
	sponsor.SetSubjectForTest(t, "test")
	a, b := heidelbergtest.New(t, 1), heidelbergtest.New(t, 3)
	point := func(s *heidelbergtest.Controller) *Loadpoint {
		c, err := charger.NewHeidelbergECFromConfig(t.Context(), map[string]any{"uri": s.Settings().URI, "phasefeedback": true, "delay": 0, "timeout": "100ms"})
		require.NoError(t, err)
		return &Loadpoint{log: util.NewLogger("dual-tcp"), bus: evbus.New(), clock: clock.NewMock(), charger: c, measuredPhases: 3}
	}
	p1, p2 := point(a), point(b)
	require.NoError(t, p1.syncChargerPhases(true))
	require.NoError(t, p2.syncChargerPhases(true))
	assert.Equal(t, 1, p1.GetPhases())
	assert.Equal(t, 3, p2.GetPhases())
	require.NoError(t, p1.scalePhases(3))
	assert.Zero(t, p1.GetPhases())
	assert.Equal(t, 3, p2.GetPhases())
	_, err := p1.getStatusChanges()
	require.ErrorIs(t, err, api.ErrPhaseStateUnknown)
	a.State(3, false)
	require.NoError(t, p1.syncChargerPhases(true))
	assert.Equal(t, 3, p1.GetPhases())
	b.State(0, true)
	require.ErrorIs(t, p2.syncChargerPhases(true), api.ErrPhaseStateUnknown)
	assert.Zero(t, p2.GetPhases())
	assert.Zero(t, p2.GetMeasuredPhases())
	assert.Equal(t, 3, p1.GetPhases())
	b.State(1, false)
	require.NoError(t, p2.syncChargerPhases(true))
	assert.Equal(t, 1, p2.GetPhases())
	require.NoError(t, p1.charger.MaxCurrent(16))
	require.NoError(t, p2.charger.MaxCurrent(16))
	meter1, ok := p1.charger.(api.Meter)
	require.True(t, ok)
	meter2, ok := p2.charger.(api.Meter)
	require.True(t, ok)
	p1.chargePower, err = meter1.CurrentPower()
	require.NoError(t, err)
	p2.chargePower, err = meter2.CurrentPower()
	require.NoError(t, err)
	assert.Equal(t, 11040.0, p1.chargePower)
	assert.Equal(t, 3680.0, p2.chargePower)
	// This is a synthetic test budget, not the installation's feeder rating.
	root, err := circuit.New(util.NewLogger("synthetic"), "test-only", 0, 12000, nil, 0)
	require.NoError(t, err)
	p1.circuit = root
	p2.circuit = root
	require.NoError(t, root.Update([]api.CircuitLoad{p1, p2}))
	allowed := root.ValidatePower(p1.chargePower, 11040)
	assert.LessOrEqual(t, allowed+p2.chargePower, 12000.0)
	restarted := point(a)
	require.NoError(t, restarted.syncChargerPhases(true))
	assert.Equal(t, 3, restarted.GetPhases())
}
