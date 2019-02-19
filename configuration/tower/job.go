package tower

// Job represents a job in tower
type Job struct {
	ID       uint   `json:"id"`
	Name     string `json:"name"`
	Playbook string `json:"playbook"`
	Status   string `json:"status"`
}
