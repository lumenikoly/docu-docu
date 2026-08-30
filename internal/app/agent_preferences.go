package toudocu

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
)

var (
	agentPreferenceValueRE = regexp.MustCompile(`^[A-Za-z0-9._-]{1,128}$`)
	agentModelValueRE      = regexp.MustCompile(`^[A-Za-z0-9._/-]{1,128}$`)
)

type AgentPreferenceStore struct{ Dir string }
type agentPreferenceFile struct {
	Version int                         `json:"version"`
	Roots   map[string]AgentPreferences `json:"roots"`
}

func NewAgentPreferenceStore() (AgentPreferenceStore, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return AgentPreferenceStore{}, err
	}
	return AgentPreferenceStore{Dir: filepath.Join(dir, "toudocu")}, nil
}

func (s AgentPreferenceStore) Load(root string) AgentPreferences {
	fallback := AgentPreferences{LaunchPreset: AgentLaunchDefault}
	data, err := os.ReadFile(filepath.Join(s.Dir, "agent-preferences.json"))
	if err != nil {
		return fallback
	}
	var file agentPreferenceFile
	if json.Unmarshal(data, &file) != nil || file.Version != 1 {
		return fallback
	}
	if saved, ok := file.Roots[agentRootKey(root)]; ok && validAgentPreferences(saved) {
		return saved
	}
	return fallback
}

func (s AgentPreferenceStore) Save(root string, preferences AgentPreferences) error {
	if !validAgentPreferences(preferences) {
		return errors.New("invalid agent preferences")
	}
	if err := os.MkdirAll(s.Dir, 0700); err != nil {
		return err
	}
	path := filepath.Join(s.Dir, "agent-preferences.json")
	file := agentPreferenceFile{Version: 1, Roots: map[string]AgentPreferences{}}
	if data, err := os.ReadFile(path); err == nil {
		if json.Unmarshal(data, &file) != nil || file.Version != 1 || file.Roots == nil {
			file = agentPreferenceFile{Version: 1, Roots: map[string]AgentPreferences{}}
		}
	}
	file.Roots[agentRootKey(root)] = preferences
	data, err := json.Marshal(file)
	if err != nil {
		return err
	}
	temp, err := os.CreateTemp(s.Dir, ".agent-preferences-*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer func() { _ = os.Remove(tempPath) }()
	if err = temp.Chmod(0600); err == nil {
		_, err = temp.Write(data)
	}
	if closeErr := temp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}

func validLaunchPreset(p AgentLaunchPreset) bool {
	return p == AgentLaunchDefault || p == AgentLaunchFullAccess
}
func validAgentPreferences(p AgentPreferences) bool {
	return validLaunchPreset(p.LaunchPreset) && (p.Model == "" || agentModelValueRE.MatchString(p.Model)) && (p.Effort == "" || agentPreferenceValueRE.MatchString(p.Effort))
}
func agentRootKey(root string) string {
	canonical, err := filepath.Abs(root)
	if err != nil {
		canonical = filepath.Clean(root)
	}
	if resolved, err := filepath.EvalSymlinks(canonical); err == nil {
		canonical = resolved
	}
	sum := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(sum[:])
}
