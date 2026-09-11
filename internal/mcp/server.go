package mcp

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/PithomLabs/conductor/internal/governance"
	"github.com/PithomLabs/conductor/internal/store"
)


// Server implements MCP stdio server
type Server struct {
	taskRepo       *store.TaskRepository
	projectRepo    *store.ProjectRepository
	activityRepo   *store.ActivityRepository
	dependencyRepo *store.DependencyRepository
	governance     governance.GovernanceReader
	agentID        string
}

// NewServer creates a new MCP server
func NewServer(db *store.DB, gov governance.GovernanceReader, agentID string) *Server {
	return &Server{
		taskRepo:       store.NewTaskRepository(db),
		projectRepo:    store.NewProjectRepository(db),
		activityRepo:   store.NewActivityRepository(db),
		dependencyRepo: store.NewDependencyRepository(db),
		governance:     gov,
		agentID:        agentID,
	}
}

// MCPRequest represents an MCP tool call request
type MCPRequest struct {
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params"`
}


// MCPResponse represents an MCP tool call response
type MCPResponse struct {
	Result interface{} `json:"result,omitempty"`
	Error  string      `json:"error,omitempty"`
}

// Run starts the MCP server reading from stdin and writing to stdout
func (s *Server) Run() error {
	decoder := json.NewDecoder(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)

	for {
		var req MCPRequest
		if err := decoder.Decode(&req); err != nil {
			if err == io.EOF {
				break
			}
			log.Printf("decode error: %v", err)
			continue
		}

		resp := s.handleRequest(req)
		if err := encoder.Encode(resp); err != nil {
			log.Printf("encode error: %v", err)
		}
	}

	return nil
}

// handleRequest processes a single MCP request
func (s *Server) handleRequest(req MCPRequest) MCPResponse {
	switch req.Method {
	case "tools/list":
		return s.handleListTools()
	case "tools/call":
		return s.handleToolCall(req.Params)
	default:
		return MCPResponse{Error: fmt.Sprintf("unknown method: %s", req.Method)}
	}
}

// handleListTools returns available tools
func (s *Server) handleListTools() MCPResponse {
	tools := []map[string]interface{}{
		{
			"name":        "conductor_get_project",
			"description": "Read project context",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"project_id": map[string]interface{}{
						"type": "string",
						"description": "Project ID",
					},
				},
				"required": []string{"project_id"},
			},
		},
		{
			"name":        "conductor_list_tasks",
			"description": "List tasks in project",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"project_id": map[string]interface{}{
						"type": "string",
						"description": "Project ID",
					},
				},
				"required": []string{"project_id"},
			},
		},
		{
			"name":        "conductor_get_task",
			"description": "Read specific task",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{
						"type": "string",
						"description": "Task ID",
					},
				},
				"required": []string{"task_id"},
			},
		},
		{
			"name":        "conductor_create_task",
			"description": "Create new task",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"project_id": map[string]interface{}{
						"type": "string",
						"description": "Project ID",
					},
					"title": map[string]interface{}{
						"type": "string",
						"description": "Task title",
					},
					"description": map[string]interface{}{
						"type": "string",
						"description": "Task description",
					},
					"governance_ref": map[string]interface{}{
						"type":        "string",
						"description": "Opaque governance reference (optional, write-once at creation)",
					},
				},
				"required": []string{"project_id", "title"},
			},
		},
		{
			"name":        "conductor_update_task",
			"description": "Update task attributes (NOT status/current_agent)",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{
						"type": "string",
						"description": "Task ID",
					},
					"title": map[string]interface{}{
						"type": "string",
						"description": "Task title",
					},
					"description": map[string]interface{}{
						"type": "string",
						"description": "Task description",
					},
				},
				"required": []string{"task_id"},
			},
		},
		{
			"name":        "conductor_claim_task",
			"description": "Agent claims task (atomic)",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{
						"type": "string",
						"description": "Task ID",
					},
				},
				"required": []string{"task_id"},
			},
		},
		{
			"name":        "conductor_submit_task",
			"description": "Agent submits for review",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{
						"type": "string",
						"description": "Task ID",
					},
				},
				"required": []string{"task_id"},
			},
		},
		{
			"name":        "conductor_accept_task",
			"description": "Reviewer accepts work",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{
						"type": "string",
						"description": "Task ID",
					},
				},
				"required": []string{"task_id"},
			},
		},
		{
			"name":        "conductor_reject_task",
			"description": "Reviewer rejects work",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{
						"type": "string",
						"description": "Task ID",
					},
				},
				"required": []string{"task_id"},
			},
		},
		{
			"name":        "conductor_report_blocker",
			"description": "Report blocking issue",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{
						"type": "string",
						"description": "Task ID",
					},
				},
				"required": []string{"task_id"},
			},
		},
		{
			"name":        "conductor_resolve_blocker",
			"description": "Resolve blocking issue",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{
						"type": "string",
						"description": "Task ID",
					},
				},
				"required": []string{"task_id"},
			},
		},
		{
			"name":        "conductor_release_task",
			"description": "Release a claimed task back to proposed state",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{
						"type":        "string",
						"description": "Task ID",
					},
				},
				"required": []string{"task_id"},
			},
		},
		{
			"name":        "conductor_post_activity",
			"description": "Record activity",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{
						"type": "string",
						"description": "Task ID",
					},
					"action": map[string]interface{}{
						"type": "string",
						"description": "Activity action",
					},
					"details": map[string]interface{}{
						"type": "string",
						"description": "Activity details",
					},
				},
				"required": []string{"task_id", "action"},
			},
		},
		{
			"name":        "conductor_get_governance",
			"description": "Get governance status (read-only)",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{
						"type": "string",
						"description": "Task ID",
					},
				},
				"required": []string{"task_id"},
			},
		},
		{
			"name":        "conductor_next_task",
			"description": "Get next unblocked task",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"project_id": map[string]interface{}{
						"type": "string",
						"description": "Project ID",
					},
				},
				"required": []string{"project_id"},
			},
		},
	}

	return MCPResponse{Result: map[string]interface{}{"tools": tools}}
}

// handleToolCall processes a tool call
func (s *Server) handleToolCall(params map[string]interface{}) MCPResponse {
	toolName, _ := params["name"].(string)
	arguments, _ := params["arguments"].(map[string]interface{})

	return s.handleTool(toolName, arguments)
}

// CallTool is a public method for calling tools from outside the package.
// This is used by the agent stub for testing.
func (s *Server) CallTool(toolName string, args map[string]interface{}) MCPResponse {
	return s.handleTool(toolName, args)
}
