package htmx

import (
	"io/fs"
	"net/url"
	"sort"
	"strings"
)

type fileTreeNode struct {
	Name        string
	DisplayName string
	Path        string
	URL         string
	IsDir       bool
	Selected    bool
	Picker      *filePickerOption
	Children    []fileTreeNode
}

type filePickerView struct {
	Legend string
	Files  []fileTreeNode
}

type filePickerOptions struct {
	Legend         string
	InputName      string
	DisableCurrent bool
	Selected       string
}

type filePickerOption struct {
	InputName string
	Current   bool
	Disabled  bool
}

type folderBreadcrumbItem struct {
	Label   string
	URL     string
	Current bool
}

func makeFilePicker(fileNames []string, current string, options filePickerOptions) filePickerView {
	files := makeFileTree(fileNames, options.Selected)
	setFilePickerOptions(files, current, options)
	return filePickerView{Legend: options.Legend, Files: files}
}

func setFilePickerOptions(nodes []fileTreeNode, current string, options filePickerOptions) {
	for index := range nodes {
		if nodes[index].IsDir {
			setFilePickerOptions(nodes[index].Children, current, options)
			continue
		}
		isCurrent := nodes[index].Path == current
		nodes[index].Picker = &filePickerOption{
			InputName: options.InputName,
			Current:   isCurrent,
			Disabled:  options.DisableCurrent && isCurrent,
		}
		nodes[index].Selected = nodes[index].Selected && !nodes[index].Picker.Disabled
	}
}

func fileURL(name string) string {
	query := url.Values{}
	query.Set("file", name)
	return "/?" + query.Encode()
}

func folderURL(name string) string {
	if name == "" {
		return "/"
	}
	query := url.Values{}
	query.Set("folder", name)
	return "/?" + query.Encode()
}

func makeFileTree(fileNames []string, selected string) []fileTreeNode {
	var roots []fileTreeNode
	for _, fileName := range fileNames {
		parts := strings.Split(fileName, "/")
		children := &roots
		currentPath := ""
		for index, name := range parts {
			if currentPath == "" {
				currentPath = name
			} else {
				currentPath += "/" + name
			}
			isDir := index < len(parts)-1
			childIndex := -1
			for i := range *children {
				if (*children)[i].Name == name {
					childIndex = i
					break
				}
			}
			if childIndex == -1 {
				displayName := name
				if !isDir {
					displayName = strings.TrimSuffix(name, ".cue")
					if displayName == "" {
						displayName = name
					}
				}
				node := fileTreeNode{Name: name, DisplayName: displayName, Path: currentPath, IsDir: isDir}
				if isDir {
					node.URL = folderURL(currentPath)
				} else {
					node.URL = fileURL(currentPath)
					node.Selected = currentPath == selected
				}
				*children = append(*children, node)
				childIndex = len(*children) - 1
			}
			children = &(*children)[childIndex].Children
		}
	}
	sortFileTree(roots)
	return roots
}

func makeSidebarTree(fileNames []string, selected string) []fileTreeNode {
	var files []fileTreeNode
	for _, node := range makeFileTree(fileNames, selected) {
		if node.IsDir && node.Path == archiveFolder {
			continue
		}
		files = append(files, node)
	}
	return files
}

func folderContents(nodes []fileTreeNode, folder string) ([]fileTreeNode, bool) {
	if folder == "" {
		return nodes, true
	}
	if !fs.ValidPath(folder) {
		return nil, false
	}
	for _, node := range nodes {
		if !node.IsDir {
			continue
		}
		if node.Path == folder {
			return node.Children, true
		}
		if strings.HasPrefix(folder, node.Path+"/") {
			return folderContents(node.Children, folder)
		}
	}
	if folder == archiveFolder {
		return nil, true
	}
	return nil, false
}

func makeFolderBreadcrumb(folder string) []folderBreadcrumbItem {
	breadcrumbs := []folderBreadcrumbItem{{Label: "Files", URL: folderURL(""), Current: folder == ""}}
	if folder == "" {
		return breadcrumbs
	}
	parts := strings.Split(folder, "/")
	for index, part := range parts {
		breadcrumbs = append(breadcrumbs, folderBreadcrumbItem{
			Label:   part,
			URL:     folderURL(strings.Join(parts[:index+1], "/")),
			Current: index == len(parts)-1,
		})
	}
	return breadcrumbs
}

func sortFileTree(nodes []fileTreeNode) {
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].IsDir != nodes[j].IsDir {
			return nodes[i].IsDir
		}
		left, right := strings.ToLower(nodes[i].Name), strings.ToLower(nodes[j].Name)
		if left == right {
			return nodes[i].Name < nodes[j].Name
		}
		return left < right
	})
	for i := range nodes {
		sortFileTree(nodes[i].Children)
	}
}
