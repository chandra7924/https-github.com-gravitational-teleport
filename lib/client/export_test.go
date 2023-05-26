// Copyright 2022 Gravitational, Inc
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

package client

import (
	dtauthn "github.com/gravitational/teleport/lib/devicetrust/authn"
)

var HasTouchIDCredentials = &hasTouchIDCredentials

func (tc *TeleportClient) SetDTAttemptLoginIgnorePing(val bool) {
	tc.dtAttemptLoginIgnorePing = val
}

func (tc *TeleportClient) SetDTAutoEnrollIgnorePing(val bool) {
	tc.dtAutoEnrollIgnorePing = val
}

func (tc *TeleportClient) SetDTAuthnCeremony(c dtauthn.CeremonyI) {
	tc.dtAuthnCeremony = c
}

func (tc *TeleportClient) SetDTAutoEnroll(fn dtAutoEnrollFunc) {
	tc.dtAutoEnroll = fn
}
