package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gravitational/teleport/api/client/proto"
)

func main() {

	proxyAddress := flag.String("proxy", "", "-proxy some-domain.teleport.sh")
	proxyPorts := flag.String("proxy-ports", "443,3025,3024,3080", "comma-separated list of ports")
	flag.Parse()

	portsStr := strings.Split(*proxyPorts, ",")
	proxyAddresses := make([]string, 0, len(portsStr))

	if *proxyAddress != "" {
		_, err := url.Parse(*proxyAddress)
		if err != nil {
			log.Fatalf("Invalid proxy address %q: %v", *proxyAddress, err)
		}
	}

	for _, port := range portsStr {
		port = strings.TrimSpace(port)
		portInt, err := strconv.Atoi(port)
		if err != nil || portInt < 1 || portInt > 65535 {
			log.Fatalf("Invalid port %q: must be 1–65535", port)
		}
		proxyAddresses = append(proxyAddresses, net.JoinHostPort(*proxyAddress, strconv.Itoa(portInt)))
	}

	ctx := context.Background()

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
