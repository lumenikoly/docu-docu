// Package toudocu exposes the typed Go facade for the Toudocu CLI.
//
// The implementation is kept in internal/app so command and library consumers
// use this package without coupling to the repository layout.
package toudocu

import (
	"io"
	core "toudocu/internal/app"
)

const Version = core.Version

var EmbeddedFiles = core.EmbeddedFiles

type (
	Options                  = core.Options
	StatusInfo               = core.StatusInfo
	Metadata                 = core.Metadata
	MetadataExtra            = core.MetadataExtra
	Heading                  = core.Heading
	Task                     = core.Task
	Link                     = core.Link
	ResolvedLink             = core.ResolvedLink
	Section                  = core.Section
	TaskStats                = core.TaskStats
	Issue                    = core.Issue
	Document                 = core.Document
	Risk                     = core.Risk
	RoadmapStage             = core.RoadmapStage
	RoadmapItem              = core.RoadmapItem
	KnowledgeModule          = core.KnowledgeModule
	KnowledgeUseCase         = core.KnowledgeUseCase
	KnowledgeFlow            = core.KnowledgeFlow
	KnowledgeStandard        = core.KnowledgeStandard
	KnowledgeRunbook         = core.KnowledgeRunbook
	ScreenState              = core.ScreenState
	KnowledgeScreen          = core.KnowledgeScreen
	ScreenTransition         = core.ScreenTransition
	PlayableFlow             = core.PlayableFlow
	Hotspot                  = core.Hotspot
	ErrorDefinition          = core.ErrorDefinition
	TraceabilityRow          = core.TraceabilityRow
	BusinessRule             = core.BusinessRule
	WorkItem                 = core.WorkItem
	CriterionVerification    = core.CriterionVerification
	VerificationCheck        = core.VerificationCheck
	KnowledgeModel           = core.KnowledgeModel
	ProjectInfo              = core.ProjectInfo
	CurrentWorkItem          = core.CurrentWorkItem
	CurrentBlocker           = core.CurrentBlocker
	CurrentStatus            = core.CurrentStatus
	Stats                    = core.Stats
	SearchItem               = core.SearchItem
	Model                    = core.Model
	GenerateResult           = core.GenerateResult
	FooterConfig             = core.FooterConfig
	HeroConfig               = core.HeroConfig
	ChangesConfig            = core.ChangesConfig
	TranslationProfile       = core.TranslationProfile
	SiteConfig               = core.SiteConfig
	GeneratorInfo            = core.GeneratorInfo
	TaskVerifyTask           = core.TaskVerifyTask
	CommandExecutionResult   = core.CommandExecutionResult
	CriterionExecutionResult = core.CriterionExecutionResult
	TargetExecutionResult    = core.TargetExecutionResult
	TaskVerifySummary        = core.TaskVerifySummary
	TaskVerifyReport         = core.TaskVerifyReport
	SearchMatch              = core.SearchMatch
	SearchReport             = core.SearchReport
	TaskInitReport           = core.TaskInitReport
	TaskHierarchyRef         = core.TaskHierarchyRef
	TaskHierarchySummary     = core.TaskHierarchySummary
	TaskHierarchy            = core.TaskHierarchy
	TaskTreeNode             = core.TaskTreeNode
	TaskTreeReport           = core.TaskTreeReport
	TaskMoveTask             = core.TaskMoveTask
	TaskMoveReport           = core.TaskMoveReport
	ScaffoldReport           = core.ScaffoldReport
	TaskReadyReport          = core.TaskReadyReport
	TaskCandidateBlocker     = core.TaskCandidateBlocker
	TaskCandidate            = core.TaskCandidate
	TaskCandidatesReport     = core.TaskCandidatesReport
	ReportLink               = core.ReportLink
	ReportDocument           = core.ReportDocument
	ReportRoadmapStage       = core.ReportRoadmapStage
	ReportRisk               = core.ReportRisk
	ReportProject            = core.ReportProject
	ReportKnowledge          = core.ReportKnowledge
	ReportScreen             = core.ReportScreen
	ProjectReport            = core.ProjectReport
	TaskContextDocument      = core.TaskContextDocument
	TaskContextSection       = core.TaskContextSection
	TaskContextReport        = core.TaskContextReport
	ChangeSetReport          = core.ChangeSetReport
	ChangeRepository         = core.ChangeRepository
	ChangeComparison         = core.ChangeComparison
	ChangeSide               = core.ChangeSide
	ChangeSummary            = core.ChangeSummary
	ChangeFileSummary        = core.ChangeFileSummary
	ChangeLineStats          = core.ChangeLineStats
	ChangeGitState           = core.ChangeGitState
	ChangeEntity             = core.ChangeEntity
	DocumentationChange      = core.DocumentationChange
	MermaidBlockChange       = core.MermaidBlockChange
	AssetDiffMetadata        = core.AssetDiffMetadata
	AssetMetadata            = core.AssetMetadata
	ScreenDiffMetadata       = core.ScreenDiffMetadata
	ScreenNodeSnapshot       = core.ScreenNodeSnapshot
	ScreenTransitionSnapshot = core.ScreenTransitionSnapshot
	ScreenTransitionChange   = core.ScreenTransitionChange
	SourceDiffHunk           = core.SourceDiffHunk
	RenderedSectionChange    = core.RenderedSectionChange
	SemanticChange           = core.SemanticChange
	ChangeLocation           = core.ChangeLocation
	RelationChange           = core.RelationChange
	TaskImpactReport         = core.TaskImpactReport
	TaskImpactEntry          = core.TaskImpactEntry
)

func PrintHelp(w io.Writer) { core.PrintHelp(w) }

func ParseArguments(argv []string) (Options, bool, bool, error) { return core.ParseArguments(argv) }

func RunCLI(argv []string, stdout, stderr io.Writer) int { return core.RunCLI(argv, stdout, stderr) }

func Main() { core.Main() }

func ClassifyDocument(relativePath string) string { return core.ClassifyDocument(relativePath) }

func StatusFor(status string) StatusInfo { return core.StatusFor(status) }

func BuildDocumentationModel(options Options) (*Model, error) {
	return core.BuildDocumentationModel(options)
}

func BuildReport(model *Model) ProjectReport { return core.BuildReport(model) }

func GenerateSite(model *Model, options Options) (GenerateResult, error) {
	return core.GenerateSite(model, options)
}

func SearchDocumentation(model *Model, query string, limit int) (SearchReport, error) {
	return core.SearchDocumentation(model, query, limit)
}

func InitTask(options Options) (TaskInitReport, error) { return core.InitTask(options) }

func Scaffold(options Options) (ScaffoldReport, error) { return core.Scaffold(options) }

func MoveTask(model *Model, options Options, operation string) (TaskMoveReport, error) {
	return core.MoveTask(model, options, operation)
}

func BuildTaskContext(model *Model, taskID string) (TaskContextReport, error) {
	return core.BuildTaskContext(model, taskID)
}

func BuildTaskTree(model *Model, taskID string) (TaskTreeReport, error) {
	return core.BuildTaskTree(model, taskID)
}

func BuildTaskReady(model *Model, taskID string, strict bool) TaskReadyReport {
	return core.BuildTaskReady(model, taskID, strict)
}

func BuildTaskCandidates(model *Model, parentTaskID string, strict bool) (TaskCandidatesReport, error) {
	return core.BuildTaskCandidates(model, parentTaskID, strict)
}

func BuildDocumentationChanges(options Options) (*ChangeSetReport, error) {
	return core.BuildDocumentationChanges(options)
}
