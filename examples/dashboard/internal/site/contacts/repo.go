package contacts

import (
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/fastygo/framework/examples/dashboard/internal/domain"
)

// Repository is a thread-safe in-memory store. Replace with a database
// implementation when graduating from a starter kit.
type Repository struct {
	mu       sync.RWMutex
	contacts map[string]domain.Contact
}

// NewRepository constructs an empty Repository.
func NewRepository() *Repository {
	return &Repository{contacts: make(map[string]domain.Contact)}
}

// Seed pre-populates the repository with a few rows. Useful for demo data.
func (r *Repository) Seed() {
	demo := []domain.Contact{
		{Name: "Ada Lovelace", Email: "ada@example.com", Company: "Analytical Engines"},
		{Name: "Alan Turing", Email: "alan@example.com", Company: "Bletchley Park"},
		{Name: "Grace Hopper", Email: "grace@example.com", Company: "US Navy"},
	}
	for _, c := range demo {
		_, _ = r.Create(c)
	}
}

// Create stores a new contact, assigning a UUID and creation timestamp.
func (r *Repository) Create(c domain.Contact) (domain.Contact, error) {
	if err := c.Validate(); err != nil {
		return domain.Contact{}, err
	}
	c.ID = uuid.NewString()
	c.CreatedAt = time.Now().UTC()

	r.mu.Lock()
	r.contacts[c.ID] = c
	r.mu.Unlock()
	return c, nil
}

// List returns every contact sorted by name.
func (r *Repository) List() []domain.Contact {
	r.mu.RLock()
	out := make([]domain.Contact, 0, len(r.contacts))
	for _, c := range r.contacts {
		out = append(out, c)
	}
	r.mu.RUnlock()

	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

// Delete removes a contact by ID. Missing IDs are ignored silently.
func (r *Repository) Delete(id string) {
	r.mu.Lock()
	delete(r.contacts, id)
	r.mu.Unlock()
}
