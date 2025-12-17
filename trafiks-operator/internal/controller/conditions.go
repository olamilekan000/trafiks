package controller

import (
	"context"
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func SetCondition(
	conditions *[]metav1.Condition,
	conditionType string,
	status metav1.ConditionStatus,
	reason, message string,
	observedGeneration int64,
) {
	if conditions == nil {
		return
	}

	newCondition := metav1.Condition{
		Type:               conditionType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: observedGeneration,
	}

	meta.SetStatusCondition(conditions, newCondition)
}

// updateStatusWithRetry updates the status of a Kubernetes resource with retry logic to handle conflicts
func updateStatusWithRetry(ctx context.Context, c client.Client, nn types.NamespacedName, obj client.Object, updateFn func(client.Object)) error {
	const maxRetries = 3
	for i := 0; i < maxRetries; i++ {
		if err := c.Get(ctx, nn, obj); err != nil {
			return err
		}

		updateFn(obj)

		if err := c.Status().Update(ctx, obj); err != nil {
			if apierrors.IsConflict(err) && i < maxRetries-1 {
				time.Sleep(time.Duration(i+1) * 50 * time.Millisecond)
				continue
			}
			return err
		}

		return nil
	}

	return fmt.Errorf("failed to update status after %d retries", maxRetries)
}
