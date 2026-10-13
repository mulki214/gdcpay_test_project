package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const openAPIDocument = `openapi: 3.0.3
info:
  title: Task Management API
  version: 0.1.0
  description: API documentation. This specification grows together with implemented endpoints.
servers:
  - url: http://localhost:8080
paths:
  /health:
    get:
      summary: Check application liveness
      tags: [Health]
      responses:
        '200':
          description: Application process is running
          content:
            application/json:
              schema:
                type: object
                properties:
                  status: { type: string, example: ok }
  /auth/register:
    post:
      summary: Register a user and return a JWT
      tags: [Authentication]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CredentialsRequest'
            example:
              email: user@example.com
              password: correct-horse-battery
      responses:
        '201':
          description: User registered
          content:
            application/json:
              schema: { $ref: '#/components/schemas/AuthResponse' }
        '400':
          description: Invalid request or credentials format
          content:
            application/json:
              schema: { $ref: '#/components/schemas/ErrorResponse' }
        '409':
          description: Email is already registered
          content:
            application/json:
              schema: { $ref: '#/components/schemas/ErrorResponse' }
        '500':
          description: Unexpected server error
          content:
            application/json:
              schema: { $ref: '#/components/schemas/ErrorResponse' }
  /auth/login:
    post:
      summary: Authenticate a user and return a JWT
      tags: [Authentication]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CredentialsRequest'
            example:
              email: user@example.com
              password: correct-horse-battery
      responses:
        '200':
          description: User authenticated
          content:
            application/json:
              schema: { $ref: '#/components/schemas/AuthResponse' }
        '400':
          description: Missing request fields
          content:
            application/json:
              schema: { $ref: '#/components/schemas/ErrorResponse' }
        '401':
          description: Email or password is incorrect
          content:
            application/json:
              schema: { $ref: '#/components/schemas/ErrorResponse' }
        '500':
          description: Unexpected server error
          content:
            application/json:
              schema: { $ref: '#/components/schemas/ErrorResponse' }
  /ready:
    get:
      summary: Check application readiness and database connectivity
      tags: [Health]
      responses:
        '200':
          description: Application is ready
          content:
            application/json:
              schema:
                type: object
                properties:
                  status: { type: string, example: ready }
        '503':
          description: Database is unavailable
          content:
            application/json:
              schema:
                type: object
                properties:
                  status: { type: integer, example: 503 }
                  code: { type: string, example: DATABASE_UNAVAILABLE }
                  message: { type: string, example: The service is not ready }
                  timestamp: { type: string, format: date-time }
components:
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
  schemas:
    CredentialsRequest:
      type: object
      required: [email, password]
      properties:
        email: { type: string, format: email, example: user@example.com }
        password: { type: string, minLength: 8, maxLength: 72, example: correct-horse-battery }
    AuthResponse:
      type: object
      properties:
        user:
          type: object
          properties:
            id: { type: string, format: uuid }
            email: { type: string, format: email }
        access_token: { type: string, description: JWT bearer token }
        token_type: { type: string, example: Bearer }
        expires_at: { type: string, format: date-time }
    ErrorResponse:
      type: object
      required: [status, code, message, timestamp]
      properties:
        status: { type: integer, example: 400 }
        code: { type: string, example: INVALID_REQUEST }
        message: { type: string, example: The request could not be processed }
        timestamp: { type: string, format: date-time }
`

func openAPISpec(c *gin.Context) {
	c.Data(http.StatusOK, "application/yaml; charset=utf-8", []byte(openAPIDocument))
}

func swaggerUI(c *gin.Context) {
	if c.Param("any") != "/" && c.Param("any") != "/index.html" {
		c.Status(http.StatusNotFound)
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(`<!doctype html>
<html lang="en">
<head><meta charset="utf-8"><title>Task Management API</title><link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css"></head>
<body><div id="swagger-ui"></div><script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script><script>SwaggerUIBundle({url:'/openapi.yaml',dom_id:'#swagger-ui'});</script></body>
</html>`))
}
