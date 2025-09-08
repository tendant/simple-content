package simplecontent

import "strings"

// NormalizeCategory lowercases a user-facing derivation category.
func NormalizeCategory(s string) DerivationCategory {
    return DerivationCategory(strings.ToLower(s))
}

// NormalizeVariant lowercases a specific derivation variant.
func NormalizeVariant(s string) DerivationVariant {
    return DerivationVariant(strings.ToLower(s))
}

