package cherry

import (
	"context"
	"regexp"
	"strings"

	"github.com/pingcap-incubator/cherry-bot/util"

	"github.com/google/go-github/v32/github"
	"github.com/pkg/errors"
)

const (
	cherryPickInvite  = "/cherry-pick-invite"
	cherryPickTrigger = "/run-cherry-picker"
)

func (cherry *cherry) ProcessPullRequest(pr *github.PullRequest) {
	// status update
	match, err := regexp.MatchString(`\(#[0-9]+\)$`, *pr.Title)
	if err != nil {
		util.Error(errors.Wrap(err, "process cherry pick"))
	} else {
		if match {
			util.Error(cherry.createCherryPick(pr))
		} else {
			for _, label := range pr.Labels {
				if strings.HasPrefix(*label.Name, "LGT") {
					continue
				}
				util.Error(cherry.commitLabel(pr, *label.Name))
			}
			if pr.MergedAt != nil {
				util.Error(cherry.commitMerge(pr))
			}
		}
	}
}

func (cherry *cherry) ProcessPullRequestEvent(event *github.PullRequestEvent) {
	var err error

	switch *event.Action {
	case "labeled":
		{
			err = cherry.commitLabel(event.GetPullRequest(), *event.Label.Name)
		}
	case "unlabeled":
		{
			{
				err = cherry.removeLabel(event.GetPullRequest(), event.GetLabel().GetName())
			}
		}
	case "closed":
		{
			util.Printf("process cp closed event %s/%s#%d", cherry.owner, cherry.repo, event.GetPullRequest().GetNumber())
			err = cherry.commitMerge(event.PullRequest)
		}
	}

	if err != nil {
		util.Error(errors.Wrap(err, "cherry picker process pull request event"))
	}
}

func (cherry *cherry) ProcessIssueCommentEvent(event *github.IssueCommentEvent) {
	cmd := ""
	for _, line := range strings.Split(event.GetComment().GetBody(), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			cmd = line
			break
		}
	}

	if cmd == cherryPickTrigger {
		cherry.ProcessCherryPick(event)
	}
	if cmd == cherryPickInvite {
		cherry.ProcessInvite(event)
	}
}
func (cherry *cherry) ProcessCherryPick(event *github.IssueCommentEvent) {
	var (
		login  = event.GetSender().GetLogin()
		number = event.GetIssue().GetNumber()
	)
	if cherry.opr.Member.IfMember(login) || event.GetIssue().GetUser().GetLogin() == event.GetComment().GetUser().GetLogin() {
		pr, _, err := cherry.opr.Github.PullRequests.Get(context.Background(),
			cherry.owner, cherry.repo, number)
		if err != nil {
			util.Error(errors.Wrap(err, "issue comment get PR"))
			return
		}
		if pr.MergedAt == nil {
			return
		}
		for _, label := range pr.Labels {
			target, version, err := cherry.getTarget(label.GetName())
			util.Println("label is", label.GetName())
			if err == nil {
				util.Println("ready to cherry pick via command", target, version)
				if err := cherry.cherryPick(pr, target, version, false); err != nil {
					util.Error(errors.Wrap(err, "commit label"))
				}
			}
		}
	} else {
		util.Printf("%s/%s#%d %s don't have access to run %s", cherry.owner, cherry.repo, number, login, cherryPickTrigger)
	}
}

func (cherry *cherry) ProcessInvite(event *github.IssueCommentEvent) {
	var (
		login  = event.GetSender().GetLogin()
		number = event.GetIssue().GetNumber()
	)

	pull, _, err := cherry.opr.Github.PullRequests.Get(context.Background(),
		cherry.owner, cherry.repo, number)
	if err != nil {
		util.Error(err)
		return
	}

	if !cherry.opr.Member.IfMember(login) {
		if _, _, err := cherry.opr.Github.Issues.CreateComment(context.Background(),
			cherry.owner, cherry.repo, number, &github.IssueComment{
				Body: github.String("This command can used by organization's member only."),
			}); err != nil {
			util.Error(err)
		}
		return
	}

	util.Error(cherry.inviteIfNotCollaborator(login, pull))
}
