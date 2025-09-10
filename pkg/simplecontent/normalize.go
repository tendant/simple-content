package simplecontent

import "strings"

// NormalizeDerivationType lowercases a user-facing derivation type.
func NormalizeDerivationType(s string) string {
    return strings.ToLower(s)
}

// NormalizeVariant lowercases a specific derivation variant.
func NormalizeVariant(s string) DerivationVariant {
    return DerivationVariant(strings.ToLower(s))
}

// DerivationTypeFromVariant infers a derivation type from a variant by taking
// the prefix before the first underscore. If no underscore exists, the entire
// variant string is returned.
func DerivationTypeFromVariant(variant string) string {
    v := strings.ToLower(strings.TrimSpace(variant))
    if v == "" {
        return ""
    }
    if i := strings.IndexByte(v, '_'); i > 0 {
        return v[:i]
    }
    return v
}
