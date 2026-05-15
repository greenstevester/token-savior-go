package tools

import "strings"

// ParseProfile maps the TOKEN_SAVIOR_PROFILE env value to a ProfileSet.
// Unknown values fall back to ProfileFull; callers may log a warning.
func ParseProfile(value string) ProfileSet {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "full":
		return ProfileFull
	case "core":
		return ProfileCore
	case "nav":
		return ProfileNav
	case "lean":
		return ProfileLean
	case "ultra":
		return ProfileUltra
	case "tiny":
		return ProfileTiny
	case "tiny_plus":
		return ProfileTinyPlus
	default:
		return ProfileFull
	}
}

// VisibleTools returns the subset of schemas in registry whose Profiles
// include p. Used to build the tools/list payload.
func VisibleTools(r *Registry, p ProfileSet) []ToolSchema {
	out := make([]ToolSchema, 0)
	for _, s := range r.All() {
		if s.Profiles.Has(p) {
			out = append(out, s)
		}
	}
	return out
}
