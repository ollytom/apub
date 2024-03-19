package main

// NodeInfo represents...
type NodeInfo struct {
	Version           string       `json:"version"`
	Software          nodeSoftware `json:"software"`
	Protocols         []string     `json:"protocols"`
	Usage             nodeUsage    `json:"usage"`
	OpenRegistrations bool         `json:"openRegistration"`
}

type nodeSoftware struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type nodeUsage struct {
	Users         nodeUserCounts `json:"users"`
	LocalPosts    int            `json:"localPosts"`
	LocalComments int            `json:"localComments"`
}

type nodeUserCounts struct {
	Total          int `json:"total"`
	ActiveHalfYear int `json:"activeHalfyear"`
	ActiveMonth    int `json:"activetMonth"`
}

func NewNodeInfo(name, version string, users int, posts int) NodeInfo {
	return NodeInfo{
		Version:   "2.0",
		Software:  nodeSoftware{name, version},
		Protocols: []string{"activitypub"},
		Usage: nodeUsage{
			Users:      nodeUserCounts{users, users, users},
			LocalPosts: posts,
		},
		OpenRegistrations: false,
	}
}
