package mcp

import (
	"github.com/google/uuid"
	"context"
	"encoding/json"

	"github.com/PithomLabs/conductor/internal/domain"
	"github.com/PithomLabs/conductor/internal/governance"
	"github.com/PithomLabs/conductor/internal/store"
)

var ctx = context.Background()

// handleTool processes a tool call
func (s *Server) handleTool(toolName string, args map[string]interface{}) MCPResponse {
	switch toolName {
	case "conductor_get_project":
		return s.getProject(args)
	case "conductor_list_tasks":
		return s.listTasks(args)
	case "conductor_get_task":
		return s.getTask(args)
	case "conductor_create_task":
		return s.createTask(args)
	case "conductor_update_task":
		return s.updateTask(args)
	case "conductor_claim_task":
		return s.claimTask(args)
	case "conductor_submit_task":
		return s.submitTask(args)
	case "conductor_accept_task":
		return s.acceptTask(args)
	case "conductor_reject_task":
		return s.rejectTask(args)
	case "conductor_report_blocker":
		return s.reportBlocker(args)
	case "conductor_resolve_blocker":
		return s.resolveBlocker(args)
	case "conductor_release_task":
		return s.releaseTask(args)
	case "conductor_post_activity":
		return s.postActivity(args)
	case "conductor_get_governance":
		return s.getGovernance(args)
	case "conductor_next_task":
		return s.nextTask(args)
	default:
		return MCPResponse{Error: "unknown tool: " + toolName}
	}
}

func (s *Server) getProject(args map[string]interface{}) MCPResponse {
	projectID, _ := args["project_id"].(string)
	if projectID == "" {
		return MCPResponse{Error: "project_id is required"}
	}

	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return MCPResponse{Error: err.Error()}
	}
	return MCPResponse{Result: project}
}

func (s *Server) listTasks(args map[string]interface{}) MCPResponse {
	projectID, _ := args["project_id"].(string)
	if projectID == "" {
		return MCPResponse{Error: "project_id is required"}
	}

	tasks, err := s.taskRepo.ListByProject(ctx, projectID)
	if err != nil {
		return MCPResponse{Error: err.Error()}
	}
	return MCPResponse{Result: tasks}
}

func (s *Server) getTask(args map[string]interface{}) MCPResponse {
	taskID, _ := args["task_id"].(string)
	if taskID == "" {
		return MCPResponse{Error: "task_id is required"}
	}

	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return MCPResponse{Error: err.Error()}
	}
	return MCPResponse{Result: task}
}

func (s *Server) createTask(args map[string]interface{}) MCPResponse {
	projectID, _ := args["project_id"].(string)
	taskID, _ := args["task_id"].(string)
	title, _ := args["title"].(string)
	description, _ := args["description"].(string)

	if projectID == "" || title == "" {
		return MCPResponse{Error: "project_id and title are required"}
	}

	if taskID == "" {
		taskID = uuid.New().String()
	}

	task := &domain.Task{
		ID:          taskID,
		ProjectID:   projectID,
		Title:       title,
		Description: description,
	}

	if err := s.taskRepo.Create(ctx, task); err != nil {
		return MCPResponse{Error: err.Error()}
	}
	return MCPResponse{Result: task}
}

func (s *Server) updateTask(args map[string]interface{}) MCPResponse {
	taskID, _ := args["task_id"].(string)
	if taskID == "" {
		return MCPResponse{Error: "task_id is required"}
	}

	title, _ := args["title"].(string)
	description, _ := args["description"].(string)

	fields := store.TaskUpdateFields{}
	if title != "" {
		fields.Title = &title
	}
	if description != "" {
		fields.Description = &description
	}

	if err := s.taskRepo.Update(ctx, taskID, fields); err != nil {
		return MCPResponse{Error: err.Error()}
	}

	task, _ := s.taskRepo.GetByID(ctx, taskID)
	return MCPResponse{Result: task}
}

func (s *Server) claimTask(args map[string]interface{}) MCPResponse {
	taskID, _ := args["task_id"].(string)
	if taskID == "" {
		return MCPResponse{Error: "task_id is required"}
	}

	if err := s.taskRepo.Claim(ctx, taskID, s.agentID); err != nil {
		return MCPResponse{Error: err.Error()}
	}

	task, _ := s.taskRepo.GetByID(ctx, taskID)
	return MCPResponse{Result: task}
}

func (s *Server) submitTask(args map[string]interface{}) MCPResponse {
	taskID, _ := args["task_id"].(string)
	if taskID == "" {
		return MCPResponse{Error: "task_id is required"}
	}

	err := s.taskRepo.Transition(ctx, taskID,
		domain.TaskStatusActive, domain.TaskStatusReview,
		"agent", s.agentID, "task.submitted", "{}")
	if err != nil {
		return MCPResponse{Error: err.Error()}
	}

	task, _ := s.taskRepo.GetByID(ctx, taskID)
	return MCPResponse{Result: task}
}

func (s *Server) acceptTask(args map[string]interface{}) MCPResponse {
	taskID, _ := args["task_id"].(string)
	if taskID == "" {
		return MCPResponse{Error: "task_id is required"}
	}

	err := s.taskRepo.Transition(ctx, taskID,
		domain.TaskStatusReview, domain.TaskStatusAccepted,
		"agent", s.agentID, "task.accepted", "{}")
	if err != nil {
		return MCPResponse{Error: err.Error()}
	}

	task, _ := s.taskRepo.GetByID(ctx, taskID)
	return MCPResponse{Result: task}
}

func (s *Server) rejectTask(args map[string]interface{}) MCPResponse {
	taskID, _ := args["task_id"].(string)
	if taskID == "" {
		return MCPResponse{Error: "task_id is required"}
	}

	err := s.taskRepo.Transition(ctx, taskID,
		domain.TaskStatusReview, domain.TaskStatusActive,
		"agent", s.agentID, "task.rejected", "{}")
	if err != nil {
		return MCPResponse{Error: err.Error()}
	}

	task, _ := s.taskRepo.GetByID(ctx, taskID)
	return MCPResponse{Result: task}
}

func (s *Server) reportBlocker(args map[string]interface{}) MCPResponse {
	taskID, _ := args["task_id"].(string)
	if taskID == "" {
		return MCPResponse{Error: "task_id is required"}
	}

	err := s.taskRepo.Transition(ctx, taskID,
		domain.TaskStatusActive, domain.TaskStatusBlocked,
		"agent", s.agentID, "task.blocked", "{}")
	if err != nil {
		return MCPResponse{Error: err.Error()}
	}

	task, _ := s.taskRepo.GetByID(ctx, taskID)
	return MCPResponse{Result: task}
}

func (s *Server) resolveBlocker(args map[string]interface{}) MCPResponse {
	taskID, _ := args["task_id"].(string)
	if taskID == "" {
		return MCPResponse{Error: "task_id is required"}
	}

	err := s.taskRepo.Transition(ctx, taskID,
		domain.TaskStatusBlocked, domain.TaskStatusActive,
		"agent", s.agentID, "task.unblocked", "{}")
	if err != nil {
		return MCPResponse{Error: err.Error()}
	}

	task, _ := s.taskRepo.GetByID(ctx, taskID)
	return MCPResponse{Result: task}
}


func (s *Server) releaseTask(args map[string]interface{}) MCPResponse {
	taskID, _ := args["task_id"].(string)
	if taskID == "" {
		return MCPResponse{Error: "task_id is required"}
	}

	if err := s.taskRepo.Release(ctx, taskID, s.agentID); err != nil {
		return MCPResponse{Error: err.Error()}
	}

	task, _ := s.taskRepo.GetByID(ctx, taskID)
	return MCPResponse{Result: task}
}

func (s *Server) postActivity(args map[string]interface{}) MCPResponse {
	taskID, _ := args["task_id"].(string)
	action, _ := args["action"].(string)
	details, _ := args["details"].(string)

	if taskID == "" || action == "" {
		return MCPResponse{Error: "task_id and action are required"}
	}

	activity := &domain.Activity{
		ID:        uuid.New().String(),
		TaskID:    taskID,
		ActorType: "agent",
		ActorID:   s.agentID,
		Action:    action,
		Details:   details,
	}

	if err := s.activityRepo.Append(ctx, activity); err != nil {
		return MCPResponse{Error: err.Error()}
	}
	return MCPResponse{Result: activity}
}

func (s *Server) getGovernance(args map[string]interface{}) MCPResponse {
	taskID, _ := args["task_id"].(string)
	if taskID == "" {
		return MCPResponse{Error: "task_id is required"}
	}

	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return MCPResponse{Error: err.Error()}
	}

	if task.GovernanceRef == nil || *task.GovernanceRef == "" {
		return MCPResponse{Result: map[string]interface{}{"status": "none"}}
	}

	var ref governance.GovernanceReference
	if err := json.Unmarshal([]byte(*task.GovernanceRef), &ref); err != nil {
		ref = governance.GovernanceReference{
			Provider:    "unknown",
			ReferenceID: *task.GovernanceRef,
		}
	}
	state, err := s.governance.GetState(ctx, ref)
	if err != nil {
		state = &governance.GovernanceState{
			Reference: ref,
			Status:    governance.GovernanceStatusUnknown,
			Blockers:  []string{"provider unavailable"},
		}
	}
	return MCPResponse{Result: state}
}


func (s *Server) nextTask(args map[string]interface{}) MCPResponse {
	projectID, _ := args["project_id"].(string)
	if projectID == "" {
		return MCPResponse{Error: "project_id is required"}
	}

	tasks, err := s.taskRepo.ListByProject(ctx, projectID)
	if err != nil {
		return MCPResponse{Error: err.Error()}
	}

	for _, task := range tasks {
		if task.Status != domain.TaskStatusProposed || task.CurrentAgent != nil {
			continue
		}

		// Check all dependencies are resolved
		deps, err := s.dependencyRepo.ListByTask(ctx, task.ID)
		if err != nil {
			return MCPResponse{Error: err.Error()}
		}

		allResolved := true
		for _, dep := range deps {
			blocker, err := s.taskRepo.GetByID(ctx, dep.BlockedByID)
			if err != nil {
				allResolved = false
				break
			}
			// Dependency resolved if blocker is accepted or cancelled
			if blocker.Status != domain.TaskStatusAccepted && blocker.Status != domain.TaskStatusCancelled {
				allResolved = false
				break
			}
		}

		if allResolved {
			return MCPResponse{Result: task}
		}
	}

	return MCPResponse{Error: "no unblocked tasks available"}
}
