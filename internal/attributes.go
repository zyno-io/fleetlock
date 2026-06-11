package fleetlock

import (
	"context"
	"encoding/json"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// fleetlock Node and Lease attribute keys. Timestamps are RFC3339 UTC.
const (
	attributePrefix = "fleetlock.poseidon.coreos.com/"
	// annotationLastLockTime records when a Node last obtained a reboot lock.
	annotationLastLockTime = attributePrefix + "last-lock-time"
	// annotationLastUnlockTime records when a Node (or group Lease) last
	// released a reboot lock.
	annotationLastUnlockTime = attributePrefix + "last-unlock-time"
	// annotationState records a Node's current reboot lock state.
	annotationState = attributePrefix + "state"

	stateLocked   = "locked"
	stateUnlocked = "unlocked"
)

// annotateNode applies a strategic merge patch to a Node's annotations. It is
// best-effort: callers log errors rather than failing the request.
func (s *Server) annotateNode(ctx context.Context, nodeName string, annotations map[string]string) error {
	patch, err := json.Marshal(map[string]interface{}{
		"metadata": map[string]interface{}{
			"annotations": annotations,
		},
	})
	if err != nil {
		return err
	}
	_, err = s.kubeClient.CoreV1().Nodes().Patch(ctx, nodeName, types.StrategicMergePatchType, patch, metav1.PatchOptions{})
	return err
}

// nodeReadyCondition returns the Node's Ready condition, or nil if absent.
func nodeReadyCondition(node *v1.Node) *v1.NodeCondition {
	if node == nil {
		return nil
	}
	for i := range node.Status.Conditions {
		if node.Status.Conditions[i].Type == v1.NodeReady {
			return &node.Status.Conditions[i]
		}
	}
	return nil
}

// rebootedWithinLock reports whether the Node's Ready condition transitioned
// after lockTime, i.e. the node went NotReady and recovered during the lock
// window, indicating it actually rebooted. A zero lockTime (no recorded lock)
// or missing Ready condition yields false.
func rebootedWithinLock(node *v1.Node, lockTime time.Time) bool {
	if lockTime.IsZero() {
		return false
	}
	cond := nodeReadyCondition(node)
	if cond == nil {
		return false
	}
	return cond.LastTransitionTime.Time.After(lockTime)
}

// withinCooldown reports whether a new lock should be denied because the most
// recent unlock for the group happened less than cooldown ago. A non-positive
// cooldown or zero lastUnlock disables the gate.
func withinCooldown(lastUnlock, now time.Time, cooldown time.Duration) bool {
	if cooldown <= 0 || lastUnlock.IsZero() {
		return false
	}
	return now.Sub(lastUnlock) < cooldown
}

// timeAnnotation parses an RFC3339 timestamp from a Node annotation, returning
// the zero time if absent or malformed.
func timeAnnotation(node *v1.Node, key string) time.Time {
	if node == nil {
		return time.Time{}
	}
	v, ok := node.Annotations[key]
	if !ok {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return time.Time{}
	}
	return t
}
