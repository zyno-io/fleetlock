package fleetlock

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// nodeWithReady builds a Node whose Ready condition last transitioned at the
// given time.
func nodeWithReady(transition time.Time) *v1.Node {
	return &v1.Node{
		Status: v1.NodeStatus{
			Conditions: []v1.NodeCondition{
				{Type: v1.NodeMemoryPressure, Status: v1.ConditionFalse},
				{
					Type:               v1.NodeReady,
					Status:             v1.ConditionTrue,
					LastTransitionTime: metav1.NewTime(transition),
				},
			},
		},
	}
}

func TestRebootedWithinLock(t *testing.T) {
	lock := time.Date(2026, 6, 10, 3, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		node     *v1.Node
		lockTime time.Time
		want     bool
	}{
		{
			name:     "ready transitioned after lock (rebooted)",
			node:     nodeWithReady(lock.Add(2 * time.Minute)),
			lockTime: lock,
			want:     true,
		},
		{
			name:     "ready transitioned before lock (no reboot)",
			node:     nodeWithReady(lock.Add(-2 * time.Minute)),
			lockTime: lock,
			want:     false,
		},
		{
			name:     "ready transition equals lock time (not after)",
			node:     nodeWithReady(lock),
			lockTime: lock,
			want:     false,
		},
		{
			name:     "zero lock time",
			node:     nodeWithReady(lock.Add(2 * time.Minute)),
			lockTime: time.Time{},
			want:     false,
		},
		{
			name:     "nil node",
			node:     nil,
			lockTime: lock,
			want:     false,
		},
		{
			name:     "node without ready condition",
			node:     &v1.Node{},
			lockTime: lock,
			want:     false,
		},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, rebootedWithinLock(tt.node, tt.lockTime), tt.name)
	}
}

func TestWithinCooldown(t *testing.T) {
	now := time.Date(2026, 6, 10, 3, 30, 0, 0, time.UTC)
	cooldown := 5 * time.Minute

	tests := []struct {
		name       string
		lastUnlock time.Time
		cooldown   time.Duration
		want       bool
	}{
		{
			name:       "unlocked 2m ago, within cooldown",
			lastUnlock: now.Add(-2 * time.Minute),
			cooldown:   cooldown,
			want:       true,
		},
		{
			name:       "unlocked 6m ago, past cooldown",
			lastUnlock: now.Add(-6 * time.Minute),
			cooldown:   cooldown,
			want:       false,
		},
		{
			name:       "unlocked exactly cooldown ago",
			lastUnlock: now.Add(-cooldown),
			cooldown:   cooldown,
			want:       false,
		},
		{
			name:       "never unlocked",
			lastUnlock: time.Time{},
			cooldown:   cooldown,
			want:       false,
		},
		{
			name:       "cooldown disabled",
			lastUnlock: now.Add(-1 * time.Minute),
			cooldown:   0,
			want:       false,
		},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, withinCooldown(tt.lastUnlock, now, tt.cooldown), tt.name)
	}
}

func TestTimeAnnotation(t *testing.T) {
	stamp := time.Date(2026, 6, 10, 3, 0, 0, 0, time.UTC)
	node := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				annotationLastLockTime: stamp.Format(time.RFC3339),
				"bogus":                "not-a-time",
			},
		},
	}

	assert.True(t, timeAnnotation(node, annotationLastLockTime).Equal(stamp), "valid annotation")
	assert.True(t, timeAnnotation(node, "bogus").IsZero(), "malformed annotation")
	assert.True(t, timeAnnotation(node, annotationLastUnlockTime).IsZero(), "absent annotation")
	assert.True(t, timeAnnotation(nil, annotationLastLockTime).IsZero(), "nil node")
}
