package core

import (
	"errors"
	"testing"

	evbus "github.com/asaskevich/EventBus"
	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestHeidelbergPhaseFeedbackSafety(t *testing.T) {
	communicationError := errors.New("simulated Modbus timeout")
	for _, tc := range []struct {
		name               string
		previous, reported int
		disabled           bool
		readErr            error
		want               int
	}{
		{name: "unknown after restart with unavailable feedback", readErr: api.ErrNotAvailable},
		{name: "retain last phase on invalid register", previous: 3, readErr: api.ErrNotAvailable, want: 3},
		{name: "retain last phase on communication error", previous: 3, readErr: communicationError, want: 3},
		{name: "disabled with communication error", previous: 3, disabled: true, readErr: communicationError, want: 3},
		{name: "seed from bridge after restart", reported: 1, want: 1},
		{name: "adopt bridge setpoint despite stale measurements", previous: 3, reported: 1, want: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			getter := api.NewMockPhaseGetter(ctrl)
			charger := struct {
				api.Charger
				api.PhaseSwitcher
				api.PhaseGetter
			}{api.NewMockCharger(ctrl), api.NewMockPhaseSwitcher(ctrl), getter}
			getter.EXPECT().GetPhases().Return(tc.reported, tc.readErr)
			lp := &Loadpoint{
				log: util.NewLogger("heidelberg-safety"), bus: evbus.New(), clock: clock.New(),
				charger: charger, phases: tc.previous, measuredPhases: 3,
			}
			err := lp.syncChargerPhases(tc.disabled)
			if tc.readErr == communicationError {
				require.ErrorIs(t, err, communicationError)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tc.want, lp.phases)
			// Strict mocks reject any switch, enable or current write from synchronization.
		})
	}
}

func TestHeidelbergConfirmedFeedbackInvalidation(t *testing.T) {
	for _, err := range []error{api.ErrPhaseStateUnknown, errors.Join(api.ErrPhaseStateUnknown, api.ErrNotAvailable), errors.Join(api.ErrPhaseStateUnknown, errors.New("Modbus timeout"))} {
		t.Run(err.Error(), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			getter := api.NewMockPhaseGetter(ctrl)
			charger := struct {
				api.Charger
				api.PhaseSwitcher
				api.PhaseGetter
			}{api.NewMockCharger(ctrl), api.NewMockPhaseSwitcher(ctrl), getter}
			getter.EXPECT().GetPhases().Return(0, err)
			lp := &Loadpoint{log: util.NewLogger("confirmed"), bus: evbus.New(), clock: clock.New(), charger: charger, phases: 3, measuredPhases: 3}
			_ = lp.syncChargerPhases(true)
			assert.Zero(t, lp.GetPhases())
			assert.Zero(t, lp.GetMeasuredPhases())
		})
	}
}

func TestHeidelbergPendingSwitchIsNotConfirmation(t *testing.T) {
	ctrl := gomock.NewController(t)
	cp := api.NewMockPhaseSwitcher(ctrl)
	charger := struct {
		api.Charger
		api.PhaseSwitcher
	}{api.NewMockCharger(ctrl), cp}
	cp.EXPECT().Phases1p3p(3).Return(errors.Join(api.ErrPhaseSwitchPending, api.ErrPhaseStateUnknown))
	lp := &Loadpoint{log: util.NewLogger("pending"), bus: evbus.New(), clock: clock.NewMock(), charger: charger, phases: 1, measuredPhases: 1}
	require.NoError(t, lp.scalePhases(3))
	assert.Zero(t, lp.GetPhases(), "acknowledged request is not a confirmed 3P configuration")
	assert.Zero(t, lp.GetMeasuredPhases())
	assert.Equal(t, lp.clock.Now(), lp.phasesSwitched)
}

func TestHeidelbergStatusErrorInvalidatesFeedback(t *testing.T) {
	ctrl := gomock.NewController(t)
	charger := api.NewMockCharger(ctrl)
	charger.EXPECT().Status().Return(api.StatusNone, errors.Join(api.ErrPhaseStateUnknown, errors.New("ESP32 reset")))
	lp := &Loadpoint{log: util.NewLogger("reset"), bus: evbus.New(), clock: clock.New(), charger: charger, phases: 3, measuredPhases: 3}
	_, err := lp.getStatusChanges()
	require.ErrorIs(t, err, api.ErrPhaseStateUnknown)
	assert.Zero(t, lp.GetPhases())
	assert.Zero(t, lp.GetMeasuredPhases())
}
