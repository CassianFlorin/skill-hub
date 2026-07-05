package skill

import (
	"fmt"
	"strings"
)

// CompareSemver returns -1/0/1 when both versions parse as semver
// (optional "v" prefix, MAJOR.MINOR.PATCH, optional -prerelease).
// comparable is false when either version does not parse, in which
// case the caller should skip ordering checks.
func CompareSemver(left string, right string) (int, bool) {
	leftParts, leftPre, ok := parseSemver(left)
	if !ok {
		return 0, false
	}
	rightParts, rightPre, ok := parseSemver(right)
	if !ok {
		return 0, false
	}
	for i := 0; i < 3; i++ {
		if leftParts[i] != rightParts[i] {
			if leftParts[i] < rightParts[i] {
				return -1, true
			}
			return 1, true
		}
	}
	switch {
	case leftPre == rightPre:
		return 0, true
	case leftPre == "":
		return 1, true
	case rightPre == "":
		return -1, true
	case leftPre < rightPre:
		return -1, true
	default:
		return 1, true
	}
}

// ValidateConstraint checks that a requires constraint is well formed.
// A constraint is one or more comma-separated clauses, each an operator
// (>=, >, <=, <, =, or none meaning exact) followed by a semver version.
func ValidateConstraint(constraint string) error {
	clauses, err := parseConstraint(constraint)
	if err != nil {
		return err
	}
	if len(clauses) == 0 {
		return fmt.Errorf("empty version constraint")
	}
	return nil
}

// SatisfiesConstraint reports whether version meets the constraint.
// A version that does not parse as semver never satisfies a constraint.
func SatisfiesConstraint(version string, constraint string) (bool, error) {
	clauses, err := parseConstraint(constraint)
	if err != nil {
		return false, err
	}
	for _, clause := range clauses {
		cmp, comparable := CompareSemver(version, clause.version)
		if !comparable {
			return false, nil
		}
		ok := false
		switch clause.operator {
		case ">=":
			ok = cmp >= 0
		case ">":
			ok = cmp > 0
		case "<=":
			ok = cmp <= 0
		case "<":
			ok = cmp < 0
		case "=":
			ok = cmp == 0
		}
		if !ok {
			return false, nil
		}
	}
	return true, nil
}

type constraintClause struct {
	operator string
	version  string
}

func parseConstraint(constraint string) ([]constraintClause, error) {
	var clauses []constraintClause
	for _, part := range strings.Split(constraint, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		operator := "="
		for _, candidate := range []string{">=", "<=", ">", "<", "="} {
			if strings.HasPrefix(part, candidate) {
				operator = candidate
				part = strings.TrimSpace(strings.TrimPrefix(part, candidate))
				break
			}
		}
		if _, _, ok := parseSemver(part); !ok {
			return nil, fmt.Errorf("invalid version %q in constraint", part)
		}
		clauses = append(clauses, constraintClause{operator: operator, version: part})
	}
	return clauses, nil
}

func parseSemver(version string) ([3]int, string, bool) {
	version = strings.TrimPrefix(version, "v")
	version, prerelease, _ := strings.Cut(version, "-")
	version, _, _ = strings.Cut(version, "+")
	fields := strings.Split(version, ".")
	if len(fields) != 3 {
		return [3]int{}, "", false
	}
	var parts [3]int
	for i, field := range fields {
		if field == "" {
			return [3]int{}, "", false
		}
		value := 0
		for _, char := range field {
			if char < '0' || char > '9' {
				return [3]int{}, "", false
			}
			value = value*10 + int(char-'0')
		}
		parts[i] = value
	}
	return parts, prerelease, true
}
