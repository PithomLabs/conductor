package solvent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is an HTTP client for the Solvent API.
type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// NewClient creates a new Solvent client.
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// BeliefExplainResponse mirrors Solvent's belief explain projection.
type BeliefExplainResponse struct {
	BeliefID                   string   `json:"belief_id"`
	Claim                      string   `json:"claim"`
	ClaimType                  string   `json:"claim_type"`
	Status                     string   `json:"status"`
	RemainingDebt              []string `json:"remaining_debt"`
	FinalTruth                 bool     `json:"final_truth"`
	IsPromoted                 bool     `json:"is_promoted"`
	IsRetracted                bool     `json:"is_retracted"`
	CanPromote                 bool     `json:"can_promote"`
	PromotionBlockedReason     string   `json:"promotion_blocked_reason,omitempty"`
	CanAuthorize               bool     `json:"can_authorize"`
	AuthorizationBlockedReason string   `json:"authorization_blocked_reason,omitempty"`
	EvidenceCount              int      `json:"evidence_count"`
	HumanSummary               string   `json:"human_summary"`
}

// AuthResult mirrors Solvent's authorization verification result.
type AuthResult struct {
	TargetID string `json:"target_id"`
	Allowed  bool   `json:"allowed"`
	Reason   string `json:"reason"`
}

// VerifyAuthRequest is the request body for POST /v1/authorizations/verify.
type VerifyAuthRequest struct {
	TargetID              string          `json:"target_id"`
	ResourceType          string          `json:"resource_type"`
	ResourceID            string          `json:"resource_id"`
	Scope                 string          `json:"scope"`
	ActionNamespace       string          `json:"action_namespace"`
	ActionName            string          `json:"action_name"`
	ConsequenceType       string          `json:"consequence_type"`
	ConsequenceParameters json.RawMessage `json:"consequence_parameters"`
}

// GetBeliefExplain returns the authoritative explain for a specific belief.
func (c *Client) GetBeliefExplain(ctx context.Context, scenarioID, beliefID string) (*BeliefExplainResponse, error) {
	url := fmt.Sprintf("%s/v1/beliefs/%s/explain?scenario_id=%s", c.BaseURL, beliefID, scenarioID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("solvent API error (status %d): %s", resp.StatusCode, string(body))
	}

	var explain BeliefExplainResponse
	if err := json.NewDecoder(resp.Body).Decode(&explain); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &explain, nil
}

// VerifyAuthorization checks if an action is authorized against a target.
func (c *Client) VerifyAuthorization(ctx context.Context, req *VerifyAuthRequest) (*AuthResult, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/v1/authorizations/verify", c.BaseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("solvent API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result AuthResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}
