package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gravitational/teleport/api/client/proto"

	"github.com/gravitational/teleport/api/types"
	"github.com/gravitational/teleport/integrations/lib/embeddedtbot"
	bot2 "github.com/gravitational/teleport/lib/tbot/bot"
	"github.com/gravitational/teleport/lib/tbot/bot/onboarding"
)

func main() {

	proxyAddress := flag.String("proxy", "", "-proxy some-domain.teleport.sh")
	tbotJoinToken := flag.String("join-token", "", "-join-token some-token-value")
	flag.Parse()

	if *proxyAddress != "" {
		_, err := url.Parse(*proxyAddress)
		if err != nil {
			log.Fatalf("Invalid proxy address %q: %v", *proxyAddress, err)
		}
	}

	ctx := context.Background()

	cfg := &embeddedtbot.BotConfig{
		AuthServer: *proxyAddress,
		Onboarding: onboarding.Config{
			TokenValue: *tbotJoinToken,
			JoinMethod: types.JoinMethodToken,
		},
		CredentialLifetime: bot2.CredentialLifetime{
			TTL:                  time.Hour,
			RenewalInterval:      20 * time.Minute,
			SkipMaxTTLValidation: false,
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	bot, err := embeddedtbot.New(cfg, logger)
	if err != nil {
		log.Fatalf("failed to create bot: %v", err)
	}

	// Create the TeleportClientManager and establish connection
	tcm, err := NewTeleportClientManager(ctx, bot, *proxyAddress)

	if err != nil {
		log.Fatalf("❌ Failed to initialize Teleport Client Manager: %v", err)
	}

	// Create a Gin router with default middleware (logger and recovery)
	r := gin.Default()

	// Define a simple GET endpoint
	r.GET("/user", func(c *gin.Context) {
		user, _ := tcm.Client.GetCurrentUser(ctx)
		// Return JSON response
		c.JSON(http.StatusOK, gin.H{
			"user": user.GetName(),
		})
	})

	r.GET("/roles", func(c *gin.Context) {
		roles, err := tcm.Client.ListRoles(ctx, &proto.ListRolesRequest{})
		if err != nil {
			// If there's an error, return a 400 Bad Request
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, roles)
	})

	r.POST("/roles", func(c *gin.Context) {
		var role RoleConfig

		if err := c.ShouldBindJSON(&role); err != nil {
			// If there's an error, return a 400 Bad Request
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		err := tcm.CreateRole(ctx, role)
		if err != nil {
			// If there's an error, return a 400 Bad Request
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"roleName": role.Name})
	})

	r.POST("/access-requests", func(c *gin.Context) {
		var accessRequest AccessRequestConfig

		// If there's an error, return a 400 Bad Request
		if err := c.ShouldBindJSON(&accessRequest); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		accessReq, err := tcm.CreateAccessRequest(ctx, accessRequest)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"accessRequestId": accessReq})
	})

	// Start server on port 8080 (default)
	if err := r.Run(); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}
