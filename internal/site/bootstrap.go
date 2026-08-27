package site

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/url"
	"path"
	"strings"
)

type Runtime string

const (
	RuntimeStatic Runtime = "static"
	RuntimeServe  Runtime = "serve"
)

type PageReference struct {
	Kind string `json:"kind"`
	ID   string `json:"id,omitempty"`
	Path string `json:"path"`
}

type PortalReference struct {
	AssetBase string `json:"assetBase"`
	DataBase  string `json:"dataBase"`
}

type UISettings struct {
	Locale       string `json:"locale"`
	Theme        string `json:"theme"`
	ColorScheme  string `json:"colorScheme"`
	Accent       string `json:"accent"`
	Density      string `json:"density"`
	ContentWidth string `json:"contentWidth"`
}

type Capabilities struct {
	Search        bool `json:"search"`
	Diagrams      bool `json:"diagrams"`
	Editor        bool `json:"editor"`
	Changes       bool `json:"changes"`
	Review        bool `json:"review"`
	Rebuild       bool `json:"rebuild"`
	TaskWorkspace bool `json:"taskWorkspace"`
	UpdateCheck   bool `json:"updateCheck"`
	AgentConsole  bool `json:"agentConsole"`
}

type Endpoints struct {
	Editor          string `json:"editor,omitempty"`
	EditorWorkspace string `json:"editorWorkspace,omitempty"`
	Changes         string `json:"changes,omitempty"`
	Review          string `json:"review,omitempty"`
	Rebuild         string `json:"rebuild,omitempty"`
	Version         string `json:"version,omitempty"`
	AgentConsole    string `json:"agentConsole,omitempty"`
}

type PageBootstrap struct {
	SchemaVersion int             `json:"schemaVersion"`
	Runtime       Runtime         `json:"runtime"`
	Page          PageReference   `json:"page"`
	Portal        PortalReference `json:"portal"`
	UI            UISettings      `json:"ui"`
	Capabilities  Capabilities    `json:"capabilities"`
	Endpoints     *Endpoints      `json:"endpoints,omitempty"`
}

var pageKinds = map[string]struct{}{
	"document": {}, "architecture": {}, "module": {}, "use-case": {},
	"flow": {}, "screen": {}, "standard": {}, "runbook": {}, "task": {},
}

func validateRelativeBase(value string) error {
	parsed, err := url.Parse(value)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || strings.HasPrefix(value, "/") || strings.Contains(value, "\\") {
		return fmt.Errorf("portal base must be relative: %q", value)
	}
	return nil
}

func (bootstrap PageBootstrap) Validate() error {
	if bootstrap.SchemaVersion != 1 {
		return fmt.Errorf("unsupported bootstrap schema version %d", bootstrap.SchemaVersion)
	}
	if bootstrap.Runtime != RuntimeStatic && bootstrap.Runtime != RuntimeServe {
		return fmt.Errorf("unsupported runtime %q", bootstrap.Runtime)
	}
	if _, ok := pageKinds[bootstrap.Page.Kind]; !ok {
		return fmt.Errorf("unsupported page kind %q", bootstrap.Page.Kind)
	}
	if bootstrap.Page.Path == "" || path.IsAbs(bootstrap.Page.Path) || strings.HasPrefix(path.Clean(bootstrap.Page.Path), "../") {
		return fmt.Errorf("page path must stay relative")
	}
	if err := validateRelativeBase(bootstrap.Portal.AssetBase); err != nil {
		return err
	}
	if err := validateRelativeBase(bootstrap.Portal.DataBase); err != nil {
		return err
	}
	if bootstrap.Runtime == RuntimeStatic && bootstrap.Endpoints != nil {
		return fmt.Errorf("static bootstrap must not contain endpoints")
	}
	if bootstrap.Endpoints != nil {
		for _, endpoint := range []string{bootstrap.Endpoints.Editor, bootstrap.Endpoints.EditorWorkspace, bootstrap.Endpoints.Changes, bootstrap.Endpoints.Review, bootstrap.Endpoints.Rebuild, bootstrap.Endpoints.Version, bootstrap.Endpoints.AgentConsole} {
			if endpoint != "" && (!strings.HasPrefix(endpoint, "/") || strings.HasPrefix(endpoint, "//")) {
				return fmt.Errorf("serve endpoint must be same-origin")
			}
		}
	}
	return nil
}

// MarshalBootstrap uses encoding/json's HTML escaping before the value enters
// a non-executable application/json script element.
func MarshalBootstrap(bootstrap PageBootstrap) (template.JS, error) {
	if err := bootstrap.Validate(); err != nil {
		return "", err
	}
	content, err := json.Marshal(bootstrap)
	if err != nil {
		return "", err
	}
	return template.JS(content), nil
}
