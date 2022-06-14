package main

import (
	"errors"
	"flag"
	"github.com/opensourceways/community-robot-lib/gitlabclient"
	"os"

	"github.com/opensourceways/community-robot-lib/config"
	"github.com/opensourceways/community-robot-lib/logrusutil"
	liboptions "github.com/opensourceways/community-robot-lib/options"
	framework "github.com/opensourceways/community-robot-lib/robot-gitlab-framework"
	"github.com/opensourceways/community-robot-lib/secret"
	"github.com/sirupsen/logrus"
)

type options struct {
	service liboptions.ServiceOptions
	gitlab  liboptions.GitLabOptions
}

func (o *options) Validate() error {
	if err := o.service.Validate(); err != nil {
		return err
	}

	return o.gitlab.Validate()
}

func gatherOptions(fs *flag.FlagSet, args ...string) options {
	var o options

	o.gitlab.AddFlags(fs)
	o.service.AddFlags(fs)

	_ = fs.Parse(args)

	return o
}

func main() {
	logrusutil.ComponentInit(botName)

	o := gatherOptions(flag.NewFlagSet(os.Args[0], flag.ExitOnError), os.Args[1:]...)
	if err := o.Validate(); err != nil {
		logrus.WithError(err).Fatal("Invalid options")
	}

	secretAgent := new(secret.Agent)
	if err := secretAgent.Start([]string{o.gitlab.TokenPath}); err != nil {
		logrus.WithError(err).Fatal("Error starting secret agent.")
	}

	defer secretAgent.Stop()

	agent := config.NewConfigAgent(func() config.Config {
		return &configuration{}
	})
	if err := agent.Start(o.service.ConfigFile); err != nil {
		logrus.WithError(err).Errorf("start config: %s", o.service.ConfigFile)
		return
	}

	defer agent.Stop()

	c := gitlabclient.NewGitlabClient(secretAgent.GetTokenGenerator(o.gitlab.TokenPath), "https://source.openeuler.sh/api/v4")

	r := newRobot(c, func() (*configuration, error) {
		_, cfg := agent.GetConfig()
		if c, ok := cfg.(*configuration); ok {
			return c, nil
		}
		return nil, errors.New("can't convert to configuration")
	})

	framework.Run(r, o.service.Port, o.service.GracePeriod)
}
