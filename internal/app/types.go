package toudocu

import "time"

// Options configures one CLI operation or a direct library call.
type Options struct {
	Command                    string
	TaskID                     string
	Query                      string
	EntityKind                 string
	EntityID                   string
	Area                       string
	TaskType                   string
	ParentTaskID               string
	Language                   string
	Limit                      int
	VerifyMode                 string
	Target                     string
	ChangeBase                 string
	ChangeTarget               string
	ChangeBranchBase           string
	ChangeFile                 string
	ChangeTaskID               string
	ChangeTaskTree             bool
	ChangeStatus               string
	ChangeEntityType           string
	ChangeModule               string
	ChangeOutput               string
	ChangePermanentOnly        bool
	ChangeRenameSimilarity     int
	ChangeMaxSourceDiffBytes   int
	ChangeMaxRenderedFileBytes int
	ChangeIncludeTaskArtifacts bool
	ChangeIncludeAssets        bool
	// ChangeForceIncludeAssets overrides changes.includeAssets for workflows that
	// require a complete binary-asset change set.
	ChangeForceIncludeAssets bool
	// ChangeTranslationInput returns every reader-facing translation input,
	// independent of configurable task, asset, and exclude filters.
	ChangeTranslationInput bool
	ChangeSemanticDiff     bool
	ChangeRenderedDiff     bool
	ChangeExclude          []string
	ChangeOmitSourceDiff   bool
	InputDirectory         string
	OutputDirectory        string
	Title                  string
	Excludes               []string
	StaleDays              int
	RepositoryRoot         string
	RepositoryURL          string
	RepositoryRef          string
	Clean                  bool
	Open                   bool
	Strict                 bool
	NoScreenMap            bool
	NoUpdateCheck          bool
	Format                 string
	ReportPath             string
	Timeout                time.Duration
	Host                   string
	Port                   int
	Example                bool
	Now                    time.Time
}

type StatusInfo struct {
	Kind       string `json:"kind"`
	Symbol     string `json:"symbol"`
	Label      string `json:"label"`
	Recognized bool   `json:"recognized"`
}

type Metadata map[string]string

type MetadataExtra struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type SectionKind string

const (
	SectionKindSummary                    SectionKind = "summary"
	SectionKindAcceptanceCriteria         SectionKind = "acceptance-criteria"
	SectionKindVerification               SectionKind = "verification"
	SectionKindRules                      SectionKind = "rules"
	SectionKindAutomatedChecks            SectionKind = "automated-checks"
	SectionKindPrerequisites              SectionKind = "prerequisites"
	SectionKindProcedure                  SectionKind = "procedure"
	SectionKindRollback                   SectionKind = "rollback"
	SectionKindStopConditions             SectionKind = "stop-conditions"
	SectionKindMainScenario               SectionKind = "main-scenario"
	SectionKindPostconditions             SectionKind = "postconditions"
	SectionKindBusinessRules              SectionKind = "business-rules"
	SectionKindImplementation             SectionKind = "implementation"
	SectionKindCodeLocation               SectionKind = "code-location"
	SectionKindBoundaries                 SectionKind = "boundaries"
	SectionKindInvariants                 SectionKind = "invariants"
	SectionKindStableInterfaces           SectionKind = "stable-interfaces"
	SectionKindRelatedUseCases            SectionKind = "related-use-cases"
	SectionKindContext                    SectionKind = "context"
	SectionKindDecision                   SectionKind = "decision"
	SectionKindConsequences               SectionKind = "consequences"
	SectionKindResult                     SectionKind = "result"
	SectionKindBehaviorChange             SectionKind = "behavior-change"
	SectionKindBefore                     SectionKind = "before"
	SectionKindAfter                      SectionKind = "after"
	SectionKindScope                      SectionKind = "scope"
	SectionKindOutOfScope                 SectionKind = "out-of-scope"
	SectionKindPlan                       SectionKind = "plan"
	SectionKindDocumentationImpact        SectionKind = "documentation-impact"
	SectionKindBlocker                    SectionKind = "blocker"
	SectionKindCancellationReason         SectionKind = "cancellation-reason"
	SectionKindUseCaseOmissionReason      SectionKind = "use-case-omission-reason"
	SectionKindSymptom                    SectionKind = "symptom"
	SectionKindExpectedBehavior           SectionKind = "expected-behavior"
	SectionKindActualBehavior             SectionKind = "actual-behavior"
	SectionKindStepsToReproduce           SectionKind = "steps-to-reproduce"
	SectionKindEvidence                   SectionKind = "evidence"
	SectionKindCause                      SectionKind = "cause"
	SectionKindRegressionTest             SectionKind = "regression-test"
	SectionKindRelationshipToUserBehavior SectionKind = "relationship-to-user-behavior"
	SectionKindRoadmapStage               SectionKind = "roadmap-stage"
	SectionKindRisk                       SectionKind = "risk"
)

type Heading struct {
	Level       int    `json:"level"`
	Title       string `json:"title"`
	ID          string `json:"id"`
	Line        int    `json:"line"`
	startOffset int
	endOffset   int
}

type Task struct {
	Line         int    `json:"line"`
	Indent       int    `json:"indent"`
	Completed    bool   `json:"completed"`
	Text         string `json:"text"`
	HeadingID    string `json:"headingId,omitempty"`
	HeadingTitle string `json:"headingTitle,omitempty"`
}

type Link struct {
	Line        int    `json:"line"`
	Image       bool   `json:"image"`
	Label       string `json:"label"`
	Destination string `json:"destination"`
	Title       string `json:"title,omitempty"`
}

type ResolvedLink struct {
	Link
	Href             string
	External         bool
	Broken           bool
	Blocked          bool
	BrokenAnchor     bool
	RepositoryEscape bool
	RepositoryAsset  bool
	ActiveAsset      bool
	UnsafeImage      bool
	RepositoryPath   string
	RepositoryKind   string
	AssetPath        string
	GeneratedTarget  string
	TargetDocument   *Document
}

type Section struct {
	Kind           SectionKind     `json:"kind,omitempty"`
	Title          string          `json:"title"`
	ID             string          `json:"id"`
	StartLine      int             `json:"startLine"`
	EndLine        int             `json:"endLine"`
	Metadata       Metadata        `json:"metadata"`
	MetadataExtras []MetadataExtra `json:"metadataExtras,omitempty"`
	Tasks          []Task          `json:"tasks"`
	Text           string          `json:"text"`
	Markdown       string          `json:"markdown"`
	children       []Section
}

type TaskStats struct {
	Total     int  `json:"total"`
	Completed int  `json:"completed"`
	Remaining int  `json:"remaining"`
	Percent   *int `json:"percent"`
}

type Issue struct {
	Severity     string `json:"severity"`
	Code         string `json:"code"`
	Message      string `json:"message"`
	Migration    string `json:"migration,omitempty"`
	DocumentPath string `json:"documentPath,omitempty"`
	Line         int    `json:"line,omitempty"`
	Column       int    `json:"column,omitempty"`
	TaskID       string `json:"taskId,omitempty"`
	RelatedID    string `json:"relatedId,omitempty"`
}

type Document struct {
	ID                  string
	AbsolutePath        string
	SourcePath          string
	OutputPath          string
	Directory           string
	FileName            string
	Type                string
	SectionType         SectionType
	TypeLabel           string
	Title               string
	Description         string
	Content             string
	Headings            []Heading
	Sections            []Section
	Metadata            Metadata
	MetadataExtras      []MetadataExtra
	metadataLocations   map[string]int
	metadataCounts      map[string]int
	metadataBlocks      int
	markdownDiagnostics []Issue
	mermaidBlocks       []mermaidBlock
	markdownTables      []markdownTable
	Tasks               []Task
	TaskStats           TaskStats
	Links               []Link
	PlainText           string
	MTime               time.Time
	UpdatedAt           time.Time
	AgeDays             int
	Stale               bool
	Status              StatusInfo
	Warnings            []Issue
	Errors              []Issue
	ResolvedLinks       []ResolvedLink
	Backlinks           []*Document
	RelatedDocuments    []*Document
	LinkedUseCases      []*Document
	LinkedModules       []*Document
}

type Risk struct {
	ID          string
	Title       string
	FullTitle   string
	Status      StatusInfo
	Probability string
	Impact      string
	TaskStats   TaskStats
	Document    *Document
	Anchor      string
	Text        string
}

type RoadmapStage struct {
	Title       string
	Status      StatusInfo
	PlannedDate string
	TaskStats   TaskStats
	Items       []RoadmapItem
	Document    *Document
	Anchor      string
	Text        string
}

type RoadmapItem struct {
	ID                 string      `json:"id"`
	Text               string      `json:"text"`
	Kind               string      `json:"kind"`
	DeclaredCompleted  bool        `json:"declaredCompleted"`
	EffectiveCompleted bool        `json:"effectiveCompleted"`
	CompletionSource   string      `json:"completionSource"`
	TargetDocument     string      `json:"targetDocument,omitempty"`
	TargetStatus       *StatusInfo `json:"targetStatus,omitempty"`
	Document           string      `json:"document"`
	Line               int         `json:"line"`
}

type KnowledgeModule struct {
	ID              string     `json:"id"`
	Title           string     `json:"title"`
	Status          StatusInfo `json:"status"`
	Document        string     `json:"document"`
	RepositoryPaths []string   `json:"repositoryPaths"`
	UseCaseIDs      []string   `json:"useCaseIds"`
	ScreenIDs       []string   `json:"screenIds"`
	BusinessRuleIDs []string   `json:"businessRuleIds"`
}

type KnowledgeUseCase struct {
	ID              string     `json:"id"`
	Title           string     `json:"title"`
	Status          StatusInfo `json:"status"`
	ModuleID        string     `json:"moduleId,omitempty"`
	Document        string     `json:"document"`
	RepositoryPaths []string   `json:"repositoryPaths"`
	BusinessRuleIDs []string   `json:"businessRuleIds"`
	FlowIDs         []string   `json:"flowIds"`
	ScreenIDs       []string   `json:"screenIds"`
	StartScreenID   string     `json:"startScreen,omitempty"`
	TerminalScreens []string   `json:"terminalScreens"`
	AllowCycle      bool       `json:"allowCycle"`
}

type KnowledgeFlow struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	ModuleID   string   `json:"moduleId,omitempty"`
	UseCaseIDs []string `json:"useCaseIds"`
	Document   string   `json:"document"`
}

type KnowledgeStandard struct {
	ID              string     `json:"id"`
	Title           string     `json:"title"`
	Status          StatusInfo `json:"status"`
	Scope           string     `json:"scope,omitempty"`
	Updated         string     `json:"updated,omitempty"`
	SupersededBy    string     `json:"supersededBy,omitempty"`
	Rules           string     `json:"rules,omitempty"`
	AutomaticChecks string     `json:"automaticChecks,omitempty"`
	Document        string     `json:"document"`
}

type KnowledgeRunbook struct {
	ID           string     `json:"id"`
	Title        string     `json:"title"`
	Status       StatusInfo `json:"status"`
	Environment  string     `json:"environment,omitempty"`
	Risk         string     `json:"risk,omitempty"`
	LastVerified string     `json:"lastVerified,omitempty"`
	Freshness    string     `json:"freshness"`
	Document     string     `json:"document"`
}

type ScreenState struct {
	ID      string `json:"id"`
	Title   string `json:"title,omitempty"`
	Preview string `json:"preview,omitempty"`
}

type KnowledgeScreen struct {
	ID                    string        `json:"id"`
	Title                 string        `json:"title"`
	Description           string        `json:"description,omitempty"`
	ModuleID              string        `json:"module"`
	Kind                  string        `json:"type"`
	Route                 string        `json:"route,omitempty"`
	Status                StatusInfo    `json:"status"`
	Preview               string        `json:"preview,omitempty"`
	Component             string        `json:"component,omitempty"`
	Updated               string        `json:"updated,omitempty"`
	ParentID              string        `json:"parent,omitempty"`
	States                []ScreenState `json:"states"`
	Document              string        `json:"document"`
	UseCaseIDs            []string      `json:"useCases"`
	WorkItemIDs           []string      `json:"workItems"`
	ContractDocuments     []string      `json:"contracts"`
	IncomingTransitionIDs []string      `json:"incomingTransitions"`
	OutgoingTransitionIDs []string      `json:"outgoingTransitions"`
	Reachable             bool          `json:"reachable"`
	Line                  int           `json:"line"`
}

type ScreenTransition struct {
	ID        string `json:"id"`
	UseCaseID string `json:"useCase,omitempty"`
	FromID    string `json:"source"`
	ToID      string `json:"target"`
	Action    string `json:"action"`
	Condition string `json:"condition"`
	StateID   string `json:"state,omitempty"`
	ErrorID   string `json:"error,omitempty"`
	Message   string `json:"message,omitempty"`
	Contract  string `json:"contract,omitempty"`
	Kind      string `json:"type"`
	Document  string `json:"document"`
	Line      int    `json:"line"`
}

type PlayableFlow struct {
	UseCaseID        string   `json:"useCase"`
	StartScreenID    string   `json:"startScreen"`
	ReachableScreens []string `json:"reachableScreens"`
	TerminalScreens  []string `json:"terminalScreens"`
	TransitionIDs    []string `json:"transitions"`
	Result           string   `json:"result,omitempty"`
	Valid            bool     `json:"valid"`
	IssueCodes       []string `json:"issueCodes"`
}

type Hotspot struct {
	ScreenID       string  `json:"screen"`
	TransitionID   string  `json:"transition"`
	X              float64 `json:"x"`
	Y              float64 `json:"y"`
	Width          float64 `json:"width"`
	Height         float64 `json:"height"`
	AllowDuplicate bool    `json:"allowDuplicate,omitempty"`
}

type ErrorDefinition struct {
	ID       string `json:"id"`
	Message  string `json:"message"`
	Document string `json:"document"`
	Line     int    `json:"line"`
}

type TraceabilityRow struct {
	UseCaseID    string `json:"useCase,omitempty"`
	ScreenID     string `json:"screen"`
	TransitionID string `json:"transition"`
	TaskID       string `json:"task"`
	CriterionID  string `json:"criterion"`
	Verification string `json:"verification"`
}

type BusinessRule struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	ModuleID string `json:"moduleId,omitempty"`
	Document string `json:"document"`
	Anchor   string `json:"anchor"`
	Line     int    `json:"line"`
	ownerDoc *Document
}

type WorkItemStatus string
type WorkItemID string
type ModuleID string

const (
	WorkItemDraft      WorkItemStatus = "draft"
	WorkItemReady      WorkItemStatus = "ready"
	WorkItemInProgress WorkItemStatus = "in-progress"
	WorkItemBlocked    WorkItemStatus = "blocked"
	WorkItemDone       WorkItemStatus = "done"
	WorkItemCancelled  WorkItemStatus = "cancelled"
)

type WorkItemType string

const (
	WorkItemFeature       WorkItemType = "feature"
	WorkItemBug           WorkItemType = "bug"
	WorkItemMaintenance   WorkItemType = "maintenance"
	WorkItemDocumentation WorkItemType = "documentation"
	WorkItemResearch      WorkItemType = "research"
)

type WorkItem struct {
	ID                  string                  `json:"id"`
	Title               string                  `json:"title"`
	Status              StatusInfo              `json:"status"`
	Type                string                  `json:"type,omitempty"`
	Archived            bool                    `json:"archived"`
	ArchiveYear         string                  `json:"archiveYear,omitempty"`
	Priority            string                  `json:"priority,omitempty"`
	Severity            string                  `json:"severity,omitempty"`
	Reproducibility     string                  `json:"reproducibility,omitempty"`
	Regression          string                  `json:"regression,omitempty"`
	Updated             string                  `json:"updated,omitempty"`
	ModuleID            string                  `json:"moduleId,omitempty"`
	UseCaseID           string                  `json:"useCaseId,omitempty"`
	FlowID              string                  `json:"flowId,omitempty"`
	ScreenIDs           []string                `json:"screenIds"`
	TransitionIDs       []string                `json:"transitionIds"`
	StandardIDs         []string                `json:"standardIds"`
	RunbookIDs          []string                `json:"runbookIds"`
	DependsOn           []string                `json:"dependsOn"`
	ParentID            *string                 `json:"parentId"`
	ChildIDs            []string                `json:"childIds"`
	Document            string                  `json:"document"`
	Anchor              string                  `json:"anchor"`
	Criteria            []Task                  `json:"criteria"`
	Verification        []CriterionVerification `json:"verificationMatrix"`
	Checks              []VerificationCheck     `json:"checks"`
	RepositoryPaths     []string                `json:"repositoryPaths"`
	Result              string                  `json:"result,omitempty"`
	BehaviorChange      string                  `json:"behaviorChange,omitempty"`
	Before              string                  `json:"before,omitempty"`
	After               string                  `json:"after,omitempty"`
	OutOfScope          string                  `json:"outOfScope,omitempty"`
	Plan                string                  `json:"plan,omitempty"`
	DocumentationImpact string                  `json:"documentationImpact,omitempty"`
	DocumentationPaths  []string                `json:"documentationPaths"`
	Blocker             string                  `json:"blocker,omitempty"`
	line                int
	parentLine          int
	parentCount         int
	ownerDoc            *Document
	statusName          WorkItemStatus
	useCaseOmitted      bool
}

type CriterionVerification struct {
	CriterionID string   `json:"criterionId"`
	Criterion   string   `json:"criterion"`
	Completed   bool     `json:"completed"`
	Commands    []string `json:"commands"`
	Transitions []string `json:"transitions"`
	References  []string `json:"verificationReferences"`
}

type VerificationCheck struct {
	Target   string   `json:"target"`
	Commands []string `json:"commands"`
	Line     int      `json:"line"`
}

type KnowledgeModel struct {
	Modules       []KnowledgeModule   `json:"modules"`
	UseCases      []KnowledgeUseCase  `json:"useCases"`
	Flows         []KnowledgeFlow     `json:"flows"`
	Standards     []KnowledgeStandard `json:"standards"`
	Runbooks      []KnowledgeRunbook  `json:"runbooks"`
	Screens       []KnowledgeScreen   `json:"screens"`
	Transitions   []ScreenTransition  `json:"screenTransitions"`
	BusinessRules []BusinessRule      `json:"businessRules"`
	WorkItems     []WorkItem          `json:"workItems"`
	PlayableFlows []PlayableFlow      `json:"playableFlows"`
	Hotspots      []Hotspot           `json:"hotspots"`
	Errors        []ErrorDefinition   `json:"errorDefinitions"`
	Traceability  []TraceabilityRow   `json:"traceability"`
}

type ProjectInfo struct {
	Title            string
	Description      string
	Status           StatusInfo
	Stage            string
	Updated          string
	Summary          string
	OverviewDocument *Document
	StatusDocument   *Document
}

type CurrentWorkItem struct {
	ID       string     `json:"id"`
	Title    string     `json:"title"`
	Status   StatusInfo `json:"status"`
	ModuleID string     `json:"moduleId,omitempty"`
	Document string     `json:"document"`
	Anchor   string     `json:"anchor"`
}

type CurrentBlocker struct {
	TaskID   string `json:"taskId"`
	Text     string `json:"text"`
	Document string `json:"document"`
	Anchor   string `json:"anchor"`
}

type CurrentStatus struct {
	ActiveWork []CurrentWorkItem `json:"activeWork"`
	Blockers   []CurrentBlocker  `json:"blockers"`
	NextResult *RoadmapItem      `json:"nextResult,omitempty"`
}

type Stats struct {
	Documents                   int            `json:"documents"`
	TotalTasks                  int            `json:"totalTasks"`
	CompletedTasks              int            `json:"completedTasks"`
	RemainingTasks              int            `json:"remainingTasks"`
	TaskProgress                *int           `json:"taskProgress"`
	DocumentsComplete           int            `json:"documentsComplete"`
	DocumentsInProgress         int            `json:"documentsInProgress"`
	DocumentsNotStarted         int            `json:"documentsNotStarted"`
	DocumentsWithoutTasks       int            `json:"documentsWithoutTasks"`
	StaleDocuments              int            `json:"staleDocuments"`
	DocumentsWithoutDescription int            `json:"documentsWithoutDescription"`
	DocumentsWithoutStatus      int            `json:"documentsWithoutStatus"`
	BrokenLinks                 int            `json:"brokenLinks"`
	Warnings                    int            `json:"warnings"`
	Errors                      int            `json:"errors"`
	Modules                     int            `json:"modules"`
	ModuleStatuses              map[string]int `json:"moduleStatuses"`
	UseCases                    int            `json:"useCases"`
	UseCaseStatuses             map[string]int `json:"useCaseStatuses"`
	Screens                     int            `json:"screens"`
	ScreensDone                 int            `json:"screensDone"`
	ScreensInProgress           int            `json:"screensInProgress"`
	ScreensPlanned              int            `json:"screensPlanned"`
	ScreensUnreachable          int            `json:"screensUnreachable"`
	Risks                       int            `json:"risks"`
	OpenRisks                   int            `json:"openRisks"`
	Decisions                   int            `json:"decisions"`
	OpenBugs                    int            `json:"openBugs"`
	CriticalBugs                int            `json:"criticalBugs"`
	HighSeverityBugs            int            `json:"highSeverityBugs"`
	RegressionBugs              int            `json:"regressionBugs"`
	UnreproducedBugs            int            `json:"unreproducedBugs"`
	BlockedBugs                 int            `json:"blockedBugs"`
	RunbooksTotal               int            `json:"runbooksTotal"`
	RunbooksRecent              int            `json:"runbooksRecent"`
	RunbooksReviewRequired      int            `json:"runbooksReviewRequired"`
	RunbooksOverdue             int            `json:"runbooksOverdue"`
}

type SearchItem struct {
	Title       string `json:"title"`
	Path        string `json:"path"`
	URL         string `json:"url"`
	Type        string `json:"type"`
	TypeLabel   string `json:"typeLabel"`
	Status      string `json:"status"`
	Archived    bool   `json:"archived"`
	ArchiveYear string `json:"archiveYear,omitempty"`
	Description string `json:"description"`
	Text        string `json:"text"`
}

type Model struct {
	RootDirectory       string
	RepositoryRoot      string
	RepositoryURL       string
	RepositoryRef       string
	GeneratedAt         time.Time
	StaleDays           int
	Documents           []*Document
	DocByPath           map[string]*Document
	Directories         map[string]struct{}
	Assets              map[string]string
	BrandingAssets      map[string]string
	SiteConfig          SiteConfig
	Issues              []Issue
	Collections         map[string][]*Document
	Risks               []Risk
	RoadmapStages       []RoadmapStage
	Knowledge           KnowledgeModel
	Project             ProjectInfo
	CurrentStatus       CurrentStatus
	Stats               Stats
	SearchIndex         []SearchItem
	ProjectChangelog    *Document
	HealthOutputPath    string
	ReportOutputPath    string
	ScreenMapEnabled    bool
	sourceOverlay       map[string][]byte
	serveMode           bool
	serveBaseURL        string
	serveRevision       string
	updateCheckEnabled  bool
	agentConsoleEnabled bool
	taskActionsEnabled  bool
	languageTargets     map[string][]LanguageTarget
	strictPolicy        bool
	openAPIContracts    []OpenAPIContract
}

type GenerateResult struct {
	OutputDirectory string
	Pages           int
	Assets          int
}

type LinkResolution struct {
	Href     string
	External bool
	Broken   bool
	Blocked  bool
}

type LinkResolver func(destination string, image bool, title string) LinkResolution
