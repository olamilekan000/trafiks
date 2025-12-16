package controller

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
