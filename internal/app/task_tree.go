package toudocu

import (
	"fmt"
	"io"
	"strings"
)

func hierarchyRef(item *WorkItem) TaskHierarchyRef {
	return TaskHierarchyRef{ID: item.ID, Title: item.Title, Status: string(item.statusName), HasBlocker: strings.TrimSpace(item.Blocker) != ""}
}

func taskHierarchy(model *Model, item *WorkItem) TaskHierarchy {
	byID := map[string]*WorkItem{}
	for index := range model.Knowledge.WorkItems {
		candidate := &model.Knowledge.WorkItems[index]
		byID[candidate.ID] = candidate
	}
	hierarchy := TaskHierarchy{Ancestors: []TaskHierarchyRef{}, Children: []TaskHierarchyRef{}}
	if parent := byID[taskParentID(item)]; parent != nil {
		ref := hierarchyRef(parent)
		hierarchy.Parent = &ref
	}
	seenAncestors := map[string]bool{}
	for current := byID[taskParentID(item)]; current != nil && !seenAncestors[current.ID]; current = byID[taskParentID(current)] {
		seenAncestors[current.ID] = true
		hierarchy.Ancestors = append([]TaskHierarchyRef{hierarchyRef(current)}, hierarchy.Ancestors...)
	}
	for _, id := range item.ChildIDs {
		if child := byID[id]; child != nil {
			hierarchy.Children = append(hierarchy.Children, hierarchyRef(child))
		}
	}
	seenDescendants := map[string]bool{}
	var count func(*WorkItem)
	count = func(current *WorkItem) {
		for _, id := range current.ChildIDs {
			child := byID[id]
			if child == nil || seenDescendants[id] {
				continue
			}
			seenDescendants[id] = true
			hierarchy.Descendants.Total++
			switch child.statusName {
			case "draft":
				hierarchy.Descendants.Draft++
			case "ready":
				hierarchy.Descendants.Ready++
			case "in-progress":
				hierarchy.Descendants.InProgress++
			case "blocked":
				hierarchy.Descendants.Blocked++
			case "done":
				hierarchy.Descendants.Done++
			case "cancelled":
				hierarchy.Descendants.Cancelled++
			}
			count(child)
		}
	}
	count(item)
	return hierarchy
}

func BuildTaskTree(model *Model, taskID string) (TaskTreeReport, error) {
	item, err := findWorkItem(model, taskID)
	if err != nil {
		return TaskTreeReport{}, err
	}
	if !strings.HasPrefix(item.ID, "TASK-") {
		return TaskTreeReport{}, fmt.Errorf("task tree is available only for TASK-* work items")
	}
	return TaskTreeReport{SchemaVersion: 1, Kind: "task-tree", Generator: GeneratorInfo{Name: "Toudocu", Version: Version}, TaskID: taskID, Tree: taskTreeNode(model, item)}, nil
}

func taskTreeNode(model *Model, item *WorkItem) TaskTreeNode {
	byID := map[string]*WorkItem{}
	for index := range model.Knowledge.WorkItems {
		byID[model.Knowledge.WorkItems[index].ID] = &model.Knowledge.WorkItems[index]
	}
	seen := map[string]bool{}
	var node func(*WorkItem) TaskTreeNode
	node = func(current *WorkItem) TaskTreeNode {
		result := TaskTreeNode{ID: current.ID, Status: string(current.statusName), Title: current.Title, Children: []TaskTreeNode{}, statusLabel: current.Status.Label}
		if seen[current.ID] {
			return result
		}
		seen[current.ID] = true
		for _, id := range current.ChildIDs {
			if child := byID[id]; child != nil {
				result.Children = append(result.Children, node(child))
			}
		}
		return result
	}
	return node(item)
}

func printTaskTreeText(w io.Writer, report TaskTreeReport) {
	var printNode func(TaskTreeNode, string, bool, bool)
	printNode = func(node TaskTreeNode, prefix string, last, root bool) {
		branch := ""
		if !root {
			if last {
				branch = "└── "
			} else {
				branch = "├── "
			}
		}
		status := node.statusLabel
		if status == "" {
			status = node.Status
		}
		_, _ = fmt.Fprintf(w, "%s%s%s  %-11s  %s\n", prefix, branch, node.ID, status, node.Title)
		if !root {
			if last {
				prefix += "    "
			} else {
				prefix += "│   "
			}
		}
		for index, child := range node.Children {
			printNode(child, prefix, index == len(node.Children)-1, false)
		}
	}
	printNode(report.Tree, "", true, true)
}
