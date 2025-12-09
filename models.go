package main

import (
	"time"

	"github.com/gravitational/teleport/api/types"
)

// AccessRequestConfig defines the core fields needed to create a new access request.
type AccessRequestConfig struct {
	// Roles is a list of Teleport Role names being requested (e.g., ["editor", "dev-access-role"]).
	Roles []string
	// Reason is a required string explaining why the user needs the access.
	Reason string
	// Resources is a list of resources (e.g., nodes, databases, K8s clusters)
	Resources []types.ResourceID
	// Username Requesters User ID
	Username string
	//Name of the Access Request
	Name string
}

// RoleConfig defines the essential configuration for the new Teleport role
type RoleConfig struct {
	Name           string
	RoleConditions types.RoleConditions
	MaxSessionTTL  time.Duration
}
