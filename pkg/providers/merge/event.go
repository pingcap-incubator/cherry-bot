package merge

import (
	"context"
	"fmt"
	"time"

	"github.com/pingcap-incubator/cherry-bot/util"

	"github.com/google/go-github/v32/github"
	"github.com/pkg/errors"
)

const (
	autoMergeCommand = "/run-auto-merge"
	autoMergeAlias   = "/merge"
	noAccessComment  = "Sorry @%s, you don't have permission to trigger auto merge event on this branch."
)

func (m *merge) ProcessPullRequestEvent(event *github.PullRequestEvent) {
	if event.GetSender().GetLogin() == m.opr.Config.Github.Bot {
		return
	}
	if *event.Action == "labeled" && *event.Label.Name == m.cfg.CanMergeLabel {
		pr := event.GetPullRequest()
		login := event.GetSender().GetLogin()

		if !m.havePermission(login, pr) {
			return
		}
		m.processPREvent(event)
	}
}

func (m *merge) havePermission(username string, pr *github.PullRequest) bool {
	base := pr.GetBase().GetRef()
	if base == "master" {
		err := m.CanMergeToMaster(m.repo, pr.Labels, username)
		if err != nil {
			msg := fmt.Sprintf(noAccessComment, username)
			msg = fmt.Sprintf("%s %s", msg, err)
			util.Error(m.addGithubComment(pr, msg))
			return false
		} else {
			return true
		}
	}
	havePermission := m.ifInWhiteList(username)
	if !havePermission {
		msg := fmt.Sprintf(noAccessComment, username)
		util.Error(m.addGithubComment(pr, msg))
	}
	return havePermission
}

func (m *merge) ProcessIssueCommentEvent(event *github.IssueCommentEvent) {
	if event.GetAction() != "created" {
		return
	}

	if event.Comment.GetBody() == autoMergeCommand || event.Comment.GetBody() == autoMergeAlias {
		// command only for org members
		login := event.GetSender().GetLogin()
		if !m.opr.Member.IfMember(login) {
			return
		}

		pr, _, err := m.opr.Github.PullRequests.Get(context.Background(),
			m.owner, m.repo, event.Issue.GetNumber())
		if err != nil {
			util.Error(errors.Wrap(err, "issue comment get PR"))
			return
		}
		if !m.havePermission(login, pr) {
			return
		}
		util.Error(m.addCanMerge(pr))
		model := AutoMerge{
			PrID:      pr.GetNumber(),
			Owner:     m.owner,
			Repo:      m.repo,
			BaseRef:   pr.GetBase().GetRef(),
			Status:    false,
			CreatedAt: time.Now(),
		}
		if err := m.saveModel(&model); err != nil {
			util.Error(errors.Wrap(err, "merge process PR event"))
		} else {
			util.Error(m.queueComment(pr))
		}
	}
}
