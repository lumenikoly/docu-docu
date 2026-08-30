package toudocu

import "time"

type GeneratorInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type TaskVerifyTask struct {
	ID       string     `json:"id"`
	Title    string     `json:"title"`
	Status   StatusInfo `json:"status"`
	Type     string     `json:"type,omitempty"`
	Document string     `json:"document"`
}

type CommandExecutionResult struct {
	Sequence        int       `json:"sequence"`
	Command         string    `json:"command"`
	Targets         []string  `json:"targets"`
	Status          string    `json:"status"`
	ExitCode        *int      `json:"exitCode"`
	StartedAt       time.Time `json:"startedAt"`
	FinishedAt      time.Time `json:"finishedAt"`
	DurationMillis  int64     `json:"durationMillis"`
	Stdout          string    `json:"stdout"`
	Stderr          string    `json:"stderr"`
	StdoutTruncated bool      `json:"stdoutTruncated"`
	StderrTruncated bool      `json:"stderrTruncated"`
}

type CriterionExecutionResult struct {
	ID                string `json:"id"`
	Description       string `json:"description"`
	DocumentCompleted bool   `json:"documentCompleted"`
	Status            string `json:"status"`
}

type TargetExecutionResult struct {
	Target string `json:"target"`
	Status string `json:"status"`
}

type TaskVerifySummary struct {
	TotalCommands    int `json:"totalCommands"`
	PassedCommands   int `json:"passedCommands"`
	FailedCommands   int `json:"failedCommands"`
	TimedOutCommands int `json:"timedOutCommands"`
	CriteriaPassed   int `json:"criteriaPassed"`
	CriteriaFailed   int `json:"criteriaFailed"`
}

type TaskVerifyReport struct {
	SchemaVersion    int                        `json:"schemaVersion"`
	Kind             string                     `json:"kind"`
	Generator        GeneratorInfo              `json:"generator"`
	Task             TaskVerifyTask             `json:"task"`
	StartedAt        time.Time                  `json:"startedAt"`
	FinishedAt       time.Time                  `json:"finishedAt"`
	DurationMillis   int64                      `json:"durationMillis"`
	Status           string                     `json:"status"`
	Mode             string                     `json:"mode"`
	Target           string                     `json:"target,omitempty"`
	FullVerification bool                       `json:"fullVerification"`
	ValidationIssues []Issue                    `json:"validationIssues"`
	Issues           []Issue                    `json:"issues"`
	Commands         []CommandExecutionResult   `json:"commands"`
	Criteria         []CriterionExecutionResult `json:"criteria"`
	Targets          []TargetExecutionResult    `json:"targets"`
	Summary          TaskVerifySummary          `json:"summary"`
}

type SearchMatch struct {
	ID              string   `json:"id,omitempty"`
	Type            string   `json:"type"`
	Title           string   `json:"title"`
	Path            string   `json:"path"`
	Archived        bool     `json:"archived"`
	ArchiveYear     string   `json:"archiveYear,omitempty"`
	MatchedSections []string `json:"matchedSections"`
}

type SearchReport struct {
	SchemaVersion int           `json:"schemaVersion"`
	Kind          string        `json:"kind"`
	Generator     GeneratorInfo `json:"generator"`
	Query         string        `json:"query"`
	Total         int           `json:"total"`
	Limit         int           `json:"limit"`
	Results       []SearchMatch `json:"results"`
}

type TaskInitReport struct {
	SchemaVersion int           `json:"schemaVersion"`
	Kind          string        `json:"kind"`
	Generator     GeneratorInfo `json:"generator"`
	ID            string        `json:"id"`
	Title         string        `json:"title"`
	Type          string        `json:"type"`
	Language      string        `json:"language"`
	Path          string        `json:"path"`
	ParentID      *string       `json:"parentId"`
}

type TaskHierarchyRef struct {
	ID         string        `json:"id"`
	Title      string        `json:"title"`
	Status     string        `json:"status"`
	WorkState  TaskWorkState `json:"workState"`
	HasBlocker bool          `json:"hasBlocker"`
}

type TaskHierarchy struct {
	Parent      *TaskHierarchyRef      `json:"parent"`
	Ancestors   []TaskHierarchyRef     `json:"ancestors"`
	Children    []TaskHierarchyRef     `json:"children"`
	Descendants TaskDescendantsSummary `json:"descendants"`
}

type TaskTreeNode struct {
	ID          string                  `json:"id"`
	Status      string                  `json:"status"`
	WorkState   TaskWorkState           `json:"workState"`
	Descendants *TaskDescendantsSummary `json:"descendants,omitempty"`
	Title       string                  `json:"title"`
	Children    []TaskTreeNode          `json:"children"`
	statusLabel string
}

type TaskTreeReport struct {
	SchemaVersion int           `json:"schemaVersion"`
	Kind          string        `json:"kind"`
	Generator     GeneratorInfo `json:"generator"`
	TaskID        string        `json:"taskId"`
	Tree          TaskTreeNode  `json:"tree"`
}

type TaskMoveTask struct {
	ID     string     `json:"id"`
	Title  string     `json:"title"`
	Status StatusInfo `json:"status"`
	Type   string     `json:"type,omitempty"`
}

type TaskMoveReport struct {
	SchemaVersion   int           `json:"schemaVersion"`
	Kind            string        `json:"kind"`
	Generator       GeneratorInfo `json:"generator"`
	Status          string        `json:"status"`
	Task            TaskMoveTask  `json:"task"`
	SourcePath      string        `json:"sourcePath,omitempty"`
	DestinationPath string        `json:"destinationPath,omitempty"`
	ArchiveYear     string        `json:"archiveYear,omitempty"`
	Issues          []Issue       `json:"issues"`
}

type ScaffoldReport struct {
	SchemaVersion int           `json:"schemaVersion"`
	Kind          string        `json:"kind"`
	Generator     GeneratorInfo `json:"generator"`
	EntityType    string        `json:"entityType"`
	ID            string        `json:"id"`
	Title         string        `json:"title"`
	Language      string        `json:"language"`
	Path          string        `json:"path"`
}

type TaskReadyReport struct {
	SchemaVersion    int            `json:"schemaVersion"`
	Kind             string         `json:"kind"`
	Generator        GeneratorInfo  `json:"generator"`
	Task             TaskVerifyTask `json:"task"`
	Status           string         `json:"status"`
	ContractComplete bool           `json:"contractComplete"`
	ReadyForWork     bool           `json:"readyForWork"`
	Issues           []Issue        `json:"issues"`
}

type TaskCandidateBlocker struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type TaskCandidate struct {
	ID                    string                  `json:"id"`
	Title                 string                  `json:"title"`
	Status                string                  `json:"status"`
	WorkState             TaskWorkState           `json:"workState"`
	Descendants           *TaskDescendantsSummary `json:"descendants,omitempty"`
	Priority              string                  `json:"priority,omitempty"`
	ParentID              *string                 `json:"parentId,omitempty"`
	ContractComplete      bool                    `json:"contractComplete"`
	DependenciesSatisfied bool                    `json:"dependenciesSatisfied"`
	ReadyForWork          bool                    `json:"readyForWork"`
	BlockedBy             []TaskCandidateBlocker  `json:"blockedBy"`
	Issues                []Issue                 `json:"issues"`
}

type TaskCandidatesReport struct {
	SchemaVersion int             `json:"schemaVersion"`
	Kind          string          `json:"kind"`
	Generator     GeneratorInfo   `json:"generator"`
	ParentTaskID  string          `json:"parentTaskId,omitempty"`
	Candidates    []TaskCandidate `json:"candidates"`
}

type ReportLink struct {
	Destination string `json:"destination"`
	Broken      bool   `json:"broken"`
	Blocked     bool   `json:"blocked"`
	TargetKind  string `json:"targetKind"`
	Target      string `json:"target"`
	Href        string `json:"href"`
}

type ReportDocument struct {
	ID               string       `json:"id"`
	SourcePath       string       `json:"sourcePath"`
	OutputPath       string       `json:"outputPath"`
	Type             string       `json:"type"`
	SectionType      SectionType  `json:"sectionType,omitempty"`
	Title            string       `json:"title"`
	Description      string       `json:"description"`
	Metadata         Metadata     `json:"metadata"`
	Status           StatusInfo   `json:"status"`
	TaskStats        TaskStats    `json:"taskStats"`
	UpdatedAt        time.Time    `json:"updatedAt"`
	Stale            bool         `json:"stale"`
	Warnings         int          `json:"warnings"`
	Errors           int          `json:"errors"`
	Links            []ReportLink `json:"links"`
	Backlinks        []string     `json:"backlinks"`
	RelatedDocuments []string     `json:"relatedDocuments"`
}

type ReportRoadmapStage struct {
	Title       string        `json:"title"`
	Status      StatusInfo    `json:"status"`
	PlannedDate string        `json:"plannedDate"`
	TaskStats   TaskStats     `json:"taskStats"`
	Items       []RoadmapItem `json:"items"`
	Document    string        `json:"document"`
	Anchor      string        `json:"anchor"`
}

type ReportRisk struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Status      StatusInfo `json:"status"`
	Probability string     `json:"probability"`
	Impact      string     `json:"impact"`
	TaskStats   TaskStats  `json:"taskStats"`
	Document    string     `json:"document"`
	Anchor      string     `json:"anchor"`
}

type ReportProject struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      StatusInfo `json:"status"`
	Stage       string     `json:"stage"`
	Updated     string     `json:"updated"`
	Summary     string     `json:"summary"`
}

type ReportKnowledge struct {
	Modules       []KnowledgeModule   `json:"modules"`
	UseCases      []KnowledgeUseCase  `json:"useCases"`
	Flows         []KnowledgeFlow     `json:"flows"`
	Standards     []KnowledgeStandard `json:"standards"`
	Runbooks      []KnowledgeRunbook  `json:"runbooks"`
	BusinessRules []BusinessRule      `json:"businessRules"`
	WorkItems     []WorkItem          `json:"workItems"`
}

type ReportScreen struct {
	ID                  string        `json:"id"`
	Title               string        `json:"title"`
	Description         string        `json:"description,omitempty"`
	Module              string        `json:"module"`
	Type                string        `json:"type"`
	Status              string        `json:"status"`
	Route               string        `json:"route,omitempty"`
	Preview             string        `json:"preview,omitempty"`
	Component           string        `json:"component,omitempty"`
	Updated             string        `json:"updated,omitempty"`
	Parent              string        `json:"parent,omitempty"`
	States              []ScreenState `json:"states"`
	IncomingTransitions []string      `json:"incomingTransitions"`
	OutgoingTransitions []string      `json:"outgoingTransitions"`
	UseCases            []string      `json:"useCases"`
	WorkItems           []string      `json:"workItems"`
	Contracts           []string      `json:"contracts"`
	Document            string        `json:"document"`
}

type ProjectReport struct {
	SchemaVersion    int                  `json:"schemaVersion"`
	Generator        GeneratorInfo        `json:"generator"`
	GeneratedAt      time.Time            `json:"generatedAt"`
	SourceDirectory  string               `json:"sourceDirectory"`
	StaleDays        int                  `json:"staleDays"`
	Project          ReportProject        `json:"project"`
	CurrentStatus    CurrentStatus        `json:"currentStatus"`
	Stats            Stats                `json:"stats"`
	Documents        []ReportDocument     `json:"documents"`
	Roadmap          []ReportRoadmapStage `json:"roadmap"`
	Risks            []ReportRisk         `json:"risks"`
	Knowledge        ReportKnowledge      `json:"knowledge"`
	Screens          []ReportScreen       `json:"screens"`
	Transitions      []ScreenTransition   `json:"transitions"`
	PlayableFlows    []PlayableFlow       `json:"playableFlows"`
	Hotspots         []Hotspot            `json:"hotspots"`
	ErrorDefinitions []ErrorDefinition    `json:"errorDefinitions"`
	Traceability     []TraceabilityRow    `json:"traceability"`
	Issues           []Issue              `json:"issues"`
}

type TaskContextDocument struct {
	ID          string               `json:"id,omitempty"`
	Path        string               `json:"path"`
	Type        string               `json:"type"`
	Title       string               `json:"title"`
	Description string               `json:"description"`
	Status      StatusInfo           `json:"status"`
	Sections    []TaskContextSection `json:"sections"`
}

type TaskContextSection struct {
	Title    string `json:"title"`
	Markdown string `json:"markdown"`
}

type TaskContextReport struct {
	SchemaVersion     int                   `json:"schemaVersion"`
	Kind              string                `json:"kind"`
	Generator         GeneratorInfo         `json:"generator"`
	Task              WorkItem              `json:"task"`
	Hierarchy         TaskHierarchy         `json:"hierarchy"`
	FullVerification  bool                  `json:"fullVerification"`
	Module            *KnowledgeModule      `json:"module,omitempty"`
	UseCase           *KnowledgeUseCase     `json:"useCase,omitempty"`
	Flow              *KnowledgeFlow        `json:"flow,omitempty"`
	Standards         []KnowledgeStandard   `json:"standards"`
	Runbooks          []KnowledgeRunbook    `json:"runbooks"`
	Screens           []KnowledgeScreen     `json:"screens"`
	ScreenTransitions []ScreenTransition    `json:"screenTransitions"`
	BusinessRules     []BusinessRule        `json:"businessRules"`
	Dependencies      []WorkItem            `json:"dependencies"`
	Dependents        []WorkItem            `json:"dependents"`
	Documents         []TaskContextDocument `json:"documents"`
	Issues            []Issue               `json:"issues"`
	RequiredReads     []string              `json:"requiredReads"`
}
