// Copyright 2023 Gravitational, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package types

import (
	"testing"

	"github.com/gravitational/trace"
	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/require"
)

func TestSessionTrackerV1_UpdatePresence(t *testing.T) {
	clock := clockwork.NewFakeClock()

	s, err := NewSessionTracker(SessionTrackerSpecV1{
		SessionID: "123",
		Participants: []Participant{
			{
				ID:         "1",
				User:       "llama",
				Mode:       string(SessionPeerMode),
				LastActive: clock.Now().UTC(),
			},
			{
				ID:         "2",
				User:       "fish",
				Mode:       string(SessionModeratorMode),
				LastActive: clock.Now().UTC(),
			},
		},
	})
	require.NoError(t, err)

	// Presence cannot be updated for a non-existent user
	err = s.UpdatePresence("alpaca")
	require.ErrorIs(t, err, trace.NotFound("participant alpaca not found"))

	// Update presence for just the user fish
	require.NoError(t, s.UpdatePresence("fish"))

	// Verify that user llama still has a LastActive time matching the
	// fake clock used by the test but that user fish has their LastActive
	// time modified
	for _, participant := range s.GetParticipants() {
		if participant.User == "fish" {
			require.NotEqual(t, clock.Now().UTC(), participant.LastActive)
		} else {
			require.Equal(t, clock.Now().UTC(), participant.LastActive)
		}
	}
}