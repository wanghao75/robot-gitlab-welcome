package main

import (
	"github.com/xanzy/go-gitlab"
	"strings"
)

func (bot *robot) getSigOfRepo(org, repo string, pid int, cfg *botConfig) (string, error) {
	sigName, err := bot.findSigName(org, repo, pid, cfg, true)
	if err != nil {
		return sigName, err
	}

	return sigName, nil
}

func (bot *robot) listAllFilesOfRepo(pid int, cfg *botConfig) (map[string]string, error) {
	recursive := true
	opt := gitlab.ListTreeOptions{Ref: &cfg.Branch, Recursive: &recursive, Path: &cfg.Path}
	trees, err := bot.cli.GetDirectoryTree(pid, opt)
	if err != nil || len(trees) == 0 {
		return nil, err
	}

	r := make(map[string]string)
	count := 4

	for i := range trees {
		item := trees[i]
		if strings.Count(item.Path, "/") == count {
			r[item.Path] = strings.Split(item.Path, "/")[1]
		}
	}

	return r, nil
}

func (bot *robot) findSigName(org, repo string, pid int, cfg *botConfig, needRefreshTree bool) (sigName string, err error) {
	if len(cfg.reposSig) == 0 {
		files, err := bot.listAllFilesOfRepo(pid, cfg)
		if err != nil {
			return "", err
		}

		cfg.reposSig = files
	}

	for i := range cfg.reposSig {
		//if strings.Split(i, "/")[2] == org && strings.Split(strings.Split(i, "/")[4], ".yaml")[0] == repo {
		if strings.Split(i, "/")[2] == "openeuler" && strings.Split(strings.Split(i, "/")[4], ".yaml")[0] == "community" {
			sigName = cfg.reposSig[i]
			needRefreshTree = false

			break
		}
	}

	if needRefreshTree {
		files, err := bot.listAllFilesOfRepo(pid, cfg)
		if err != nil {
			return "", err
		}

		cfg.reposSig = files

		sigName = bot.fillData(cfg.reposSig, org, repo)
	}

	return sigName, nil
}

func (bot *robot) fillData(reposSig map[string]string, org, repo string) (sigName string) {
	for i := range reposSig {
		//if strings.Split(i, "/")[2] == org && strings.Split(strings.Split(i, "/")[4], ".yaml")[0] == repo {
		if strings.Split(i, "/")[2] == "openeuler" && strings.Split(strings.Split(i, "/")[4], ".yaml")[0] == "community" {
			sigName = reposSig[i]

			break
		}
	}

	return sigName
}
