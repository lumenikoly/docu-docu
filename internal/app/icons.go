package toudocu

import "strings"

var iconPaths = map[string]string{
	"activity":     `<path d="M3 12h4l2-7 4 14 2-7h6"/>`,
	"archive":      `<path d="M3 6h18M5 6v14h14V6M9 10h6"/>`,
	"book":         `<path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20V4H6.5A2.5 2.5 0 0 0 4 6.5z"/><path d="M4 6.5v13"/>`,
	"box":          `<path d="m21 8-9 5-9-5 9-5z"/><path d="m3 8 9 5v9l9-5V8M12 13v9"/>`,
	"checkCircle":  `<circle cx="12" cy="12" r="9"/><path d="m8 12 2.5 2.5L16 9"/>`,
	"chevronDown":  `<path d="m7 10 5 5 5-5"/>`,
	"chevronRight": `<path d="m10 7 5 5-5 5"/>`,
	"clipboard":    `<rect width="14" height="16" x="5" y="5" rx="2"/><path d="M9 5V3h6v2M9 10h6M9 14h4"/>`,
	"code":         `<path d="m8 9-3 3 3 3M16 9l3 3-3 3M14 5l-4 14"/>`,
	"compass":      `<circle cx="12" cy="12" r="9"/><path d="m16 8-2.5 5.5L8 16l2.5-5.5z"/>`,
	"file":         `<path d="M6 2h8l4 4v16H6z"/><path d="M14 2v5h5"/>`,
	"fileEdit":     `<path d="M6 2h8l4 4v6M14 2v5h5"/><path d="m13 18 5-5 2 2-5 5-3 1z"/>`,
	"gitCompare":   `<path d="M8 3v12a3 3 0 1 0 3 3M16 21V9a3 3 0 1 0-3-3"/><circle cx="8" cy="18" r="2"/><circle cx="16" cy="6" r="2"/>`,
	"history":      `<path d="M3 12a9 9 0 1 0 3-6.7L3 8"/><path d="M3 3v5h5M12 7v5l3 2"/>`,
	"home":         `<path d="m3 11 9-8 9 8"/><path d="M5 10v11h14V10M9 21v-7h6v7"/>`,
	"layers":       `<path d="m12 2 9 5-9 5-9-5z"/><path d="m3 12 9 5 9-5M3 17l9 5 9-5"/>`,
	"lightbulb":    `<path d="M9 18h6M10 22h4M8.5 15.5A7 7 0 1 1 15.5 15.5L15 18H9z"/>`,
	"map":          `<path d="m3 6 6-3 6 3 6-3v15l-6 3-6-3-6 3zM9 3v15M15 6v15"/>`,
	"menu":         `<path d="M4 7h16M4 12h16M4 17h16"/>`,
	"milestone":    `<path d="M5 3v18M5 5h12l-2 4 2 4H5"/>`,
	"monitor":      `<rect width="18" height="13" x="3" y="4" rx="2"/><path d="M8 21h8M12 17v4"/>`,
	"network":      `<rect width="6" height="5" x="9" y="2" rx="1"/><rect width="6" height="5" x="3" y="17" rx="1"/><rect width="6" height="5" x="15" y="17" rx="1"/><path d="M12 7v5M6 17v-2h12v2"/>`,
	"print":        `<path d="M6 9V3h12v6M6 18H4a2 2 0 0 1-2-2v-5a2 2 0 0 1 2-2h16a2 2 0 0 1 2 2v5a2 2 0 0 1-2 2h-2"/><rect width="12" height="8" x="6" y="14"/>`,
	"route":        `<circle cx="6" cy="19" r="2"/><circle cx="18" cy="5" r="2"/><path d="M8 19h3a3 3 0 0 0 3-3V8a3 3 0 0 1 3-3"/>`,
	"shield":       `<path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10"/><path d="m9 12 2 2 4-4"/>`,
	"stickyNote":   `<path d="M5 3h14v12l-6 6H5z"/><path d="M13 21v-6h6M8 8h8M8 12h5"/>`,
	"triangle":     `<path d="M12 3 2 21h20z"/><path d="M12 9v5M12 18h.01"/>`,
	"userFlow":     `<circle cx="9" cy="7" r="3"/><path d="M4 20v-2a5 5 0 0 1 10 0v2M16 8h5M19 5l3 3-3 3"/>`,
	"workflow":     `<rect width="6" height="5" x="3" y="3" rx="1"/><rect width="6" height="5" x="15" y="16" rx="1"/><path d="M9 5.5h3a3 3 0 0 1 3 3v10"/>`,
}

func renderIcon(name, class string) string {
	path := iconPaths[name]
	if path == "" {
		path = iconPaths["file"]
	}
	classes := "td-icon"
	if strings.TrimSpace(class) != "" {
		classes += " " + class
	}
	return `<svg class="` + escapeAttr(classes) + `" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">` + path + `</svg>`
}
