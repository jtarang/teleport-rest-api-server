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
	"github.com/gravitational/teleport/integrations/lib/embeddedtbot"
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

func NewTeleportClientManager(ctx context.Context, bot *embeddedtbot.EmbeddedBot, addr string) (*TeleportClientManager, error) {
	creds, err := bot.StartAndWaitForCredentials(ctx, 30*time.Second)
	if err != nil {
		return nil, trace.Wrap(err, "failed to get bot credentials")
	}

	tcm, err := client.New(ctx, client.Config{
		Addrs:       []string{addr},
		Credentials: []client.Credentials{creds},
	})
	if err != nil {
		return nil, trace.Wrap(err, "failed to connect to Teleport API")
	}

	return &TeleportClientManager{Client: tcm}, nil
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

// WatchAccessRequests opens a long-lived watch stream for access request events
// and forwards them to the returned channel. The watcher runs until ctx is cancelled.
//
// Usage:
//
//	events, err := tcm.WatchAccessRequests(ctx)
//	for event := range events {
//	    log.Printf("event: %s request: %s state: %s", event.Type, event.Request.GetName(), event.Request.GetState())
//	}
func (tcm *TeleportClientManager) WatchAccessRequests(ctx context.Context) (<-chan AccessRequestEvent, error) {
	// Subscribe to the access_request resource kind only.
	watcher, err := tcm.Client.NewWatcher(ctx, types.Watch{
		Kinds: []types.WatchKind{
			{Kind: types.KindAccessRequest},
		},
	})
	if err != nil {
		return nil, trace.Wrap(err, "failed to create access request watcher")
	}

	// Block until the INIT event confirms the watch stream is ready.
	select {
	case event := <-watcher.Events():
		if event.Type != types.OpInit {
			watcher.Close()
			return nil, trace.Errorf("expected OpInit, got %v", event.Type)
		}
		log.Println("👀 Access request watcher initialised and ready.")
	case <-watcher.Done():
		return nil, trace.Wrap(watcher.Error(), "watcher closed before init")
	case <-ctx.Done():
		watcher.Close()
		return nil, ctx.Err()
	}

	out := make(chan AccessRequestEvent, 64)

	go func() {
		defer watcher.Close()
		defer close(out)

		for {
			select {
			case event, ok := <-watcher.Events():
				if !ok {
					if err := watcher.Error(); err != nil {
						log.Printf("⚠️ Access request watcher closed with error: %v", err)
					}
					return
				}

				req, ok := event.Resource.(types.AccessRequest)
				if !ok {
					log.Printf("⚠️ Unexpected resource type %T in access request watch stream, skipping.", event.Resource)
					continue
				}

				logAccessRequestEvent(event.Type, req)

				select {
				case out <- AccessRequestEvent{Type: event.Type, Request: req}:
				case <-ctx.Done():
					return
				}

			case <-watcher.Done():
				if err := watcher.Error(); err != nil {
					log.Printf("⚠️ Access request watcher stopped: %v", err)
				}
				return

			case <-ctx.Done():
				log.Println("🛑 Access request watcher context cancelled, shutting down.")
				return
			}
		}
	}()

	return out, nil
}

// logAccessRequestEvent logs a human-readable summary of each event.
func logAccessRequestEvent(op types.OpType, req types.AccessRequest) {
	switch op {
	case types.OpPut:
		// OpPut covers both creation and state updates (pending → approved/denied).
		state := req.GetState()
		switch {
		case state.IsPending():
			log.Printf("🆕 New access request | id=%s user=%s roles=%v reason=%q",
				req.GetName(), req.GetUser(), req.GetRoles(), req.GetRequestReason())
		case state.IsApproved():
			log.Printf("✅ Access request approved | id=%s user=%s roles=%v",
				req.GetName(), req.GetUser(), req.GetRoles())
		case state.IsDenied():
			log.Printf("❌ Access request denied | id=%s user=%s reason=%q",
				req.GetName(), req.GetUser(), req.GetResolveReason())
		default:
			log.Printf("🔄 Access request updated | id=%s user=%s state=%v",
				req.GetName(), req.GetUser(), state)
		}
	case types.OpDelete:
		log.Printf("🗑️  Access request deleted | id=%s", req.GetName())
	default:
		log.Printf("ℹ️  Access request event | op=%v id=%s", op, req.GetName())
	}
}
