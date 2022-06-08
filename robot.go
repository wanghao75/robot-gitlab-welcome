package main

import (
	"fmt"
	"github.com/opensourceways/community-robot-lib/gitlabclient"
	framework "github.com/opensourceways/community-robot-lib/robot-gitlab-framework"
	"github.com/opensourceways/community-robot-lib/utils"
	"github.com/sirupsen/logrus"
	"github.com/xanzy/go-gitlab"
	"strings"

	"github.com/opensourceways/community-robot-lib/config"
)

const (
	botName        = "welcome"
	actionOpen     = "open"
	welcomeMessage = `
Hi ***%s***, welcome to the %s Community.
I'm the Bot here serving you. You can find the instructions on how to interact with me at **[Here](%s)**.
If you have any questions, please contact the SIG: [%s](https://gitee.com/openeuler/community/tree/master/sig/%s), and any of the maintainers: @%s`
)

type iClient interface {
	CreateMergeRequestComment(projectID interface{}, mrID int, comment string) error
	AddMergeRequestLabel(projectID interface{}, mrID int, labels gitlab.Labels) error
	GetProjectLabels(projectID interface{}) ([]*gitlab.Label, error)
	CreateProjectLabel(pid interface{}, label, color string) error
	GetDirectoryTree(projectID interface{}, opts gitlab.ListTreeOptions) ([]*gitlab.TreeNode, error)
	ListCollaborators(projectID interface{}) ([]*gitlab.ProjectMember, error)
	CreateIssueComment(projectID interface{}, issueID int, comment string) error
	AddIssueLabels(projectID interface{}, issueID int, labels gitlab.Labels) error
}

func newRobot(cli iClient) *robot {
	return &robot{cli: cli}
}

type robot struct {
	cli iClient
}

func (bot *robot) NewConfig() config.Config {
	return &configuration{}
}

func (bot *robot) RobotName() string {
	return botName
}

func (bot *robot) getConfig(cfg config.Config, org, repo string) (*botConfig, error) {
	c, ok := cfg.(*configuration)
	if !ok {
		return nil, fmt.Errorf("can't convert to configuration")
	}

	if bc := c.configFor(org, repo); bc != nil {
		return bc, nil
	}

	return nil, fmt.Errorf("no config for this repo: %s/%s", org, repo)
}

func (bot *robot) RegisterEventHandler(f framework.HandlerRegister) {
	f.RegisterMergeEventHandler(bot.handleMergeEvent)
	f.RegisterIssueHandler(bot.handleIssueEvent)
}

func (bot *robot) handleMergeEvent(e *gitlab.MergeEvent, c config.Config, log *logrus.Entry) error {
	if e.ObjectAttributes.Action != actionOpen {
		return nil
	}

	projectID := e.Project.ID
	mrNumber := gitlabclient.GetMRNumber(e)
	author := gitlabclient.GetMRAuthor(e)

	org, repo := gitlabclient.GetMROrgAndRepo(e)
	cfg, err := bot.getConfig(c, org, repo)
	if err != nil {
		return err
	}

	return bot.handle(
		org, repo, author, projectID, cfg, log,

		func(c string) error {
			return bot.cli.CreateMergeRequestComment(projectID, mrNumber, c)
		},

		func(label string) error {
			return bot.cli.AddMergeRequestLabel(projectID, mrNumber, gitlab.Labels{label})
		},
	)
}

func (bot *robot) handleIssueEvent(e *gitlab.IssueEvent, c config.Config, log *logrus.Entry) error {
	if e.ObjectAttributes.Action != actionOpen {
		return nil
	}
	org, repo := gitlabclient.GetIssueOrgAndRepo(e)
	projectID := e.Project.ID
	number := gitlabclient.GetIssueNumber(e)
	author := gitlabclient.GetIssueAuthor(e)
	cfg, err := bot.getConfig(c, org, repo)
	if err != nil {
		return err
	}

	return bot.handle(
		org, repo, author, projectID, cfg, log,

		func(c string) error {
			return bot.cli.CreateIssueComment(projectID, number, c)
		},

		func(label string) error {
			return bot.cli.AddIssueLabels(projectID, number, gitlab.Labels{label})
		},
	)
}

func (bot *robot) handle(
	org, repo, author string,
	projectID int,
	cfg *botConfig, log *logrus.Entry,
	addMsg, addLabel func(string) error,
) error {
	sigName, comment, err := bot.genComment(org, repo, author, projectID, cfg)
	if err != nil {
		return err
	}

	mErr := utils.NewMultiErrors()

	if err := addMsg(comment); err != nil {
		mErr.AddError(err)
	}

	label := fmt.Sprintf("sig/%s", sigName)

	if err := bot.createLabelIfNeed(projectID, label); err != nil {
		log.Errorf("create repo label:%s, err:%s", label, err.Error())
	}

	if err := addLabel(label); err != nil {
		mErr.AddError(err)
	}

	return mErr.Err()
}

func (bot robot) genComment(org, repo, author string, pid int, cfg *botConfig) (string, string, error) {
	sigName, err := bot.getSigOfRepo(org, repo, pid, cfg)
	if err != nil {
		return "", "", err
	}

	if sigName == "" {
		return "", "", fmt.Errorf("cant get sig name of repo: %s/%s", org, repo)
	}

	maintainers, err := bot.getMaintainers(pid)
	if err != nil {
		return "", "", err
	}

	return sigName, fmt.Sprintf(
		welcomeMessage, author, cfg.CommunityName, cfg.CommandLink,
		sigName, sigName, strings.Join(maintainers, " , @"),
	), nil
}

func (bot *robot) getMaintainers(pid int) ([]string, error) {
	v, err := bot.cli.ListCollaborators(pid)
	if err != nil {
		return nil, err
	}

	r := make([]string, 0, len(v))
	for i := range v {
		p := v[i]
		if p != nil && (p.AccessLevel == 30 || p.AccessLevel == 40 || p.AccessLevel == 50) {
			r = append(r, v[i].Username)
		}
	}
	return r, nil
}

func (bot *robot) createLabelIfNeed(pid int, label string) error {
	repoLabels, err := bot.cli.GetProjectLabels(pid)
	if err != nil {
		return err
	}

	for _, v := range repoLabels {
		if v.Name == label {
			return nil
		}
	}

	return bot.cli.CreateProjectLabel(pid, label, "")
}
