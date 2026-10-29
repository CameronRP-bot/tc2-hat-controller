package comms

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTrapMessengerStatus(t *testing.T) {
	trap := &fakeTrap{lines: make(chan []byte, 16),
		responses: map[string]string{"STATUS": "STATUS_R"},
		payloads:  map[string]string{"STATUS_R": `{"State": "ARMED", "Enabled": false}`},
	}
	h := newTestHistory(t, 100)
	m := &TrapMessenger{write: trap.write, lines: trap.lines, pending: map[int]chan *Message{}, History: h}
	m.Start()

	status, err := m.Status()
	require.NoError(t, err)
	require.Equal(t, `{"State": "ARMED", "Enabled": false}`, status)

	// Older trap software NACKs it.
	trap.responses["STATUS"] = "NACK"
	_, err = m.Status()
	require.ErrorContains(t, err, "too old")

	// Polling the status isn't saved, unless it fails.
	trap.responses["STATUS"] = ""
	_, err = m.Status()
	require.Error(t, err)
	require.Equal(t, []string{"sent STATUS 3", "failed STATUS 3 write failed"}, historySummary(t, h))
}
