package cherry

import (
	"testing"

	"github.com/google/go-github/v32/github"
	"github.com/stretchr/testify/assert"
)

func TestFindMatchMilestone(t *testing.T) {
	var nimMilestone *github.Milestone
	milestone309 := &github.Milestone{
		Number: github.Int(1),
		Title:  github.String("v3.0.9"),
	}
	milestone30x := &github.Milestone{
		Number: github.Int(2),
		Title:  github.String("v3.0.10"),
	}
	milestone400rc1 := &github.Milestone{
		Number: github.Int(3),
		Title:  github.String("v4.0.0-rc1"),
	}
	milestone400rc2 := &github.Milestone{
		Number: github.Int(4),
		Title:  github.String("v4.0.0-rc2"),
	}
	milestone400ga := &github.Milestone{
		Number: github.Int(5),
		Title:  github.String("v4.0.0-ga"),
	}
	milestone400 := &github.Milestone{
		Number: github.Int(6),
		Title:  github.String("v4.0.0"),
	}
	milestone401 := &github.Milestone{
		Number: github.Int(7),
		Title:  github.String("v4.0.1"),
	}
	milestone500rc := &github.Milestone{
		Number: github.Int(8),
		Title:  github.String("v5.0.0-rc"),
	}
	milestone500alpha := &github.Milestone{
		Number: github.Int(8),
		Title:  github.String("v5.0.0-alpha"),
	}
	milestone500beta := &github.Milestone{
		Number: github.Int(8),
		Title:  github.String("v5.0.0-alpha"),
	}
	milestone500ga := &github.Milestone{
		Number: github.Int(8),
		Title:  github.String("v5.0.0-ga"),
	}

	v3milestones := []*github.Milestone{milestone309, milestone30x}
	v4rcmilestones := []*github.Milestone{milestone400rc1, milestone400rc2, milestone400ga, milestone400}
	v4gamilestones := []*github.Milestone{milestone400ga, milestone400}
	v4milestones := []*github.Milestone{milestone400, milestone401}
	v5milestones := []*github.Milestone{milestone500rc, milestone500alpha, milestone500beta, milestone500ga}

	assert.Equal(t, findMatchMilestones(v3milestones, "3.0"), milestone309, "find latest version")
	assert.Equal(t, findMatchMilestones(v3milestones, "4.0"), nimMilestone, "find latest version")
	assert.Equal(t, findMatchMilestones(v4rcmilestones, "4.0"), milestone400rc1, "find latest version")
	assert.Equal(t, findMatchMilestones(v4gamilestones, "4.0"), milestone400ga, "find latest version")
	assert.Equal(t, findMatchMilestones(v4milestones, "4.0"), milestone400, "find latest version")
	assert.Equal(t, findMatchMilestones(v5milestones, "5.0"), milestone500rc, "find latest version")
	assert.Equal(t, findMatchMilestones(v5milestones[1:], "5.0"), milestone500alpha, "find latest version")
	assert.Equal(t, findMatchMilestones(v5milestones[2:], "5.0"), milestone500beta, "find latest version")
	assert.Equal(t, findMatchMilestones(v5milestones[3:], "5.0"), milestone500ga, "find latest version")
}
