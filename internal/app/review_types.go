package toudocu

import "time"

const (
	reviewSchemaVersion     = 1
	reviewStoreVersion      = 1
	reviewSnapshotLimit     = 2 << 20
	reviewMessageLimit      = 64 << 10
	reviewResponseLimit     = 64 << 10
	reviewRequestLimit      = 104 << 10
	reviewSelectionLimit    = 32 << 10
	reviewContextByteSize   = 2 << 10
	reviewClaimLease        = 30 * time.Minute
	reviewResolvedRetention = 30 * 24 * time.Hour
)

// RepositoryReviewReport is an internal repository-wide projection used by
// the Changes workspace. Agent Feedback never stores this projection.
type RepositoryReviewReport struct {
	SchemaVersion      int                    `json:"schemaVersion"`
	Repository         ChangeRepository       `json:"repository"`
	Comparison         ChangeComparison       `json:"comparison"`
	RepositoryRevision string                 `json:"repositoryRevision"`
	Summary            ChangeSummary          `json:"summary"`
	Files              []RepositoryReviewFile `json:"files"`
	FeedbackWritable   bool                   `json:"feedbackWritable"`
	Diagnostics        []Issue                `json:"diagnostics"`
}

type RepositoryReviewFile struct {
	Status        string               `json:"status"`
	Path          string               `json:"path"`
	OldPath       string               `json:"oldPath,omitempty"`
	GitState      ChangeGitState       `json:"gitState"`
	Lines         ChangeLineStats      `json:"lines"`
	Binary        bool                 `json:"binary"`
	Size          int64                `json:"size,omitempty"`
	Digest        string               `json:"digest,omitempty"`
	Language      string               `json:"language"`
	Documentation *DocumentationChange `json:"documentation,omitempty"`
}

type RepositoryReviewFileDetail struct {
	SchemaVersion      int                  `json:"schemaVersion"`
	RepositoryRevision string               `json:"repositoryRevision"`
	File               RepositoryReviewFile `json:"file"`
	Before             *string              `json:"before,omitempty"`
	Current            *string              `json:"current,omitempty"`
	RenderedBefore     *string              `json:"renderedBefore,omitempty"`
	RenderedCurrent    *string              `json:"renderedCurrent,omitempty"`
	Patch              string               `json:"patch,omitempty"`
	Hunks              []SourceDiffHunk     `json:"hunks"`
	Documentation      *DocumentationChange `json:"documentation,omitempty"`
}

type ReviewState struct {
	SchemaVersion      int             `json:"schemaVersion"`
	StoreVersion       int             `json:"storeVersion,omitempty"`
	Revision           uint64          `json:"revision"`
	StateDigest        string          `json:"stateDigest"`
	RepositoryRevision string          `json:"repositoryRevision,omitempty"`
	DocsPath           string          `json:"docsPath,omitempty"`
	NextSequence       uint64          `json:"nextSequence,omitempty"`
	Session            *ReviewSession  `json:"session,omitempty"`
	Deliveries         []AgentDelivery `json:"deliveries"`
}

type ReviewSession struct {
	ID          string       `json:"id"`
	CreatedAt   time.Time    `json:"createdAt"`
	Discussions []Discussion `json:"discussions"`
}

type Discussion struct {
	ID        string          `json:"id"`
	State     string          `json:"state"`
	Target    ReviewTarget    `json:"target"`
	Anchor    *DocumentAnchor `json:"anchor,omitempty"`
	Placement AnchorPlacement `json:"placement"`
	Messages  []ReviewMessage `json:"messages"`
	CreatedAt time.Time       `json:"createdAt"`
	UpdatedAt time.Time       `json:"updatedAt"`
}

type ReviewPosition struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

type ReviewRange struct {
	Start ReviewPosition `json:"start"`
	End   ReviewPosition `json:"end"`
}

type ReviewTarget struct {
	Kind       string       `json:"kind"`
	Path       string       `json:"path"`
	DocumentID string       `json:"documentId,omitempty"`
	Range      *ReviewRange `json:"range,omitempty"`
}

type DocumentAnchor struct {
	Kind          string       `json:"kind"`
	Path          string       `json:"path"`
	DocumentID    string       `json:"documentId,omitempty"`
	SourceDigest  string       `json:"sourceDigest"`
	Range         *ReviewRange `json:"range,omitempty"`
	SelectedText  string       `json:"selectedText,omitempty"`
	ContextBefore string       `json:"contextBefore,omitempty"`
	ContextAfter  string       `json:"contextAfter,omitempty"`
}

type AnchorPlacement struct {
	Status string       `json:"status"`
	Path   string       `json:"path"`
	Range  *ReviewRange `json:"range,omitempty"`
	Reason string       `json:"reason,omitempty"`
}

type ReviewMessage struct {
	ID           string          `json:"id"`
	Author       string          `json:"author"`
	Intent       string          `json:"intent,omitempty"`
	State        string          `json:"state,omitempty"`
	Text         string          `json:"text"`
	DeliveryID   string          `json:"deliveryId,omitempty"`
	Outcome      string          `json:"outcome,omitempty"`
	Evidence     []AgentEvidence `json:"evidence"`
	ChangedPaths []string        `json:"changedPaths"`
	CreatedAt    time.Time       `json:"createdAt"`
	EditedAt     *time.Time      `json:"editedAt,omitempty"`
	SubmittedAt  *time.Time      `json:"submittedAt,omitempty"`
}

type AgentDelivery struct {
	SchemaVersion  int        `json:"schemaVersion"`
	ID             string     `json:"id"`
	Sequence       uint64     `json:"sequence"`
	State          string     `json:"state"`
	DiscussionID   string     `json:"discussionId"`
	MessageIDs     []string   `json:"messageIds"`
	CreatedAt      time.Time  `json:"createdAt"`
	ClaimedAt      *time.Time `json:"claimedAt,omitempty"`
	LeaseExpiresAt *time.Time `json:"leaseExpiresAt,omitempty"`
	RespondedAt    *time.Time `json:"respondedAt,omitempty"`
	ResponseDigest string     `json:"responseDigest,omitempty"`
}

type AgentEvidence struct {
	Path      string `json:"path"`
	StartLine int    `json:"startLine,omitempty"`
	EndLine   int    `json:"endLine,omitempty"`
}

type AgentResponse struct {
	SchemaVersion int             `json:"schemaVersion"`
	DeliveryID    string          `json:"deliveryId"`
	DiscussionID  string          `json:"discussionId"`
	Outcome       string          `json:"outcome"`
	Message       string          `json:"message"`
	Evidence      []AgentEvidence `json:"evidence"`
	ChangedPaths  []string        `json:"changedPaths"`
}

type AgentRequest struct {
	SchemaVersion int                 `json:"schemaVersion"`
	Pending       bool                `json:"pending"`
	DeliveryID    string              `json:"deliveryId,omitempty"`
	Discussion    *Discussion         `json:"discussion,omitempty"`
	Target        *AgentRequestTarget `json:"target,omitempty"`
	Repository    *AgentRepository    `json:"repository,omitempty"`
	PendingCount  int                 `json:"pendingCount,omitempty"`
	HasMore       bool                `json:"hasMore,omitempty"`
}

type AgentRequestTarget struct {
	Kind         string       `json:"kind"`
	Path         string       `json:"path"`
	DocumentID   string       `json:"documentId,omitempty"`
	AnchorState  string       `json:"anchorState"`
	Range        *ReviewRange `json:"range,omitempty"`
	SelectedText string       `json:"selectedText,omitempty"`
}

type AgentRepository struct {
	Head string `json:"head"`
}

type ReviewMutationGuard struct {
	ExpectedRevision    uint64 `json:"expectedRevision"`
	ExpectedStateDigest string `json:"expectedStateDigest"`
}

type SelectionHint struct {
	SelectedText string `json:"selectedText"`
	Occurrence   int    `json:"occurrence,omitempty"`
}

type CreateDiscussionRequest struct {
	ReviewMutationGuard
	Target    ReviewTarget   `json:"target"`
	Selection *SelectionHint `json:"selection,omitempty"`
	Intent    string         `json:"intent"`
	Text      string         `json:"text"`
}

type CreateMessageRequest struct {
	ReviewMutationGuard
	Intent string `json:"intent"`
	Text   string `json:"text"`
}

type UpdateMessageRequest struct {
	ReviewMutationGuard
	Intent string `json:"intent"`
	Text   string `json:"text"`
}

type UpdateDiscussionRequest struct {
	ReviewMutationGuard
	State string `json:"state"`
}

type reviewFailure struct {
	Code    string
	Status  int
	Message string
}

func (failure *reviewFailure) Error() string { return failure.Message }
