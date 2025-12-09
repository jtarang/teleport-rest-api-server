// teleport_client_manager.go
package main

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/gravitational/teleport/api/client"
	"github.com/gravitational/teleport/api/client/proto"
	"github.com/gravitational/teleport/api/types"
	"github.com/gravitational/trace"
)

// TeleportConfig holds the connection details.
type TeleportConfig struct {
	ProxyAddresses []string
}

// TeleportClientManager encapsulates the connection and methods that operate on it.
type TeleportClientManager struct {
	Client *client.Client
	Config TeleportConfig
}

// NewTeleportClientManager establishes the connection and returns the manager struct.
func NewTeleportClientManager(ctx context.Context, cfg TeleportConfig) (*TeleportClientManager, error) {
	// WARNING: client.LoadProfile("", "") is used here for demonstration purposes

	tcm, err := client.New(ctx, client.Config{
		Addrs:       cfg.ProxyAddresses,
		Credentials: []client.Credentials{client.LoadProfile("", "")},
	})
	if err != nil {
		return nil, trace.Wrap(err, "failed to connect to Teleport API")
	}

	log.Printf("✅ Successfully established connection to Teleport at: %v", cfg.ProxyAddresses)

	return &TeleportClientManager{
		Client: tcm,
		Config: cfg,
	}, nil
}

// Close is a method to ensure the underlying client connection is properly closed.
func (tcm *TeleportClientManager) Close() error {
	if tcm.Client != nil {
		log.Println("🔌 Closing Teleport API connection.")
		return tcm.Client.Close()
	}
	return nil
}

// CreateRole is a method on TeleportClientManager to create a new role.
func (tcm *TeleportClientManager) CreateRole(ctx context.Context, roleConfig RoleConfig) error {
	roleSpec := types.RoleSpecV6{
		Options: types.RoleOptions{
			MaxSessionTTL: types.NewDuration(roleConfig.MaxSessionTTL * time.Second),
		},
		Allow: roleConfig.RoleConditions,
	}
	role, err := types.NewRole(roleConfig.Name, roleSpec)
	if err != nil {
		return trace.Wrap(err, "failed to create role object")
	}

	// We use tcm.Client to access the established connection
	if _, err := tcm.Client.CreateRole(ctx, role); err != nil {
		if trace.IsAlreadyExists(err) {
			log.Printf("⚠️ Role '%s' already exists. Skipping creation.", roleConfig.Name)
			return nil
		}
		return trace.Wrap(err, "failed to create role on Teleport")
	}

	log.Printf("✅ Role '%s' created successfully.", roleConfig.Name)
	return nil
}

// ListRoles is a method on TeleportClientManager to retrieve all role names.
func (tcm *TeleportClientManager) ListRoles(ctx context.Context) ([]string, error) {
	// The request object, as identified by your client version
	req := proto.ListRolesRequest{}

	roles, err := tcm.Client.ListRoles(ctx, &req)
	if err != nil {
		return nil, trace.Wrap(err, "failed to list roles")
	}

	var names []string
	for _, r := range roles.Roles {
		names = append(names, r.GetName())
	}
	log.Printf(" Fetched all Roles successfully.")
	return names, nil
}

// CreateAccessRequest is a method on TeleportClientManager to submit a new access request.
func (tcm *TeleportClientManager) CreateAccessRequest(ctx context.Context, requestConfig AccessRequestConfig) (string, error) {

	accessRequestObject := types.AccessRequestV3{
		Kind:    "access_request",
		SubKind: "",
		Version: "v3",
		Metadata: types.Metadata{
			Name: uuid.NewString(),
		},
		Spec: types.AccessRequestSpecV3{
			User:          requestConfig.Username,
			Roles:         requestConfig.Roles,
			RequestReason: requestConfig.Reason,
		},
	}

	accessRequest, err := tcm.Client.CreateAccessRequestV2(ctx, &accessRequestObject)
	if err != nil {
		return "", trace.Wrap(err, "failed to create new access request")
	}

	log.Printf("✅ Access Request submitted by user %s for roles: %v", requestConfig.Username, requestConfig.Roles)
	return accessRequest.GetName(), nil
}
