package contributor

import (
	"github.com/pingcap-incubator/cherry-bot/util"

	"github.com/google/go-github/v32/github"
	"github.com/pkg/errors"
)

func (c *Contributor) ProcessPullRequestEvent(event *github.PullRequestEvent) {
	var errs []error

	if *event.Action == "opened" || *event.Action == "reopened" {
		errs = c.processOpenedPR(event.PullRequest)
	}

	for _, err := range errs {
		util.Error(errors.Wrap(err, "cherry picker process pull request event"))
	}
}

func (c *Contributor) processOpenedPR(pull *github.PullRequest) (errs []error) {
	if pull.GetUser().GetLogin() == c.opr.Config.Github.Bot {
		return
	}
	authorType, err := c.authorType(pull)
	if err != nil {
		return []error{err}
	}
	switch pull.GetAuthorAssociation() {
	case "FIRST_TIME_CONTRIBUTOR", "FIRST_TIMER", "NONE":
		{
			//err = c.labelPull(pull, "first-time-contributor")
			err = nil
		}
	default:
		{
			if authorType == contributor {
				err = c.addContributorLabel(pull)
			} else {
				return errs
			}
		}
	}
	if err != nil {
		errs = []error{err}
	}

	if err := c.notifyNewContributorPR(pull); err != nil {
		errs = append(errs, err)
	}

	return
}

type authorType int

const (
	employee    authorType = 1
	reviewer    authorType = 2
	contributor authorType = 3
)

func (c *Contributor) authorType(pull *github.PullRequest) (authorType, error) {
	login := pull.GetUser().GetLogin()
	if c.opr.Member.IfMember(login) {
		isReviewer, err := c.isReviewer(login)
		if err != nil {
			return 0, errors.Wrap(err, "process opened PR")
		}
		if isReviewer {
			return reviewer, nil
		} else {
			// is a member and is not a reviewer -> employee
			return employee, nil
		}
	}
	return contributor, nil
}
