package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mulki214/gdcpay_test_project/internal/response"
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
  /tasks:
    post:
      summary: Create a task for the authenticated user
      tags: [Tasks]
      security: [{ bearerAuth: [] }]
      requestBody:
        required: true
        content:
          application/json:
            schema: { $ref: '#/components/schemas/CreateTaskRequest' }
            example: { title: Prepare report, description: Quarterly summary }
      responses:
        '201':
          description: Task created
          content:
            application/json:
              schema: { $ref: '#/components/schemas/Task' }
        '400':
          description: Invalid request
          content:
            application/json:
              schema: { $ref: '#/components/schemas/ErrorResponse' }
        '401':
          description: Authentication required
          content:
            application/json:
              schema: { $ref: '#/components/schemas/ErrorResponse' }
    get:
      summary: List the authenticated user's tasks
      tags: [Tasks]
      security: [{ bearerAuth: [] }]
      parameters:
        - { in: query, name: page, schema: { type: integer, minimum: 1, default: 1 } }
        - { in: query, name: limit, schema: { type: integer, minimum: 1, maximum: 100, default: 20 } }
        - { in: query, name: status, schema: { type: string, enum: [pending, in_progress, completed] } }
        - { in: query, name: search, description: Case-insensitive title substring, schema: { type: string, maxLength: 200 } }
      responses:
        '200':
          description: Page of tasks
          content:
            application/json:
              schema: { $ref: '#/components/schemas/TaskPage' }
        '400':
          description: Invalid filters
          content:
            application/json:
              schema: { $ref: '#/components/schemas/ErrorResponse' }
        '401':
          description: Authentication required
          content:
            application/json:
              schema: { $ref: '#/components/schemas/ErrorResponse' }
  /tasks/{id}:
    parameters:
      - { in: path, name: id, required: true, schema: { type: string, format: uuid } }
    get:
      summary: Get a task owned by the authenticated user
      tags: [Tasks]
      security: [{ bearerAuth: [] }]
      responses:
        '200':
          description: Task details
          content:
            application/json:
              schema: { $ref: '#/components/schemas/Task' }
        '400':
          description: Invalid task ID
          content:
            application/json:
              schema: { $ref: '#/components/schemas/ErrorResponse' }
        '404':
          description: Task not found
          content:
            application/json:
              schema: { $ref: '#/components/schemas/ErrorResponse' }
    put:
      summary: Replace editable task fields
      tags: [Tasks]
      security: [{ bearerAuth: [] }]
      requestBody:
        required: true
        content:
          application/json:
            schema: { $ref: '#/components/schemas/UpdateTaskRequest' }
            example: { title: Prepare report, description: Final draft, status: in_progress }
      responses:
        '200':
          description: Task updated
          content:
            application/json:
              schema: { $ref: '#/components/schemas/Task' }
        '400':
          description: Invalid request
          content:
            application/json:
              schema: { $ref: '#/components/schemas/ErrorResponse' }
        '404':
          description: Task not found
          content:
            application/json:
              schema: { $ref: '#/components/schemas/ErrorResponse' }
    delete:
      summary: Delete a task owned by the authenticated user
      tags: [Tasks]
      security: [{ bearerAuth: [] }]
      responses:
        '204': { description: Task deleted }
        '404':
          description: Task not found
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
    CreateTaskRequest:
      type: object
      required: [title]
      properties:
        title: { type: string, minLength: 1, maxLength: 200 }
        description: { type: string, maxLength: 5000 }
    UpdateTaskRequest:
      type: object
      required: [title, status]
      properties:
        title: { type: string, minLength: 1, maxLength: 200 }
        description: { type: string, maxLength: 5000 }
        status: { type: string, enum: [pending, in_progress, completed] }
    Task:
      type: object
      properties:
        id: { type: string, format: uuid }
        owner_id: { type: string, format: uuid }
        assignee_id: { type: string, format: uuid, nullable: true }
        title: { type: string, example: Prepare report }
        description: { type: string, example: Quarterly summary }
        status: { type: string, enum: [pending, in_progress, completed] }
        created_at: { type: string, format: date-time }
        updated_at: { type: string, format: date-time }
    TaskPage:
      type: object
      properties:
        tasks: { type: array, items: { $ref: '#/components/schemas/Task' } }
        page: { type: integer, example: 1 }
        limit: { type: integer, example: 20 }
        total: { type: integer, format: int64, example: 42 }
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
		response.WriteError(c, http.StatusNotFound, "NOT_FOUND", "The requested resource was not found")
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(`<!doctype html>
<html lang="en">
<head><meta charset="utf-8"><title>Task Management API</title><link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css"></head>
<body><div id="swagger-ui"></div><script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script><script>SwaggerUIBundle({url:'/openapi.yaml',dom_id:'#swagger-ui'});</script></body>
</html>`))
}
