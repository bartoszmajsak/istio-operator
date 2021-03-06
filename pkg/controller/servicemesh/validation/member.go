package validation

import (
	"context"
	"fmt"
	"net/http"

	admissionv1 "k8s.io/api/admission/v1beta1"
	authorizationv1 "k8s.io/api/authorization/v1"

	maistrav1 "github.com/maistra/istio-operator/pkg/apis/maistra/v1"
	"github.com/maistra/istio-operator/pkg/controller/common"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	atypes "sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"
)

type memberValidator struct {
	client  client.Client
	decoder atypes.Decoder
}

var _ admission.Handler = (*memberValidator)(nil)
var _ inject.Client = (*memberValidator)(nil)
var _ inject.Decoder = (*memberValidator)(nil)

func (v *memberValidator) Handle(ctx context.Context, req atypes.Request) atypes.Response {
	logger := log.WithValues("Request.Namespace", req.AdmissionRequest.Namespace, "Request.Name", req.AdmissionRequest.Name).WithName("smm-validator")
	smm := &maistrav1.ServiceMeshMember{}

	err := v.decoder.Decode(req, smm)
	if err != nil {
		logger.Error(err, "error decoding admission request")
		return admission.ErrorResponse(http.StatusBadRequest, err)
	}

	// do we care about this object?
	if smm.ObjectMeta.DeletionTimestamp != nil {
		logger.Info("skipping deleted smm resource")
		return admission.ValidationResponse(true, "")
	}

	// verify name == default
	if common.MemberName != smm.Name {
		return admission.ErrorResponse(http.StatusBadRequest, fmt.Errorf("ServiceMeshMember must be named %q", common.MemberName))
	}

	if req.AdmissionRequest.Operation == admissionv1.Update {
		oldSmm := &maistrav1.ServiceMeshMember{}
		err := v.decoder.DecodeRaw(req.AdmissionRequest.OldObject, oldSmm)
		if err != nil {
			logger.Error(err, "error decoding old object in admission request")
			return admission.ErrorResponse(http.StatusBadRequest, err)
		}

		if smm.Spec.ControlPlaneRef.Name != oldSmm.Spec.ControlPlaneRef.Name ||
			smm.Spec.ControlPlaneRef.Namespace != oldSmm.Spec.ControlPlaneRef.Namespace {
			logger.Info("Client tried to mutate ServiceMeshMember.spec.controlPlaneRef")
			return admission.ErrorResponse(http.StatusBadRequest, fmt.Errorf("Mutation of .spec.controlPlaneRef isn't allowed"))
		}
	}

	sar := &authorizationv1.SubjectAccessReview{
		Spec: authorizationv1.SubjectAccessReviewSpec{
			User:   req.AdmissionRequest.UserInfo.Username,
			UID:    req.AdmissionRequest.UserInfo.UID,
			Extra:  convertUserInfoExtra(req.AdmissionRequest.UserInfo.Extra),
			Groups: req.AdmissionRequest.UserInfo.Groups,
			ResourceAttributes: &authorizationv1.ResourceAttributes{
				Verb:      "use",
				Group:     "maistra.io",
				Resource:  "servicemeshcontrolplanes",
				Name:      smm.Spec.ControlPlaneRef.Name,
				Namespace: smm.Spec.ControlPlaneRef.Namespace,
			},
		},
	}

	err = v.client.Create(ctx, sar)
	if err != nil {
		logger.Error(err, "error processing SubjectAccessReview")
		return admission.ErrorResponse(http.StatusInternalServerError, err)
	}
	if !sar.Status.Allowed || sar.Status.Denied {
		return admission.ErrorResponse(http.StatusForbidden, fmt.Errorf("user '%s' does not have permission to use ServiceMeshControlPlane %s/%s", req.AdmissionRequest.UserInfo.Username, smm.Spec.ControlPlaneRef.Namespace, smm.Spec.ControlPlaneRef.Name))
	}

	return admission.ValidationResponse(true, "")
}

// InjectClient injects the client.
func (v *memberValidator) InjectClient(c client.Client) error {
	v.client = c
	return nil
}

// InjectDecoder injects the decoder.
func (v *memberValidator) InjectDecoder(d atypes.Decoder) error {
	v.decoder = d
	return nil
}
