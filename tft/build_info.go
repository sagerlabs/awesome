package tft

import "time"

var buildInfo = BuildInfo{
	Version:   "dev",
	GitCommit: "unknown",
	BuildTime: "unknown",
}

type BuildInfo struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	BuildTime string `json:"build_time"`
}

func SetBuildInfo(version, gitCommit, buildTime string) {
	if version != "" {
		buildInfo.Version = version
	}
	if gitCommit != "" {
		buildInfo.GitCommit = gitCommit
	}
	if buildTime != "" {
		buildInfo.BuildTime = buildTime
	}
}

func CurrentBuildInfo() BuildInfo {
	return buildInfo
}

func ServiceStartTime() time.Time {
	return serviceStartTime
}

var serviceStartTime = time.Now()
