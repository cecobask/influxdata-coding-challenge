/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"errors"
	"fmt"
	hiringv1alpha1 "github.com/cannonpalms/email-controller-template/api/v1alpha1"
	"github.com/cannonpalms/email-controller-template/pkg/fakeemail"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"math"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"time"
)

const (
	ConditionTypeSucceeded = "Succeeded"
	ConditionTypeBlocked   = "Blocked"
	ConditionTypeBounced   = "Bounced"
	ConditionTypeInvalid   = "Invalid"
)

// EmailRequestReconciler reconciles a EmailRequest object
type EmailRequestReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	EmailService *fakeemail.EmailService
}

//+kubebuilder:rbac:groups=hiring.influxdata.io,resources=emailrequests,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=hiring.influxdata.io,resources=emailrequests/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=hiring.influxdata.io,resources=emailrequests/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *EmailRequestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := ctrl.LoggerFrom(ctx)
	emailRequest := &hiringv1alpha1.EmailRequest{}
	if err := r.Client.Get(ctx, req.NamespacedName, emailRequest); err != nil {
		return ctrl.Result{}, err
	}
	succeeded := meta.IsStatusConditionTrue(emailRequest.Status.Conditions, ConditionTypeSucceeded)
	bounced := meta.IsStatusConditionTrue(emailRequest.Status.Conditions, ConditionTypeBounced)
	invalid := meta.IsStatusConditionTrue(emailRequest.Status.Conditions, ConditionTypeInvalid)
	if succeeded || bounced || invalid {
		return ctrl.Result{}, nil
	}
	emailIdentifier, err := r.EmailService.Send(fakeemail.Email{
		DestinationAddress: emailRequest.Spec.RecipientEmail,
		Subject:            "Hello",
		Body:               fmt.Sprintf("Hello, %s!", emailRequest.Spec.RecipientName),
	})
	if err != nil {
		result, retry := handleEmailServiceSendError(emailRequest, err)
		emailRequest.Status.Failures++
		if updateErr := r.Status().Update(ctx, emailRequest); updateErr != nil {
			return ctrl.Result{}, updateErr
		}
		if retry {
			logger.Error(err, "email blocked, will retry")
			return result, nil
		}
		return result, err
	}
	message := fmt.Sprintf("sent email with id %d to %s", emailIdentifier, emailRequest.Spec.RecipientEmail)
	meta.SetStatusCondition(&emailRequest.Status.Conditions, metav1.Condition{
		Type:    ConditionTypeSucceeded,
		Status:  metav1.ConditionTrue,
		Reason:  ConditionTypeSucceeded,
		Message: message,
	})
	logger.Info(message)
	return ctrl.Result{}, r.Status().Update(ctx, emailRequest)
}

// SetupWithManager sets up the controller with the Manager.
func (r *EmailRequestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&hiringv1alpha1.EmailRequest{}).
		Complete(r)
}

func handleEmailServiceSendError(emailRequest *hiringv1alpha1.EmailRequest, emailErr error) (ctrl.Result, bool) {
	var (
		result                 ctrl.Result
		retry                  bool
		errEmailBounced        *fakeemail.ErrEmailBounced
		errEmailBlocked        *fakeemail.ErrEmailBlocked
		errInvalidEmailAddress *fakeemail.ErrInvalidEmailAddress
	)
	if errors.As(emailErr, &errEmailBounced) {
		meta.SetStatusCondition(&emailRequest.Status.Conditions, metav1.Condition{
			Type:    ConditionTypeBounced,
			Status:  metav1.ConditionTrue,
			Reason:  strings.Split(reflect.TypeOf(errEmailBounced).String(), ".")[1],
			Message: errEmailBounced.Error(),
		})
	}
	if errors.As(emailErr, &errEmailBlocked) {
		meta.SetStatusCondition(&emailRequest.Status.Conditions, metav1.Condition{
			Type:    ConditionTypeBlocked,
			Status:  metav1.ConditionTrue,
			Reason:  strings.Split(reflect.TypeOf(errEmailBlocked).String(), ".")[1],
			Message: errEmailBlocked.Error(),
		})
		result.RequeueAfter = calculateBackoff(
			emailRequest.Spec.BaseDelay.Duration,
			emailRequest.Spec.MaxDelay.Duration,
			emailRequest.Status.Failures,
		)
		retry = true
	}
	if errors.As(emailErr, &errInvalidEmailAddress) {
		meta.SetStatusCondition(&emailRequest.Status.Conditions, metav1.Condition{
			Type:    ConditionTypeInvalid,
			Status:  metav1.ConditionTrue,
			Reason:  strings.Split(reflect.TypeOf(errInvalidEmailAddress).String(), ".")[1],
			Message: errInvalidEmailAddress.Error(),
		})
	}
	return result, retry
}

func calculateBackoff(baseDelay time.Duration, maxDelay time.Duration, failures int) time.Duration {
	backoff := float64(baseDelay.Nanoseconds()) * math.Pow(2, float64(failures))
	if backoff > math.MaxInt64 {
		return maxDelay
	}
	calculated := time.Duration(backoff)
	if calculated > maxDelay {
		return maxDelay
	}
	return calculated
}
