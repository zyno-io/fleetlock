package fleetlock

import (
	"context"
	"fmt"
	"time"

	coordv1 "k8s.io/api/coordination/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	coordclient "k8s.io/client-go/kubernetes/typed/coordination/v1"
)

// RebootLease uses a Lease to hold a RebootLock.
type RebootLease struct {
	// name and metadata
	Meta metav1.ObjectMeta
	// wrapped coordination client
	Client coordclient.LeasesGetter
	// internal Lease
	lease *coordv1.Lease
}

// RebootLock represents a node wishing to reboot.
type RebootLock struct {
	Holder           string
	LeaseTransitions int32
	// LastUnlock is the time the group's lock was most recently released. It
	// is persisted as a Lease annotation and used to enforce a post-reboot
	// cooldown before the next lock is granted. Zero if never unlocked.
	LastUnlock time.Time
}

// Name returns the RebootLease namespace and name.
func (l *RebootLease) Name() string {
	return fmt.Sprintf("%s/%s", l.Meta.Namespace, l.Meta.Name)
}

// Get reads the RebootLock from the Lease or initializes a Lease.
func (l *RebootLease) Get(ctx context.Context) (*RebootLock, error) {
	var err error
	l.lease, err = l.Client.Leases(l.Meta.Namespace).Get(ctx, l.Meta.Name, metav1.GetOptions{})

	// create initial reboot lease
	if errors.IsNotFound(err) {
		// initial lock has no holder
		lock := &RebootLock{}
		l.lease, err = l.Client.Leases(l.Meta.Namespace).Create(ctx, &coordv1.Lease{
			ObjectMeta: metav1.ObjectMeta{
				Name:      l.Meta.Name,
				Namespace: l.Meta.Namespace,
			},
			Spec: rebootLockToLeaseSpec(lock),
		}, metav1.CreateOptions{})
		if err != nil {
			return nil, err
		}
	}
	if err != nil {
		return nil, err
	}

	// decode the LeaseSpec and annotations
	slot := leaseSpecToRebootLock(&l.lease.Spec)
	slot.LastUnlock = leaseLastUnlock(l.lease)
	return slot, nil
}

// Update tries to store the RebootLock into the the Lease.
func (l *RebootLease) Update(ctx context.Context, slot *RebootLock) error {
	l.lease.Spec = rebootLockToLeaseSpec(slot)
	if !slot.LastUnlock.IsZero() {
		if l.lease.Annotations == nil {
			l.lease.Annotations = map[string]string{}
		}
		l.lease.Annotations[annotationLastUnlockTime] = slot.LastUnlock.UTC().Format(time.RFC3339)
	}
	var err error
	l.lease, err = l.Client.Leases(l.Meta.Namespace).Update(ctx, l.lease, metav1.UpdateOptions{})
	return err
}

// leaseLastUnlock reads the group's last unlock time from a Lease annotation,
// returning the zero time if absent or malformed.
func leaseLastUnlock(lease *coordv1.Lease) time.Time {
	v, ok := lease.Annotations[annotationLastUnlockTime]
	if !ok {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return time.Time{}
	}
	return t
}

// rebootLockToLeaseSpec encodes a RebootLock into a LeaseSpec.
func rebootLockToLeaseSpec(slot *RebootLock) coordv1.LeaseSpec {
	return coordv1.LeaseSpec{
		HolderIdentity:   &slot.Holder,
		LeaseTransitions: &slot.LeaseTransitions,
	}
}

// leaseSpecToRebootLock decodes a LeaseSpec to a RebootLock
func leaseSpecToRebootLock(spec *coordv1.LeaseSpec) *RebootLock {
	slot := &RebootLock{}
	if spec.HolderIdentity != nil {
		slot.Holder = *spec.HolderIdentity
	}
	if spec.LeaseTransitions != nil {
		slot.LeaseTransitions = *spec.LeaseTransitions
	}
	return slot
}
