// Package app — pure filter engine for hierarchy trees.
//
// Performs fuzzy matching on hierarchy node names while preserving ancestor
// paths so the tree context remains visible.
package app

import (
	"github.com/sahilm/fuzzy"
)

// FilterHierarchy returns a filtered copy of the hierarchy tree where nodes
// (at any depth) whose Name fuzzy-matches the query are retained along with
// all their ancestors.
//
// The returned tree is a new structure — the original nodes are never mutated.
// Returns nil when the query is empty (callers should interpret nil as "show
// all / unfiltered").
func FilterHierarchy(nodes []*HierarchyNode, query string) []*HierarchyNode {
	if query == "" || len(nodes) == 0 {
		return nil
	}

	return filterNodes(nodes, query)
}

// filterNodes recursively walks the tree and returns a pruned copy containing
// only nodes that match (or have matching descendants).
func filterNodes(nodes []*HierarchyNode, query string) []*HierarchyNode {
	var result []*HierarchyNode

	for _, n := range nodes {
		// Recurse into children first.
		filteredChildren := filterNodes(n.Children, query)

		// Check if this node itself matches.
		selfMatches := fuzzyMatchNode(n.Name, query)

		if selfMatches || len(filteredChildren) > 0 {
			// Shallow copy the node so we don't mutate the original.
			clone := *n
			if selfMatches {
				// When the node itself matches, include all its children
				// unchanged (deep copy) so the user can see the full
				// sub-tree under a matching node.
				clone.Children = deepCopyNodes(n.Children)
			} else {
				// Only an ancestor — include only the filtered subtree.
				clone.Children = filteredChildren
			}
			result = append(result, &clone)
		}
	}

	return result
}

// fuzzyMatchNode reports whether a node name fuzzy-matches the query.
func fuzzyMatchNode(name, query string) bool {
	matches := fuzzy.Find(query, []string{name})
	return len(matches) > 0
}

// deepCopyNodes creates a recursive deep copy of a node slice.
func deepCopyNodes(nodes []*HierarchyNode) []*HierarchyNode {
	if nodes == nil {
		return nil
	}
	result := make([]*HierarchyNode, len(nodes))
	for i, n := range nodes {
		clone := *n
		clone.Children = deepCopyNodes(n.Children)
		result[i] = &clone
	}
	return result
}
