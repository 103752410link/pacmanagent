package agent

import (
	"fmt"
	"sync"
)

// AgentRole defines the role of an agent
type AgentRole int

const (
	RolePlayer AgentRole = iota
	RoleGhost
)

func (r AgentRole) String() string {
	switch r {
	case RolePlayer:
		return "player"
	case RoleGhost:
		return "ghost"
	default:
		return "unknown"
	}
}

// Agent represents a registered game agent
type Agent struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Strategy string    `json:"strategy,omitempty"`
	Role     AgentRole `json:"role"`
}

// Registry manages registered agents
type Registry struct {
	mu        sync.RWMutex
	agents    map[string]*Agent
	agentsList []*Agent
	playerID  string
}

// NewRegistry creates a new agent registry
func NewRegistry() *Registry {
	return &Registry{
		agents: make(map[string]*Agent),
	}
}

// Register adds a new agent. First agent becomes player, rest become ghosts.
func (r *Registry) Register(name, strategy string) *Agent {
	r.mu.Lock()
	defer r.mu.Unlock()

	counter := len(r.agents) + 1
	id := generateID(counter)

	var role AgentRole
	if r.playerID == "" {
		role = RolePlayer
		r.playerID = id
	} else {
		role = RoleGhost
	}

	agent := &Agent{
		ID:       id,
		Name:     name,
		Strategy: strategy,
		Role:     role,
	}
	r.agents[id] = agent
	r.agentsList = append(r.agentsList, agent)
	return agent
}

// Unregister removes an agent
func (r *Registry) Unregister(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.agents[id]; exists {
		delete(r.agents, id)
		if r.playerID == id {
			r.playerID = ""
		}
		return true
	}
	return false
}

// Get returns an agent by ID
func (r *Registry) Get(id string) *Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.agents[id]
}

// GetPlayerID returns the player agent ID
func (r *Registry) GetPlayerID() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.playerID
}

// GetGhosts returns all ghost agents
func (r *Registry) GetGhosts() []*Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var ghosts []*Agent
	for _, a := range r.agentsList {
		if a.Role == RoleGhost {
			ghosts = append(ghosts, a)
		}
	}
	return ghosts
}

// List returns all registered agents
func (r *Registry) List() []*Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*Agent, len(r.agentsList))
	copy(result, r.agentsList)
	return result
}

// Count returns the number of registered agents
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.agents)
}

func generateID(n int) string {
	return fmt.Sprintf("agent-%04d", n)
}
