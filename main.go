package main

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gravitational/teleport/api/client/proto"
)

func main() {

	ctx := context.Background()

	// Your Teleport Proxy's public address
	proxyAddresses := []string{
		"example.teleport.sh:443",
		"example.teleport.sh:3025",
		"example.teleport.sh:3024",
		"example.teleport.sh:3080",
	}

	// Create the TeleportClientManager and establish connection
	tcm, err := NewTeleportClientManager(ctx, TeleportConfig{ProxyAddresses: proxyAddresses})

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
