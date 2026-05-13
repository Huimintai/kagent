# go/core/internal/httpserver — REST API Server

HTTP server exposing the platform API. Handles agent management, sessions, tools, models, and more.

## Structure

| Package/File | Role |
|-------------|------|
| `server.go` | Server initialization, route registration |
| `middleware.go` | Request middleware (logging, CORS, request ID) |
| `middleware_error.go` | Error recovery middleware |
| `handlers/` | Route handler implementations (~35+ handlers) |
| `auth/` | Authentication/authorization (authn, authz, proxy authn) |
| `errors/` | Structured error responses |

## handlers/ Overview

Handlers cover: agents, sessions, tools, toolservers, models, model configs, model provider configs, namespaces, memory, feedback, health, checkpoints, tasks, comments, stats, scheduled runs, prompt templates, broker, visibility, current user.

## auth/ Package

| File | Role |
|------|------|
| `authn.go` | Authentication middleware |
| `authz.go` | Authorization checks |
| `proxy_authn.go` | Proxy-based authentication (trusted headers) |
