// Copyright (c) 2020, AT&T Intellectual Property. All rights reserved.
//
// SPDX-License-Identifier: LGPL-2.1-only

package session

// Allow UTs to set the owner of an existing Session
func (s *Session) SetOwner(owner uint32) {
	s.s.owner = &owner
}
