package cherry

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pingcap-incubator/cherry-bot/util"

	"github.com/google/go-github/v32/github"
	"github.com/pkg/errors"
)

const day = 24 * time.Hour

var tag2val = map[string]int{
	"rc":    1,
	"alpha": 2,
	"beta":  3,
	"ga":    4,
	"":      10, // no tag means stable
}

func (cherry *cherry) commitLabel(pr *github.PullRequest, label string) error {
	model, err := cherry.getPullRequest(*pr.Number)
	if err != nil {
		return errors.Wrap(err, "commit label")
	}

	// if pr not exist
	if model.ID == 0 {
		util.Println("commit label, found model not exist", *pr.Number, model)
		return errors.New("pull request not exist when add label")
	}

	target, version, err := cherry.getTarget(label)
	if err != nil {
		// label parse failed, skip it
		return nil
	}

	// add label to database
	ifHasLabel, err := hasLabel(label, model.Label.String())
	if err != nil {
		return errors.Wrap(err, "commit label")
	}
	if !ifHasLabel {
		if err := cherry.addLabel(model, label); err != nil {
			return errors.Wrap(err, "commit label")
		}
	}

	// unmerged issue will be cherry picked later when it merged
	if model.Merge {
		if err := cherry.cherryPick(pr, target, version, true); err != nil {
			return errors.Wrap(err, "commit label")
		}
	}
	return nil
}

func (cherry *cherry) commitMerge(pr *github.PullRequest) error {
	model, err := cherry.getPullRequest(pr.GetNumber())
	if err != nil {
		return errors.Wrap(err, "commit merge")
	}

	merge := false
	if pr.MergedAt != nil {
		merge = true
	}
	if model.Merge != merge {
		model.Merge = merge
		if err := cherry.saveModel(model); err != nil {
			return errors.Wrap(err, "commit merge")
		}
		if merge {
			// commit pr
			// labels := labelSlice{}
			// if err := json.Unmarshal([]byte(model.Label.String()), &labels); err != nil {
			// 	return errors.Wrap(err, "commit merge")
			// }
			for _, l := range model.Label {
				target, version, err := cherry.getTarget(l)
				fmt.Printf("cp %s/%s#%d to %s version %s\n", cherry.owner, cherry.repo, pr.GetNumber(), target, version)
				if err != nil {
					util.Error(errors.Wrap(err, "commit merge"))
				} else if version != "" {
					if err := cherry.cherryPick(pr, target, version, true); err != nil {
						util.Error(errors.Wrap(err, "commit merge"))
					}
				}
			}
		}
	}
	return nil
}

func (cherry *cherry) cherryPick(pr *github.PullRequest, target string, version string, byLabel bool) error {
	if !cherry.ready {
		util.Printf("%s/%s#%d cherry pick not ready", cherry.owner, cherry.repo, pr.GetNumber())
		return nil
	}
	// cherry pick from a branch to itself
	if pr.GetBase().GetRef() == target {
		return nil
	}
	// hotfix for the empty target
	// TODO: find out the real problem
	if target == "" {
		return nil
	}
	// if pr.MergedAt != nil {
	// 	util.Println("try creating PR, ID:", *pr.Number, ", merged at", *pr.MergedAt)
	// }
	// if byLabel && pr.MergedAt != nil && pr.MergedAt.Before(time.Now().Add(-2*day)) {
	if byLabel && pr.MergedAt == nil {
		return nil
	}

	util.Printf("%s/%s#%d cherry pick locked", cherry.owner, cherry.repo, pr.GetNumber())
	cherry.Lock()
	defer cherry.Unlock()

	title := fmt.Sprintf("%s (#%d)", pr.GetTitle(), pr.GetNumber())
	from := pr.GetNumber()

	model, err := cherry.getCherryPick(from, target)
	if err != nil {
		return errors.Wrap(err, "commit cherry pick")
	}

	if model.FromPr != 0 {
		if model.Success {
			// cherry pick pull request already created
			return nil
		}
		if byLabel && model.TryTime >= maxRetryTime {
			// fail over max try times, cancel pull request plan
			return nil
			// return errors.New(fmt.Sprintf("create cherry pick PR failed over %d times", model.TryTime))
		}
	}

	newPr, prepareMessage, err := cherry.prepareCherryPick(pr, target)
	if err != nil {
		model.Owner = cherry.owner
		model.Repo = cherry.repo
		model.FromPr = from
		model.Base = target
		model.TryTime = model.TryTime + 1
		model.Success = false
		cherry.saveModel(model)
		// util.Error(cherry.prNotice(false, pr, target, 0, from, prepareMessage))
		util.Error(cherry.prNotice(false, target, pr, nil, "fail", prepareMessage))
		util.Error(cherry.addGithubReadyComment(pr, false, target, 0))
		return errors.Wrap(err, "commit cherry pick")
	}
	resPr, tryTime, err := cherry.submitCherryPick(newPr)

	success := false
	if resPr == nil && err == nil {
		// pr already exist
		return nil
	} else if err != nil {
		// pr create failed
		// util.Error(cherry.prNotice(false, pr, target, 0, from, "submit PR failed"))
		util.Error(cherry.prNotice(false, target, pr, nil, "fail", "submit PR failed"))
		util.Error(cherry.addGithubReadyComment(pr, false, target, 0))
		// util.Error(cherry.opr.Slack.FailPR(cherry.cfg.CherryPickChannel,
		// 	cherry.owner, cherry.repo, *pr.Head.Label, target, *pr.Number))
		return errors.Wrap(err, "commit cherry pick")
	} else {
		success = true
	}

	if success {
		err = cherry.patchCherryBody(pr, resPr)
		if err != nil {
			return errors.Wrap(err, "edit cherry pick")
		}
	}

	model.PrID = *resPr.Number
	model.FromPr = from
	model.Owner = cherry.owner
	model.Repo = cherry.repo
	model.Title = title
	model.Head = *pr.Head.Label
	model.Base = target
	model.Body = *resPr.Body
	model.CreatedByBot = true
	model.Success = success
	model.TryTime = model.TryTime + tryTime

	err = cherry.saveModel(model)
	if err != nil {
		return errors.Wrap(err, "record cherry pick")
	}

	if success {
		if cherry.cfg.RunTestCommand != "" {
			util.Error(cherry.addGithubTestComment(resPr))
		}
		util.Error(cherry.addGithubReadyComment(pr, true, target, *resPr.Number))
		util.Error(cherry.addGithubLabel(resPr, pr, version))
		util.Error(cherry.addGithubRequestReviews(resPr, cherry.getReviewers(pr)))
		util.Error(cherry.replaceGithubLabel(pr, version))
		if cherry.cfg.CherryPickMilestone {
			util.Error(cherry.assignMilestone(resPr, version))
		}
		var inviteUserName string
		if cherry.cfg.CherryPickAssign {
			assignee, err := cherry.addAssignee(pr, resPr)
			util.Error(err)
			inviteUserName = assignee
		} else {
			inviteUserName = pr.GetUser().GetLogin()
		}
		if cherry.cfg.InviteCollaborator {
			util.Error(cherry.inviteIfNotCollaborator(inviteUserName, resPr))
		}
	}

	updateResPr, _, err := cherry.opr.Github.PullRequests.Get(context.Background(),
		cherry.owner, cherry.repo, resPr.GetNumber())
	if err != nil {
		updateResPr = resPr
		util.Error(err)
	}
	util.Error(cherry.prNotice(true, target, pr, updateResPr, "success", ""))

	return nil
}

func (cherry *cherry) addAssignee(oldPull *github.PullRequest, newPull *github.PullRequest) (string, error) {
	assignee := oldPull.GetUser()
	if !cherry.opr.Member.IfMember(assignee.GetLogin()) {
		reviews, _, err := cherry.opr.Github.PullRequests.ListReviews(context.Background(), cherry.owner, cherry.repo, oldPull.GetNumber(), &github.ListOptions{PerPage: 100})
		if err != nil {
			return "", errors.Wrap(err, "assign reviewer, get reviews failed")
		}
		submitAt := time.Time{}
		for _, review := range reviews {
			if review.GetSubmittedAt().After(submitAt) {
				submitAt = review.GetSubmittedAt()
				assignee = review.GetUser()
			}
		}
	}

	_, _, err := cherry.opr.Github.Issues.AddAssignees(context.Background(), cherry.owner, cherry.repo, newPull.GetNumber(), []string{assignee.GetLogin()})
	return assignee.GetLogin(), errors.Wrap(err, "assign reviewer, update pull request")
}

func (cherry *cherry) assignMilestone(newPull *github.PullRequest, version string) error {
	openedMilestones, err := cherry.getAllOpenedMilestones()
	if err != nil {
		return errors.Wrap(err, "assign milestone, get milestones")
	}
	matchedMilestone := findMatchMilestones(openedMilestones, version)
	if matchedMilestone == nil {
		return errors.New("assign milestone, milestone not found")
	}

	_, _, err = cherry.opr.Github.Issues.Edit(context.Background(), cherry.owner, cherry.repo, newPull.GetNumber(), &github.IssueRequest{
		Milestone: matchedMilestone.Number,
	})
	return errors.Wrap(err, "assign milestone, update pull request")
}

func findMatchMilestones(milestones []*github.Milestone, version string) *github.Milestone {
	r, err := regexp.Compile(fmt.Sprintf(`^v?%s\.(\d+)\-?(.*)$`, version))
	if err != nil {
		return nil
	}

	var (
		m          *github.Milestone
		subVersion = int(^uint(0) >> 1) // max int
		versionTag string
	)

	for _, milestone := range milestones {
		matches := r.FindStringSubmatch(strings.Trim(strings.ToLower(milestone.GetTitle()), " "))
		if len(matches) != 3 {
			continue
		}
		matchSubVersion, _ := strconv.Atoi(matches[1])
		matchVersionTag := matches[2]
		if matchSubVersion < subVersion {
			subVersion = matchSubVersion
			versionTag = matchVersionTag
			m = milestone
		}
		if matchSubVersion == subVersion && compareVersionTag(matchVersionTag, versionTag) {
			versionTag = matchVersionTag
			m = milestone
		}
	}

	return m
}

func compareVersionTag(tag1, tag2 string) bool {
	if tag1 == tag2 {
		return false
	}
	r := regexp.MustCompile(`^([a-z0-9]*)\.?(.*?)$`)
	tag1Matches := r.FindStringSubmatch(tag1)
	tag2Matches := r.FindStringSubmatch(tag2)

	if len(tag1Matches) == 0 && len(tag2Matches) == 0 {
		return false
	}

	if len(tag1Matches) != len(tag2Matches) {
		return len(tag1Matches) > len(tag2Matches)
	}

	if tag2val[tag1Matches[1]] != tag2val[tag2Matches[1]] {
		return tag2val[tag1Matches[1]] < tag2val[tag2Matches[1]]
	}

	tag1SubVersion, _ := strconv.Atoi(tag1Matches[2])
	tag2SubVersion, _ := strconv.Atoi(tag1Matches[2])
	return tag1SubVersion < tag2SubVersion
}
