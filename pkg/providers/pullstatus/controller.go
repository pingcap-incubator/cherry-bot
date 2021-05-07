package pullstatus

import (
	"strings"
	"time"

	"github.com/pingcap-incubator/cherry-bot/util"

	"github.com/google/go-github/v32/github"
	"github.com/pkg/errors"
)

const (
	pollingInterval = 120 * time.Second
	durationUnit    = 24 * time.Hour
)

var memberAssociation = []string{"COLLABORATOR", "MEMBER", "OWNER"}

func (p *pullStatus) processPullStatusControl(pull *github.PullRequest) {
	for _, label := range pull.Labels {
		p.labelFilter(pull, label.GetName())
	}
}

func (p *pullStatus) labelFilter(pull *github.PullRequest, label string) {
	for _, control := range p.cfg.PullStatusControl {
		if control.Label == label {
			p.processLabel(pull, label)
		}
	}
}

func (p *pullStatus) processLabel(pull *github.PullRequest, label string) {
	model, err := p.getPullStatusControl(pull.GetNumber(), label)
	if err != nil {
		util.Error(errors.Wrap(err, "pull status process label"))
		return
	}
	if model.ID != 0 {
		return
	}
	if err := p.createPullStatusControl(pull.GetNumber(), label); err != nil {
		util.Error(errors.Wrap(err, "pull status process label"))
	}
}

func (p *pullStatus) matchCond(cond string, cb func(label string)) {
	for _, ctl := range p.cfg.PullStatusControl {
		condSlice := strings.Split(ctl.Cond, "&")
		for _, target := range condSlice {
			if cond == target {
				cb(ctl.Label)
			}
		}
	}
}

func (p *pullStatus) processSynchronize(pull *github.PullRequest) {
	p.matchCond("noCommit", func(label string) {
		if err := p.updatePull(pull.GetNumber(), label); err != nil {
			util.Error(errors.Wrap(err, "process synchronize"))
		}
	})
}

func (p *pullStatus) processClosed(pull *github.PullRequest) {
	if err := p.closePull(pull); err != nil {
		util.Error(errors.Wrap(err, "process closed"))
	}
}

func (p *pullStatus) processReopened(pull *github.PullRequest) {
	if err := p.openPull(pull); err != nil {
		util.Error(errors.Wrap(err, "process reopened"))
	}
}

func (p *pullStatus) processReviewSubmitted(pull *github.PullRequest) {
	p.matchCond("noReview", func(label string) {
		if err := p.updatePull(pull.GetNumber(), label); err != nil {
			util.Error(errors.Wrap(err, "process review submitted"))
		}
	})
}

func (p *pullStatus) processIssueComment(sender *github.User, comment *github.IssueComment, issue *github.Issue) {
	if comment.GetUser().GetLogin() == p.opr.Config.Github.Bot {
		return
	}
	for _, association := range memberAssociation {
		if comment.GetAuthorAssociation() == association {
			p.matchCond("noComment", func(label string) {
				if err := p.updatePull(issue.GetNumber(), label); err != nil {
					util.Error(errors.Wrap(err, "process comment created"))
				}
			})
		}
	}
}

// polling jobs
func (p *pullStatus) startPolling() {
	ticker := time.NewTicker(pollingInterval)
	go func() {
		for range ticker.C {
			checks, err := p.getPullStatusControls()
			if err != nil {
				util.Error(errors.Wrap(err, "pull status polling job"))
				continue
			}
			for _, check := range checks {
				p.checkStatus(check)
			}
		}
	}()
}

func (p *pullStatus) matchDuration(duration time.Duration, label string, cb func(events []string, duration int)) {
	for _, ctl := range p.cfg.PullStatusControl {
		for _, controlEvent := range ctl.Events {
			limitDuration := time.Duration(controlEvent.Duration) * durationUnit
			if duration > limitDuration && label == ctl.Label {
				cb(strings.Split(controlEvent.Events, "+"), controlEvent.Duration)
			}
		}
	}
}

func (p *pullStatus) checkStatus(check *PullStatusControl) {
	duration := time.Now().Sub(check.LastUpdate)
	p.matchDuration(duration, check.Label, func(events []string, duration int) {
		if model, err := p.getPullStatusCheck(check.PullID, check.Label, duration, check.LastUpdate); err != nil {
			util.Error(errors.Wrap(err, "pull status check status"))
			return
		} else if model.ID != 0 {
			return
		}
		eventsSuccess := false
		for _, event := range events {
			if err := p.checkEvent(check, event, check.LastUpdate); err != nil {
				util.Error(errors.Wrap(err, "pull status check status"))
			} else {
				eventsSuccess = true
			}
		}
		if eventsSuccess {
			if err := p.createPullStatusCheck(check.PullID, check.Label, duration, check.LastUpdate); err != nil {
				util.Error(errors.Wrap(err, "pull status check status"))
			}
		}
	})
}

func (p *pullStatus) checkEvent(check *PullStatusControl, event string, lastUpdate time.Time) error {
	pull, er := p.getPullRequest(check.PullID)
	if er != nil {
		return errors.Wrap(er, "check event")
	}
	if p.checkPullStatus(pull) {
		return nil
	}

	var err error = nil

	switch event {
	case "PingAuthor":
		{
			err = errors.Wrap(p.noticePingAuthor(pull), "check event")
		}
	case "ClosePull":
		{
			err = errors.Wrap(p.noticeClosePull(pull), "check event")
		}
	case "LabelOutdated":
		{
			err = errors.Wrap(p.noticeLabelOutdated(pull), "check event")
		}
	case "CommentOutdated":
		{
			err = errors.Wrap(p.noticeCommentOutdated(pull), "check event")
		}
	case "PingReviewer":
		{
			//err = errors.Wrap(p.noticePingReviewer(pull, lastUpdate), "check event")
			err = nil
		}
	case "SlackDirect":
		{
			err = errors.Wrap(p.noticeSlackDirect(pull), "check event")
		}
	case "SlackChannel":
		{
			err = errors.Wrap(p.noticeSlackChannel(pull), "check event")
		}
	case "RedPacket":
		{
			err = errors.Wrap(p.noticeRedPacket(pull), "check event")
		}
	case "AskForReviewer":
		{
			err = errors.Wrap(p.askForReviewer(pull), "check event")
		}
	}

	return errors.Wrap(err, "check event")
}

func (p *pullStatus) checkPullStatus(pull *github.PullRequest) bool {
	if pull.GetState() == "closed" {
		if err := p.closePull(pull); err != nil {
			util.Error(errors.Wrap(err, "process closed"))
		}
		return true
	}
	return false
}
