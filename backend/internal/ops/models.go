package ops

type Overview struct {
	Mode        string `json:"mode"`
	Clusters    int    `json:"clusters"`
	Namespaces  int    `json:"namespaces"`
	Workloads   int    `json:"workloads"`
	Alerts      int    `json:"alerts"`
	Environment string `json:"environment"`
}

type Cluster struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Version string `json:"version"`
	Nodes   int    `json:"nodes"`
}

type Namespace struct {
	Name    string `json:"name"`
	Phase   string `json:"phase"`
	Age     string `json:"age"`
	Cluster string `json:"cluster"`
}

type Workload struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Kind      string `json:"kind"`
	Ready     string `json:"ready"`
	Image     string `json:"image"`
}

type Tool struct {
	Name             string `json:"name"`
	Description      string `json:"description"`
	Write            bool   `json:"write"`
	RequiresApproval bool   `json:"requiresApproval"`
}

type ToolRequest struct {
	Tool       string            `json:"tool"`
	Actor      string            `json:"actor"`
	Target     string            `json:"target"`
	Approved   bool              `json:"approved"`
	Parameters map[string]string `json:"parameters"`
}

type ToolResult struct {
	AuditID string `json:"auditId"`
	Status  string `json:"status"`
	Message string `json:"message"`
}
