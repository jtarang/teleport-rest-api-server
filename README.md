# 🚀 Teleport REST API Server
**Golang 1.25 · Gin Framework · Teleport API**

A lightweight REST API server that exposes Teleport management operations—such as fetching the current user, listing roles, and creating access requests—using Teleport’s Go SDK and the Gin web framework.

This service provides a simple REST interface on top of Teleport, making it easy for external tools, scripts, and integrations to interact with Teleport without using tsh or the Teleport API directly.

---

## 📌 Hoppscotch Collection Included

This repository includes a **Hoppscotch API collection** you can import directly into Hoppscotch (or Postman, Bruno, etc.) to test all REST endpoints instantly.

The example requests in this README match the Hoppscotch file exactly.

File name suggestion:

```
teleport-rest-api-hoppscotch.json
```

---

## 📁 Project Structure

```
teleport-rest-api-server/
├── main.go
├── teleport_client_manager.go
├── models.go
├── go.mod
└── README.md
```

---

## ▶️ Running the Server

```bash
go run main.go
```

Server runs at:

```
http://localhost:8080
```

---

# 📡 API Endpoints (Matches Hoppscotch Collection)

Below are the same requests included in the Hoppscotch export you provided.

---

## 🟦 GET /user
### Get Current User

**Endpoint**

```
GET http://localhost:8080/user
```

**Sample Response**

```json
{
  "user": "alice"
}
```

---

## 🟦 GET /roles
### Get Roles

**Endpoint**

```
GET http://localhost:8080/roles
```

**Sample Response**

```json
[
  {
    "name": "db-access-role",
    "version": "v8"
    ...
  }
]
```

---

## 🟩 POST /roles
### Create Role

**Endpoint**

```
POST http://localhost:8080/roles
```

**Request Body (matches Hoppscotch)**

```json
{
  "Name": "db-access-role",
  "MaxSessionTTL": 1800,
  "RoleConditions": {
    "db_labels": {
      "*": "*"
    },
    "db_names": [
      "databaseA",
      "databaseB"
    ]
  }
}
```

**Sample Response**

```json
{
  "roleName": "db-access-role"
}
```

---

## 🟩 POST /access-requests
### Create Access Request

**Endpoint**

```
POST http://localhost:8080/access-requests
```

**Request Body (matches Hoppscotch)**

```json
{
  "Name": "some-user-db-access",
  "Reason": "Need access to databases for databaseA and databaseB",
  "Roles": [
    "db-access-role"
  ],
  "Username": "username@somewhere.com",
  "Resources": [
    {
      "kind": "database",
      "name": "databaseA"
    },
    {
      "kind": "database",
      "name": "databaseB"
    }
  ]
}
```

**Sample Response**

```json
{
  "accessRequestId": "983fc3fa-a68f-4de3-bde0-27cf830cc3a3"
}
```

---

# 📥 Importing the Hoppscotch File

1. Open **hoppscotch.io**
2. Go to **Collections → Import**
3. Upload the file:

```
teleport-rest-api-hoppscotch.json
```

You will now see:

- Get Current User
- Get Roles
- Create Role
- Create Access Request

all preconfigured and ready to execute.

---

# 🧩 Extending the API

You can extend this API server further:

- Approve / deny access requests
- List databases, servers, apps, kubernetes clusters
- Manage Teleport users

Teleport API docs:  
`https://pkg.go.dev/github.com/gravitational/teleport/api/client`
