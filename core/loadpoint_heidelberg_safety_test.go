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
