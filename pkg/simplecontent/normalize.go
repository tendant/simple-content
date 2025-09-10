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

// ExtractCategoryFromVariant derives a category from a variant name by taking
// the prefix before the first underscore. If no underscore exists, the entire
// variant is treated as the category.
func ExtractCategoryFromVariant(variant string) DerivationCategory {
    v := strings.ToLower(strings.TrimSpace(variant))
    if v == "" {
        return ""
    }
    if i := strings.IndexByte(v, '_'); i > 0 {
        return DerivationCategory(v[:i])
    }
    return DerivationCategory(v)
}
